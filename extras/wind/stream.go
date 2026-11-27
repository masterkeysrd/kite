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

// Latest is a convenience reducer that replaces previous state with the latest event.
func Latest[T any](_ T, next T) T { return next }

// Stream configures a declarative streaming query for [UseStream].
type Stream[E, S any] struct {
	// Initial seeds the state before the first Fetch or Source emission.
	Initial S
	// Fetch optionally loads an authoritative snapshot on connect and reconnect
	// before Source begins streaming deltas.
	Fetch func(ctx context.Context) (S, error)
	// Source opens the event stream, receiving the current state (seeded from
	// Initial, cached RefetchAppend state, or Fetch) so cursors like LastEventID
	// can be forwarded on connect/reconnect.
	Source func(ctx context.Context, state S) iter.Seq2[E, error]
	// Reduce folds each incoming event into the accumulated state. When nil and E
	// is assignable to S, incoming events replace the state directly.
	Reduce func(state S, event E) S
	// Enabled controls whether the stream connects (defaults to true when unset).
	Enabled Optional[bool]
	// Throttle bounds UI notification frequency (defaults to 16ms when unset; <= 0 coalesces via microtask).
	Throttle Optional[time.Duration]
	// RefetchMode controls how cached state behaves across reconnects.
	RefetchMode RefetchMode
}

type StreamResult[T any] struct {
	Result[T]
	Status string
}

// UseStream subscribes to a declarative stream configured by spec, optionally
// loading an initial snapshot via Fetch, opening a state-aware Source iterator,
// and folding incoming events with Reduce.
func UseStream[K comparable, E, S any](key K, spec Stream[E, S]) StreamResult[S] {
	client := UseClient()
	if client == nil {
		panic("wind.UseStream must be used inside a wind.Provider")
	}

	enabled := spec.Enabled.UnwrapOr(true)
	throttle := spec.Throttle
	refetchMode := spec.RefetchMode

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
			initial:     spec.Initial,
			throttle:    throttle,
			refetchMode: refetchMode,
			run: func(ctx context.Context, prev any, hasPrev bool, mode RefetchMode, emit func(data any, err error) bool) {
				acc := spec.Initial
				if mode == RefetchAppend && hasPrev {
					if typed, ok := prev.(S); ok {
						acc = typed
					}
				}

				if spec.Fetch != nil {
					snap, err := spec.Fetch(ctx)
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						emit(acc, err)
						return
					}
					acc = snap
					if !emit(acc, nil) {
						return
					}
				}

				if spec.Source != nil {
					seq := spec.Source(ctx, acc)
					if seq != nil {
						seq(func(ev E, err error) bool {
							if err != nil {
								return emit(acc, err)
							}
							if spec.Reduce != nil {
								acc = spec.Reduce(acc, ev)
							} else if typed, ok := any(ev).(S); ok {
								acc = typed
							}
							return emit(acc, nil)
						})
					}
				}
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

				client.armGCTimerLocked(entry, key)
			}
			entry.mu.Unlock()
		}
	}, key)

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
	}, key, enabled)

	// Prepare Result state
	entry.mu.Lock()
	defer entry.mu.Unlock()
	renderedVersion = entry.version

	data := spec.Initial
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
