package wind

import (
	"context"
	"iter"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
)

// RefetchMode controls how cached stream state is treated when reconnecting.
type RefetchMode uint8

const (
	// RefetchReplace keeps cached Data visible while reconnecting, folds incoming
	// events into a fresh initial state, and swaps Data on the first yielded event.
	RefetchReplace RefetchMode = iota
	// RefetchReset clears cached Data back to initial immediately upon reconnect
	// and transitions status back to loading/connecting until the first event.
	RefetchReset
	// RefetchAppend folds incoming events on reconnect onto the existing cached Data
	// (if present, otherwise starting from initial).
	RefetchAppend
)

type StreamOptions struct {
	Enabled     Optional[bool]
	Throttle    Optional[time.Duration] // Unset defaults to 16ms; <= 0 disables timer throttling.
	RefetchMode RefetchMode
}

type StreamResult[T any] struct {
	Result[T]
	Status string
}

// UseStream subscribes to an iterator stream of T, emitting the latest yielded value.
func UseStream[K comparable, T any](
	key K,
	fetcher func(ctx context.Context, key K) iter.Seq2[T, error],
	opts ...StreamOptions,
) StreamResult[T] {
	var zero T
	return UseStreamReduce(key, zero, fetcher, func(_ T, item T) T {
		return item
	}, opts...)
}

// UseStreamReduce subscribes to an iterator stream of E and folds every yielded
// event into state S via reduce, ensuring throttled updates never drop events.
func UseStreamReduce[K comparable, E, S any](
	key K,
	initial S,
	fetcher func(ctx context.Context, key K) iter.Seq2[E, error],
	reduce func(prev S, event E) S,
	opts ...StreamOptions,
) StreamResult[S] {
	client := UseClient()
	if client == nil {
		panic("wind.UseStreamReduce must be used inside a wind.Provider")
	}

	enabled := true
	var throttle Optional[time.Duration]
	var refetchMode RefetchMode
	if len(opts) > 0 {
		enabled = opts[0].Enabled.UnwrapOr(true)
		throttle = opts[0].Throttle
		refetchMode = opts[0].RefetchMode
	}

	_, setTick := kitex.UseState[uint64](0)
	tick := func() {
		setTick(hookTickSeq.Add(1))
	}

	entry, closed := client.getOrCreateEntry(key)

	// Stop gcTimer if it was running and register refetch/reconnect implementation
	entry.mu.Lock()
	if entry.gcTimer != nil {
		entry.gcTimer.Stop()
		entry.gcTimer = nil
	}
	if !closed && !entry.closed {
		cfg := streamConfig{
			initial:     initial,
			throttle:    throttle,
			refetchMode: refetchMode,
			run: func(ctx context.Context, prev any, hasPrev bool, mode RefetchMode, emit func(data any, err error) bool) {
				acc := initial
				if mode == RefetchAppend && hasPrev {
					if typed, ok := prev.(S); ok {
						acc = typed
					}
				}
				seq := fetcher(ctx, key)
				seq(func(ev E, err error) bool {
					if err != nil {
						return emit(acc, err)
					}
					acc = reduce(acc, ev)
					return emit(acc, nil)
				})
			},
		}
		entry.refetch = func() {
			client.executeStream(entry, cfg)
		}
	}
	entry.mu.Unlock()

	var renderedVersion uint64

	// Passive subscriber lifecycle: manage notifications and cache eviction timer.
	kitex.UseLayoutEffectCleanup(func() func() {
		entry.mu.Lock()
		if entry.closed {
			entry.mu.Unlock()
			return nil
		}
		id := entry.nextSubID
		entry.nextSubID++
		entry.subscribers[id] = tick
		stale := entry.version != renderedVersion
		entry.mu.Unlock()

		if stale {
			tick()
		}

		return func() {
			entry.mu.Lock()
			delete(entry.subscribers, id)
			if len(entry.subscribers) == 0 {
				if entry.cancel != nil {
					entry.cancel()
					entry.cancel = nil
				}
				if entry.throttleTimer != nil {
					entry.throttleTimer.Stop()
					entry.throttleTimer = nil
				}
				entry.updatePending = false
				if entry.state.isFetching {
					entry.state.isFetching = false
					entry.version++
				}
				entry.refetch = nil

				if !entry.closed {
					gcTime := client.GcTime
					if gcTime == 0 {
						gcTime = 5 * time.Minute
					}
					if entry.gcTimer != nil {
						entry.gcTimer.Stop()
					}
					entry.gcTimer = time.AfterFunc(gcTime, func() {
						client.mu.Lock()
						if client.closed {
							client.mu.Unlock()
							return
						}
						entry.mu.Lock()
						if len(entry.subscribers) == 0 {
							delete(client.cache, key)
						}
						entry.mu.Unlock()
						client.mu.Unlock()
					})
				}
			}
			entry.mu.Unlock()
		}
	}, []any{key})

	// Active enabled consumer lifecycle: connect on 0->1, deduplicate on 1->2, cancel on 1->0.
	kitex.UseLayoutEffectCleanup(func() func() {
		if !enabled {
			return nil
		}
		entry.mu.Lock()
		if entry.closed {
			entry.mu.Unlock()
			return nil
		}
		eid := entry.nextSubID
		entry.nextSubID++
		wasZero := len(entry.enabledSubs) == 0
		entry.enabledSubs[eid] = struct{}{}
		refetch := entry.refetch
		entry.mu.Unlock()

		if wasZero && refetch != nil {
			refetch()
		}

		return func() {
			entry.mu.Lock()
			delete(entry.enabledSubs, eid)
			becameZero := len(entry.enabledSubs) == 0
			hasPassive := len(entry.subscribers) > 0
			if becameZero {
				if entry.cancel != nil {
					entry.cancel()
					entry.cancel = nil
				}
				if entry.throttleTimer != nil {
					entry.throttleTimer.Stop()
					entry.throttleTimer = nil
				}
				entry.updatePending = false
				if entry.state.isFetching {
					entry.state.isFetching = false
				}
				entry.version++
			}
			entry.mu.Unlock()

			if becameZero && hasPassive {
				entry.notifySubscribers()
			}
		}
	}, []any{key, enabled})

	// Prepare Result state
	entry.mu.Lock()
	defer entry.mu.Unlock()
	renderedVersion = entry.version

	data := initial
	if entry.state.hasData {
		if typed, ok := entry.state.data.(S); ok {
			data = typed
		}
	}

	var status string
	switch entry.state.status {
	case "loading":
		status = "connecting"
	case "success":
		if entry.state.isFetching {
			status = "open"
		} else {
			status = "closed"
		}
	case "error":
		status = "error"
	}

	return StreamResult[S]{
		Result: Result[S]{
			Data:       data,
			IsLoading:  entry.state.status == "loading",
			IsFetching: entry.state.isFetching,
			IsError:    entry.state.status == "error",
			Error:      entry.state.err,
			Refetch:    entry.refetch,
		},
		Status: status,
	}
}
