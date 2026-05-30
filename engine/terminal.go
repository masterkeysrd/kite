package engine

import (
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"
)

type TerminalProxy struct {
	e *Engine
}

var _ terminal.Terminal = (*TerminalProxy)(nil)

func (tp *TerminalProxy) Clipboard() terminal.Clipboard {
	return &Clipboard{}
}

func (tp *TerminalProxy) Scheduler() terminal.Scheduler {
	return tp.e.scheduler
}

type Clipboard struct{}

var _ terminal.Clipboard = (*Clipboard)(nil)

func (c *Clipboard) ReadText() *promise.Promise[string] {
	return promise.Resolved("")
}

func (c *Clipboard) WriteText(text string) *promise.Promise[struct{}] {
	return promise.Resolved(struct{}{})
}

func (c *Clipboard) Read(mime string) *promise.Promise[[]byte] {
	return promise.Resolved([]byte(nil))
}

func (c *Clipboard) Write(mime string, data []byte) *promise.Promise[struct{}] {
	return promise.Resolved(struct{}{})
}
