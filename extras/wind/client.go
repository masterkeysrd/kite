package wind

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

const defaultStreamThrottle = 16 * time.Millisecond

var hookTickSeq atomic.Uint64

type queryState struct {
	data       any
	hasData    bool
	err        error
	status     string // "loading", "success", "error"
	isFetching bool
	updatedAt  time.Time
}

type queryEntry struct {
	mu                     sync.Mutex
	key                    any
	state                  queryState
	version                uint64
	subscribers            map[int]func()
	enabledSubs            map[int]struct{}
	nextSubID              int
	refetch                func()
	cancel                 context.CancelFunc
	updatePending          bool
	refetchPending         bool
	gcTimer                *time.Timer
	invalidatedDuringFetch bool
	lastNotifyTime         time.Time
	throttleTimer          *time.Timer
	closed                 bool
}

func (e *queryEntry) notifySubscribers() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
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
	mu         sync.Mutex
	cache      map[any]*queryEntry
	GcTime     time.Duration
	closed     bool
	rootCtx    context.Context
	rootCancel context.CancelFunc
	wg         sync.WaitGroup
}

func NewClient() *Client {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	return &Client{
		cache:      make(map[any]*queryEntry),
		GcTime:     5 * time.Minute,
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
	}
}

func (c *Client) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Client) getOrCreateEntry(key any) (*queryEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return &queryEntry{
			key: key,
			state: queryState{
				status: "loading",
			},
			subscribers: make(map[int]func()),
			enabledSubs: make(map[int]struct{}),
			closed:      true,
		}, true
	}

	entry, ok := c.cache[key]
	if !ok {
		entry = &queryEntry{
			key: key,
			state: queryState{
				status: "loading",
			},
			subscribers: make(map[int]func()),
			enabledSubs: make(map[int]struct{}),
		}
		c.cache[key] = entry
	}
	return entry, false
}

// Close cancels all active queries and streams, stops pending timers, clears
// the cache, and waits for all background operations started by the client to
// finish. Close is safe to call multiple times.
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.wg.Wait()
		return
	}
	c.closed = true
	if c.rootCancel != nil {
		c.rootCancel()
	}
	entries := make([]*queryEntry, 0, len(c.cache))
	for _, entry := range c.cache {
		if entry != nil {
			entries = append(entries, entry)
		}
	}
	clear(c.cache)
	c.mu.Unlock()

	for _, entry := range entries {
		entry.mu.Lock()
		entry.closed = true
		if entry.cancel != nil {
			entry.cancel()
			entry.cancel = nil
		}
		if entry.gcTimer != nil {
			entry.gcTimer.Stop()
			entry.gcTimer = nil
		}
		if entry.throttleTimer != nil {
			entry.throttleTimer.Stop()
			entry.throttleTimer = nil
		}
		entry.updatePending = false
		entry.refetchPending = false
		entry.invalidatedDuringFetch = false
		clear(entry.subscribers)
		clear(entry.enabledSubs)
		entry.refetch = nil
		if entry.state.isFetching {
			entry.state.isFetching = false
			entry.version++
		}
		entry.mu.Unlock()
	}

	c.wg.Wait()
}

type streamConfig struct {
	initial     any
	throttle    Optional[time.Duration]
	refetchMode RefetchMode
	run         func(ctx context.Context, prev any, hasPrev bool, mode RefetchMode, emit func(data any, err error) bool)
}

