package wind

import (
	"context"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

type Result[T any] struct {
	Data       T
	IsLoading  bool
	IsFetching bool
	IsError    bool
	Error      error
	Refetch    func()
}

type Options struct {
	Enabled   bool
	StaleTime time.Duration
}

func Use[K comparable, T any](key K, fetcher func(context.Context, K) *promise.Promise[T], opts ...Options) Result[T] {
	client := UseClient()
	if client == nil {
		panic("wind.Use must be used inside a wind.Provider")
	}

	enabled := true
	staleTime := time.Duration(0)
	if len(opts) > 0 {
		enabled = opts[0].Enabled
		staleTime = opts[0].StaleTime
	}

	_, setTick := kitex.UseState(time.Now().UnixNano())
	tick := func() {
		setTick(time.Now().UnixNano())
	}

	// Retrieve or create query entry
	client.mu.Lock()
	entry, ok := client.cache[key]
	if !ok {
		entry = &queryEntry{
			key: key,
			state: queryState{
				status: "loading",
			},
			subscribers: make(map[int]func()),
		}
		client.cache[key] = entry
	}
	client.mu.Unlock()

	// Stop gcTimer if it was running
	entry.mu.Lock()
	if entry.gcTimer != nil {
		entry.gcTimer.Stop()
		entry.gcTimer = nil
	}
	entry.mu.Unlock()

	// Register refetch implementation
	entry.mu.Lock()
	typeEraser := func(ctx context.Context) *promise.Promise[any] {
		p := fetcher(ctx, key)
		return promise.New(func(ctx context.Context) (any, error) {
			return p.Await(ctx)
		})
	}
	entry.refetch = func() {
		client.executeFetch(entry, typeEraser)
	}
	entry.mu.Unlock()

	// Subscribe to changes
	kitex.UseLayoutEffectCleanup(func() func() {
		entry.mu.Lock()
		id := entry.nextSubID
		entry.nextSubID++
		entry.subscribers[id] = tick
		entry.mu.Unlock()

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
				entry.state.isFetching = false
				entry.refetch = nil

				// Start eviction timer
				gcTime := client.GcTime
				if gcTime == 0 {
					gcTime = 5 * time.Minute
				}
				if entry.gcTimer != nil {
					entry.gcTimer.Stop()
				}
				entry.gcTimer = time.AfterFunc(gcTime, func() {
					client.mu.Lock()
					entry.mu.Lock()
					if len(entry.subscribers) == 0 {
						delete(client.cache, key)
					}
					entry.mu.Unlock()
					client.mu.Unlock()
				})
			}
			entry.mu.Unlock()
		}
	}, []any{key})

	// Trigger fetch if enabled and stale/first load
	kitex.UseLayoutEffect(func() {
		if !enabled {
			return
		}
		entry.mu.Lock()
		needsFetch := !entry.state.isFetching && (entry.state.status == "loading" || time.Since(entry.state.updatedAt) > staleTime)
		entry.mu.Unlock()

		if needsFetch {
			entry.refetch()
		}
	}, []any{key, enabled, staleTime})

	// Prepare result
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var data T
	if entry.state.data != nil {
		data = entry.state.data.(T)
	}

	return Result[T]{
		Data:       data,
		IsLoading:  entry.state.status == "loading",
		IsFetching: entry.state.isFetching,
		IsError:    entry.state.status == "error",
		Error:      entry.state.err,
		Refetch:    entry.refetch,
	}
}

func (c *Client) executeFetch(entry *queryEntry, fetcher func(context.Context) *promise.Promise[any]) {
	entry.mu.Lock()
	if entry.state.isFetching {
		entry.mu.Unlock()
		return
	}

	entry.state.isFetching = true
	ctx, cancel := context.WithCancel(context.Background())
	entry.cancel = cancel
	entry.mu.Unlock()

	// Notify observers that fetching has started
	entry.notifySubscribers()

	fetcher(ctx).
		Then(func(data any) {
			entry.mu.Lock()
			if ctx.Err() != nil {
				entry.mu.Unlock()
				return
			}

			entry.state.isFetching = false
			entry.cancel = nil
			entry.state.data = data
			entry.state.err = nil
			entry.state.status = "success"
			entry.state.updatedAt = time.Now()

			shouldRefetch := entry.invalidatedDuringFetch
			entry.invalidatedDuringFetch = false
			entry.mu.Unlock()

			entry.notifySubscribers()

			if shouldRefetch && entry.refetch != nil {
				entry.refetch()
			}
		}, func(err error) {
			entry.mu.Lock()
			if ctx.Err() != nil {
				entry.mu.Unlock()
				return
			}

			entry.state.isFetching = false
			entry.cancel = nil
			entry.state.err = err
			entry.state.status = "error"

			shouldRefetch := entry.invalidatedDuringFetch
			entry.invalidatedDuringFetch = false
			entry.mu.Unlock()

			entry.notifySubscribers()

			if shouldRefetch && entry.refetch != nil {
				entry.refetch()
			}
		})
}
