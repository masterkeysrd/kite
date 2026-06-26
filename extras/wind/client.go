package wind

import (
	"context"
	"iter"
	"reflect"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

type queryState struct {
	data       any
	err        error
	status     string // "loading", "success", "error"
	isFetching bool
	updatedAt  time.Time
}

type queryEntry struct {
	mu                     sync.Mutex
	key                    any
	state                  queryState
	subscribers            map[int]func()
	nextSubID              int
	refetch                func()
	cancel                 context.CancelFunc
	updatePending          bool
	refetchPending         bool
	gcTimer                *time.Timer
	invalidatedDuringFetch bool
}

func (e *queryEntry) notifySubscribers() {
	e.mu.Lock()
	subs := make([]func(), 0, len(e.subscribers))
	for _, sub := range e.subscribers {
		subs = append(subs, sub)
	}
	e.mu.Unlock()

	for _, sub := range subs {
		sub()
	}
}

type Client struct {
	mu     sync.Mutex
	cache  map[any]*queryEntry
	GcTime time.Duration
}

func NewClient() *Client {
	return &Client{
		cache:  make(map[any]*queryEntry),
		GcTime: 5 * time.Minute,
	}
}

func (c *Client) executeStream(entry *queryEntry, fetcher func(context.Context) iter.Seq2[any, error]) {
	entry.mu.Lock()
	// If a stream is already running, cancel the old one first (reconnection/invalidation)
	if entry.cancel != nil {
		entry.cancel()
		entry.cancel = nil
	}

	entry.state.isFetching = true
	if entry.state.data == nil {
		entry.state.status = "loading"
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry.cancel = cancel
	entry.mu.Unlock()

	// Notify observers that connection is starting
	entry.notifySubscribers()

	_ = promise.New(func(taskCtx context.Context) (any, error) {
		seq := fetcher(ctx)

		// Consume the iterator
		seq(func(data any, err error) bool {
			entry.mu.Lock()
			// If this connection was cancelled or replaced, stop iterating
			if ctx.Err() != nil {
				entry.mu.Unlock()
				return false
			}

			// Update cache state synchronously (very cheap memory operation)
			if err != nil {
				entry.state.status = "error"
				entry.state.err = err
				entry.state.isFetching = false
				entry.cancel = nil
				entry.mu.Unlock()

				promise.Resolved(any(nil)).Then(func(any) {
					entry.notifySubscribers()
				}, nil)
				return false // Stop iterating on error
			}

			entry.state.status = "success"
			entry.state.data = data
			entry.state.err = nil

			// Microtask Coalescing: Only queue a main thread update if one is not already pending.
			if !entry.updatePending {
				entry.updatePending = true
				entry.mu.Unlock()

				promise.Resolved(any(nil)).Then(func(any) {
					entry.mu.Lock()
					entry.updatePending = false
					if ctx.Err() != nil {
						entry.mu.Unlock()
						return
					}
					entry.mu.Unlock()

					// Notify subscribers to trigger re-renders on the main UI thread
					entry.notifySubscribers()
				}, nil)
			} else {
				entry.mu.Unlock()
			}

			return true // Continue iterating
		})

		// When iterator exits naturally (reaches EOF/ends), mark isFetching as false
		promise.Resolved(any(nil)).Then(func(any) {
			entry.mu.Lock()
			if ctx.Err() != nil {
				entry.mu.Unlock()
				return
			}

			shouldNotify := false
			if entry.state.isFetching {
				entry.state.isFetching = false
				entry.cancel = nil
				shouldNotify = true
			}
			entry.mu.Unlock()

			if shouldNotify {
				entry.notifySubscribers()
			}
		}, nil)

		return nil, nil
	})
}

var clientContext = kitex.CreateContext[*Client](nil)

type ProviderProps struct {
	Client   *Client
	Children []kitex.Node
}

// Provider makes the Client available to the component tree.
var Provider = kitex.FCC("WindProvider", func(props ProviderProps) kitex.Node {
	return clientContext.Provider(props.Client, props.Children...)
})

// UseClient extracts the client from the context.
func UseClient() *Client {
	return kitex.UseContext(clientContext)
}

func safeEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	ta := reflect.TypeOf(a)
	tb := reflect.TypeOf(b)
	if ta != tb || !ta.Comparable() {
		return false
	}
	return a == b
}

func isMatch(cachedKey, searchKey any) bool {
	if safeEqual(cachedKey, searchKey) {
		return true
	}
	if pred, ok := searchKey.(func(any) bool); ok {
		return pred(cachedKey)
	}

	sv := reflect.ValueOf(searchKey)
	cv := reflect.ValueOf(cachedKey)

	// Check if both are array/slice
	if (sv.Kind() == reflect.Slice || sv.Kind() == reflect.Array) &&
		(cv.Kind() == reflect.Slice || cv.Kind() == reflect.Array) {
		sLen := sv.Len()
		cLen := cv.Len()
		if sLen > cLen {
			return false
		}
		for i := 0; i < sLen; i++ {
			se := sv.Index(i).Interface()
			ce := cv.Index(i).Interface()
			if !safeEqual(se, ce) {
				return false
			}
		}
		return true
	}

	return false
}

// InvalidateQueries invalidates all queries matching the search key/predicate.
func (c *Client) InvalidateQueries(key any) {
	c.mu.Lock()
	var entries []*queryEntry
	for k, entry := range c.cache {
		if isMatch(k, key) {
			entries = append(entries, entry)
		}
	}
	c.mu.Unlock()

	for _, entry := range entries {
		if entry != nil {
			entry.mu.Lock()
			if entry.state.isFetching {
				entry.invalidatedDuringFetch = true
				entry.mu.Unlock()
				continue
			}
			if entry.refetchPending {
				entry.mu.Unlock()
				continue
			}
			refetch := entry.refetch
			if refetch != nil {
				entry.refetchPending = true
				entry.mu.Unlock()

				promise.Resolved(any(nil)).Then(func(any) {
					entry.mu.Lock()
					entry.refetchPending = false
					entry.mu.Unlock()

					refetch()
				}, nil)
			} else {
				entry.mu.Unlock()
			}
		}
	}
}
