package engine

import (
	"github.com/masterkeysrd/kite/terminal"
)

type TerminalProxy struct{}

var _ terminal.Terminal = (*TerminalProxy)(nil)

func (tp *TerminalProxy) Clipboard() terminal.Clipboard {
	return &Clipboard{}
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
