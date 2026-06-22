package wind

import (
	"context"
	"iter"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
)

type StreamOptions struct {
	Enabled bool
}

type StreamResult[T any] struct {
	Result[T]
	Status string
}

func UseStream[K comparable, T any](
	key K,
	fetcher func(ctx context.Context, key K) iter.Seq2[T, error],
	opts ...StreamOptions,
) StreamResult[T] {
	client := UseClient()
	if client == nil {
		panic("wind.UseStream must be used inside a wind.Provider")
	}

	enabled := true
	if len(opts) > 0 {
		enabled = opts[0].Enabled
	}

	_, setTick := kitex.UseState(time.Now().UnixNano())
	tick := func() {
		setTick(time.Now().UnixNano())
	}

	// Retrieve or create query entry in client cache
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

	// Register refetch/reconnect implementation
	entry.mu.Lock()
	typeEraser := func(ctx context.Context) iter.Seq2[any, error] {
		return func(yield func(any, error) bool) {
			seq := fetcher(ctx, key)
			seq(func(val T, err error) bool {
				return yield(val, err)
			})
		}
	}
	entry.refetch = func() {
		client.executeStream(entry, typeEraser)
	}
	entry.mu.Unlock()

	// Component mounts/unmounts: Manage subscriber count and cancel on drop to zero
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

	// Trigger connection if enabled and not currently active
	kitex.UseLayoutEffect(func() {
		if !enabled {
			return
		}
		entry.mu.Lock()
		needsConnect := !entry.state.isFetching && entry.state.status == "loading"
		entry.mu.Unlock()

		if needsConnect {
			entry.refetch()
		}
	}, []any{key, enabled})

	// Prepare Result state
	entry.mu.Lock()
	defer entry.mu.Unlock()

	var data T
	if entry.state.data != nil {
		data = entry.state.data.(T)
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

	return StreamResult[T]{
		Result: Result[T]{
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
