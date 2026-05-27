package wind

import (
	"context"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
)

type queryState struct {
	data       any
	err        error
	status     string // "loading", "success", "error"
	isFetching bool
	updatedAt  time.Time
}

type queryEntry struct {
	mu          sync.Mutex
	key         any
	state       queryState
	subscribers map[int]func()
	nextSubID   int
	refetch     func()
	cancel      context.CancelFunc
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
	mu    sync.Mutex
	cache map[any]*queryEntry
}

func NewClient() *Client {
	return &Client{
		cache: make(map[any]*queryEntry),
	}
}

// InvalidateQueries invalidates query for key.
func (c *Client) InvalidateQueries(key any) {
	c.mu.Lock()
	entry, ok := c.cache[key]
	c.mu.Unlock()

	if ok && entry != nil {
		entry.mu.Lock()
		refetch := entry.refetch
		entry.mu.Unlock()
		if refetch != nil {
			refetch()
		}
	}
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
