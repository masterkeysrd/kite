package wind

import (
	"context"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
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

func Use[K comparable, T any](key K, fetcher func(context.Context) (T, error), opts ...Options) Result[T] {
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

	// Register refetch implementation
	entry.mu.Lock()
	typeEraser := func(ctx context.Context) (any, error) {
		return fetcher(ctx)
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
			if len(entry.subscribers) == 0 && entry.cancel != nil {
				entry.cancel()
				entry.cancel = nil
				entry.state.isFetching = false
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
		IsLoading:  entry.state.status == "loading" && entry.state.isFetching,
		IsFetching: entry.state.isFetching,
		IsError:    entry.state.status == "error",
		Error:      entry.state.err,
		Refetch:    entry.refetch,
	}
}

func (c *Client) executeFetch(entry *queryEntry, fetcher func(context.Context) (any, error)) {
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

	go func() {
		data, err := fetcher(ctx)

		entry.mu.Lock()
		defer entry.mu.Unlock()

		if ctx.Err() != nil {
			// Context was cancelled, ignore result
			return
		}

		entry.state.isFetching = false
		entry.cancel = nil
		if err != nil {
			entry.state.err = err
			entry.state.status = "error"
		} else {
			entry.state.data = data
			entry.state.err = nil
			entry.state.status = "success"
			entry.state.updatedAt = time.Now()
		}

		// Notify observers of completion
		go entry.notifySubscribers()
	}()
}
