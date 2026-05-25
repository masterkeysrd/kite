package osc52

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/event"
)

// Extension implements backend.TerminalExtension and backend.ClipboardProvider
// for the standard OSC 52 protocol.
type Extension struct {
	out io.Writer
}

var _ backend.TerminalExtension = (*Extension)(nil)
var _ event.ClipboardProvider = (*Extension)(nil)

func NewExtension() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string { return "osc52" }

func (e *Extension) Init(out io.Writer) {
	e.out = out
}

func (e *Extension) SetClipboard(text string) {
	if e.out == nil {
		return
	}
	data := base64.StdEncoding.EncodeToString([]byte(text))
	e.writeEsc(fmt.Sprintf("\x1b]52;c;%s\x1b\\", data))
}

func (e *Extension) RequestClipboard() {
	if e.out == nil {
		return
	}
	e.writeEsc("\x1b]52;c;?\x1b\\")
}

func (e *Extension) writeEsc(s string) {
	fmt.Fprint(e.out, s)
}

func (e *Extension) HandleEvent(raw event.RawEvent) (bool, event.Event) {
	if osc, ok := raw.(*event.RawOscEvent); ok && osc.Code == 52 {
		// Data format is "c;<base64>" or "p;<base64>" etc.
		parts := strings.SplitN(osc.Data, ";", 2)
		if len(parts) == 2 {
			b64 := parts[1]
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err == nil {
				ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
				ce.SetText(string(decoded))
				return true, ce
			}
		}
		return true, nil
	}
	return false, nil
}
