package engine

import (
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/terminal"
)

type TerminalProxy struct{}

var _ terminal.Terminal = (*TerminalProxy)(nil)

func (tp *TerminalProxy) Clipboard() terminal.Clipboard {
	return &Clipboard{}
}

func (tp *TerminalProxy) Layout() terminal.Layout {
	return &Layout{}
}

type Clipboard struct{}

var _ terminal.Clipboard = (*Clipboard)(nil)

func (c *Clipboard) ReadText() (string, error) {
	return "", nil
}

func (c *Clipboard) WriteText(text string) error {
	return nil
}

func (c *Clipboard) Read(mime string) ([]byte, error) {
	return nil, nil
}

func (c *Clipboard) Write(mime string, data []byte) error {
	return nil
}

type Layout struct {
	nodes map[terminal.Node]layout.Node
}

var _ terminal.Layout = (*Layout)(nil)

func (l *Layout) GetSizeOf(node terminal.Node) (geom.Size, bool) {
	return geom.Size{}, false
}

func (l *Layout) GetAbsoluteBoundsOf(node terminal.Node) (geom.Rect, bool) {
	return geom.Rect{}, false
}