func (c *Client) executeStream(entry *queryEntry, cfg streamConfig) {
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
	if entry.closed {
		entry.mu.Unlock()
		c.wg.Done()
		return
	}
	// If a stream is already running, cancel the old one first (reconnection/invalidation)
	if entry.cancel != nil {
		entry.cancel()
		entry.cancel = nil
	}
	if entry.throttleTimer != nil {
		entry.throttleTimer.Stop()
		entry.throttleTimer = nil
	}
	entry.updatePending = false

	switch cfg.refetchMode {
	case RefetchReset:
		entry.state.data = cfg.initial
		entry.state.hasData = false
		entry.state.err = nil
		entry.state.status = "loading"
	default:
		entry.state.err = nil
		if !entry.state.hasData {
			entry.state.status = "loading"
		} else if entry.state.status == "error" {
			entry.state.status = "success"
		}
	}

	prevData := entry.state.data
	hasPrev := entry.state.hasData
	entry.state.isFetching = true
	entry.version++

	ctx, cancel := context.WithCancel(rootCtx)
	entry.cancel = cancel
	entry.mu.Unlock()

	// Notify observers that connection is starting
	entry.notifySubscribers()

	throttle := cfg.throttle.UnwrapOr(defaultStreamThrottle)

	started := promise.Spawn(func(schedCtx context.Context) {
		defer c.wg.Done()
		defer cancel()

		stopLink := context.AfterFunc(schedCtx, cancel)
		defer stopLink()
		if schedCtx.Err() != nil {
			cancel()
		}
		if ctx.Err() != nil {
			entry.mu.Lock()
			if !entry.closed && entry.state.isFetching {
				entry.state.isFetching = false
				entry.cancel = nil
				entry.version++
			}
			entry.mu.Unlock()
			return
		}

		cfg.run(ctx, prevData, hasPrev, cfg.refetchMode, func(data any, err error) bool {
			entry.mu.Lock()
			if ctx.Err() != nil || entry.closed {
				entry.mu.Unlock()
				return false
			}

			if err != nil {
				entry.state.status = "error"
				entry.state.err = err
				entry.state.isFetching = false
				entry.cancel = nil
				entry.updatePending = false
				entry.version++
				if entry.throttleTimer != nil {
					entry.throttleTimer.Stop()
					entry.throttleTimer = nil
				}
				entry.mu.Unlock()

				promise.Resolved(any(nil)).Then(func(any) {
					if c.isClosed() {
						return
					}
					entry.notifySubscribers()
				}, nil)
				return false
			}

			entry.state.status = "success"
			entry.state.data = data
			entry.state.hasData = true
			entry.state.err = nil
			entry.state.updatedAt = time.Now()
			entry.version++

			if !entry.updatePending {
				if throttle <= 0 {
					entry.updatePending = true
					entry.mu.Unlock()

					promise.Resolved(any(nil)).Then(func(any) {
						entry.mu.Lock()
						if ctx.Err() == nil && !entry.closed {
							entry.updatePending = false
						}
						entry.mu.Unlock()
						if c.isClosed() {
							return
						}
						entry.notifySubscribers()
					}, nil)
				} else {
					entry.updatePending = true
					now := time.Now()
					delay := throttle - now.Sub(entry.lastNotifyTime)
					if delay <= 0 {
						entry.lastNotifyTime = now
						if entry.throttleTimer != nil {
							entry.throttleTimer.Stop()
							entry.throttleTimer = nil
						}
						entry.mu.Unlock()

						promise.Resolved(any(nil)).Then(func(any) {
							entry.mu.Lock()
							if ctx.Err() == nil && !entry.closed {
								entry.updatePending = false
							}
							entry.mu.Unlock()
							if c.isClosed() {
								return
							}
							entry.notifySubscribers()
						}, nil)
					} else {
						if entry.throttleTimer == nil {
							entry.throttleTimer = time.AfterFunc(delay, func() {
								if c.isClosed() {
									return
								}
								entry.mu.Lock()
								if ctx.Err() != nil || entry.closed {
									entry.mu.Unlock()
									return
								}
								entry.lastNotifyTime = time.Now()
								entry.updatePending = false
								entry.throttleTimer = nil
								entry.mu.Unlock()

								promise.Resolved(any(nil)).Then(func(any) {
									if c.isClosed() {
										return
									}
									entry.notifySubscribers()
								}, nil)
							})
						}
						entry.mu.Unlock()
					}
				}
			} else {
				entry.mu.Unlock()
			}

			return true
		})

		entry.mu.Lock()
		shouldNotify := false
		if ctx.Err() == nil && !entry.closed {
			entry.updatePending = false
			if entry.throttleTimer != nil {
				entry.throttleTimer.Stop()
				entry.throttleTimer = nil
			}
			if entry.state.isFetching {
				entry.state.isFetching = false
				entry.cancel = nil
				entry.version++
				shouldNotify = true
			}
		} else if schedCtx.Err() != nil && !entry.closed {
			entry.updatePending = false
			if entry.throttleTimer != nil {
				entry.throttleTimer.Stop()
				entry.throttleTimer = nil
			}
			if entry.state.isFetching {
				entry.state.isFetching = false
				entry.cancel = nil
				entry.version++
				shouldNotify = true
			}
		}
		entry.mu.Unlock()

		if shouldNotify {
			promise.Resolved(any(nil)).Then(func(any) {
				if c.isClosed() {
					return
				}
				entry.notifySubscribers()
			}, nil)
		}
	})
	if !started {
		entry.mu.Lock()
		if !entry.closed && entry.state.isFetching {
			entry.state.isFetching = false
			entry.cancel = nil
			entry.version++
		}
		entry.mu.Unlock()
		cancel()
		c.wg.Done()
		entry.notifySubscribers()
	}
}

var clientContext = kitex.CreateContext[*Client](nil)

type ProviderProps struct {
	Client *Client
}

var providerFCC = kitex.FCC("WindProvider", func(props ProviderProps, children ...kitex.Node) kitex.Node {
	return clientContext.Provider(props.Client, children...)
})

// Provider makes the Client available to the component tree.
func Provider(props ProviderProps, children ...kitex.Node) kitex.Node {
	return providerFCC.Props(props, children...)
}

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
		for i := range sLen {
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
	if c.closed {
		c.mu.Unlock()
		return
	}
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
			if entry.closed {
				entry.mu.Unlock()
				continue
			}
			if entry.state.isFetching {
				entry.invalidatedDuringFetch = true
				entry.mu.Unlock()
				continue
			}
			if entry.refetchPending {
				entry.mu.Unlock()
				continue
			}
			if entry.refetch != nil {
				entry.refetchPending = true
				entry.mu.Unlock()

				promise.Resolved(any(nil)).Then(func(any) {
					if c.isClosed() {
						return
					}
					entry.mu.Lock()
					entry.refetchPending = false
					refetch := entry.refetch
					closed := entry.closed
					entry.mu.Unlock()

					if !closed && refetch != nil {
						refetch()
					}
				}, nil)
			} else {
				entry.mu.Unlock()
			}
		}
	}
}
