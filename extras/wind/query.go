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
	Enabled   Optional[bool]
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
		enabled = opts[0].Enabled.UnwrapOr(true)
		staleTime = opts[0].StaleTime
	}

	_, setTick := kitex.UseState[uint64](0)
	tick := func() {
		setTick(hookTickSeq.Add(1))
	}

	entry, closed := client.getOrCreateEntry(key)

	// Stop gcTimer if it was running
	entry.mu.Lock()
	if entry.gcTimer != nil {
		entry.gcTimer.Stop()
		entry.gcTimer = nil
	}
	if !closed && !entry.closed {
		typeEraser := func(ctx context.Context) *promise.Promise[any] {
			p := fetcher(ctx, key)
			return promise.New(func(ctx context.Context) (any, error) {
				return p.Await(ctx)
			})
		}
		entry.refetch = func() {
			client.executeFetch(entry, typeEraser)
		}
	}
	entry.mu.Unlock()

	var renderedVersion uint64

	// Subscribe to changes
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

	// Trigger fetch if enabled and stale/first load
	kitex.UseLayoutEffect(func() {
		if !enabled {
			return
		}
		entry.mu.Lock()
		if entry.closed {
			entry.mu.Unlock()
			return
		}
		needsFetch := !entry.state.isFetching && (entry.state.status == "loading" || time.Since(entry.state.updatedAt) > staleTime)
		refetch := entry.refetch
		entry.mu.Unlock()

		if needsFetch && refetch != nil {
			refetch()
		}
	}, []any{key, enabled, staleTime})

	// Prepare result
	entry.mu.Lock()
	defer entry.mu.Unlock()
	renderedVersion = entry.version

	var data T
	if entry.state.hasData {
		if typed, ok := entry.state.data.(T); ok {
			data = typed
		}
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
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	rootCtx := c.rootCtx
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	c.wg.Add(1)
	c.mu.Unlock()

	entry.mu.Lock()
	if entry.closed || entry.state.isFetching {
		entry.mu.Unlock()
		c.wg.Done()
		return
	}

	entry.state.isFetching = true
	entry.version++
	ctx, cancel := context.WithCancel(rootCtx)
	entry.cancel = cancel
	entry.mu.Unlock()

	// Notify observers that fetching has started
	entry.notifySubscribers()

	p := fetcher(ctx)
	if p == nil {
		entry.mu.Lock()
		if entry.state.isFetching {
			entry.state.isFetching = false
			entry.cancel = nil
			entry.version++
		}
		entry.mu.Unlock()
		cancel()
		c.wg.Done()
		return
	}

	go func() {
		_, _ = p.Await(context.Background())
		c.wg.Done()
	}()

	p.Then(func(data any) {
		entry.mu.Lock()
		if ctx.Err() != nil || entry.closed {
			entry.mu.Unlock()
			return
		}

		entry.state.isFetching = false
		entry.cancel = nil
		entry.state.data = data
		entry.state.hasData = true
		entry.state.err = nil
		entry.state.status = "success"
		entry.state.updatedAt = time.Now()
		entry.version++

		shouldRefetch := entry.invalidatedDuringFetch
		entry.invalidatedDuringFetch = false
		refetch := entry.refetch
		entry.mu.Unlock()

		entry.notifySubscribers()

		if shouldRefetch && refetch != nil {
			refetch()
		}
	}, func(err error) {
		entry.mu.Lock()
		if ctx.Err() != nil || entry.closed {
			entry.mu.Unlock()
			return
		}

		entry.state.isFetching = false
		entry.cancel = nil
		entry.state.err = err
		entry.state.status = "error"
		entry.version++

		shouldRefetch := entry.invalidatedDuringFetch
		entry.invalidatedDuringFetch = false
		refetch := entry.refetch
		entry.mu.Unlock()

		entry.notifySubscribers()

		if shouldRefetch && refetch != nil {
			refetch()
		}
	}).
		Finally(cancel)
}
