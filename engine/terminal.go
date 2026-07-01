package engine

import (
	"context"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/internal/collections"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"
)

type TerminalProxy struct {
	e *Engine
}

var _ terminal.Terminal = (*TerminalProxy)(nil)

func (tp *TerminalProxy) Clipboard() terminal.Clipboard {
	return &ClipboardProxy{e: tp.e}
}

func (tp *TerminalProxy) Scheduler() terminal.Scheduler {
	return tp.e.scheduler
}

func (tp *TerminalProxy) SetTitle(title string) {
	tp.e.SetTitle(title)
}

func (tp *TerminalProxy) Bell() {
	tp.e.Bell()
}

func (tp *TerminalProxy) SetProgressBar(state terminal.ProgressBarState, percentage int) {
	tp.e.SetProgressBar(state, percentage)
}

type pendingRead struct {
	mime string
	ch   chan []byte
}

type clipboardState struct {
	mu      sync.Mutex
	pending []pendingRead
}

func (s *clipboardState) resolvePending(items map[string][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stillPending []pendingRead
	for _, p := range s.pending {
		if data, ok := items[p.mime]; ok {
			select {
			case p.ch <- data:
			default:
			}
		} else {
			stillPending = append(stillPending, p)
		}
	}
	s.pending = stillPending
}

type ClipboardProxy struct {
	e *Engine
}

var _ terminal.Clipboard = (*ClipboardProxy)(nil)

func (c *ClipboardProxy) ReadText() *promise.Promise[string] {
	return promise.New(func(ctx context.Context) (string, error) {
		p := c.Read("text/plain")
		data, err := p.Await(ctx)
		if err != nil {
			return "", err
		}
		return string(data), nil
	})
}

func (c *ClipboardProxy) WriteText(text string) *promise.Promise[struct{}] {
	return c.Write("text/plain", []byte(text))
}

func (c *ClipboardProxy) Read(mime string) *promise.Promise[[]byte] {
	return promise.New(func(ctx context.Context) ([]byte, error) {
		ch := make(chan []byte, 1)

		c.e.clipboard.mu.Lock()
		c.e.clipboard.pending = append(c.e.clipboard.pending, pendingRead{mime: mime, ch: ch})
		firstForMime := true
		for _, p := range c.e.clipboard.pending[:len(c.e.clipboard.pending)-1] {
			if p.mime == mime {
				firstForMime = false
				break
			}
		}
		c.e.clipboard.mu.Unlock()

		if firstForMime {
			c.e.backend.Clipboard().Request(mime)
		}

		select {
		case data := <-ch:
			return data, nil
		case <-time.After(500 * time.Millisecond):
			// Timeout: remove ourselves from the pending list if still there.
			c.e.clipboard.mu.Lock()
			for i, p := range c.e.clipboard.pending {
				if p.ch == ch {
					c.e.clipboard.pending = collections.DeleteAt(c.e.clipboard.pending, i)
					break
				}
			}
			c.e.clipboard.mu.Unlock()
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
}

func (c *ClipboardProxy) Write(mime string, data []byte) *promise.Promise[struct{}] {
	c.e.backend.Clipboard().Set(mime, data)
	return promise.Resolved(struct{}{})
}
