package kitty

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/event"
)

const (
	// OSCSecureClipboard is the OSC code for secure clipboard transfer.
	OSCSecureClipboard = 5522

	// ModeSecureClipboard is the DEC mode used to enable the protocol.
	ModeSecureClipboard = 5522

	// Protocol parameter keys.
	paramType     = "type"
	paramStatus   = "status"
	paramPassword = "password"
	paramMime     = "mime"

	// Protocol values.
	typeRead     = "read"
	statusOK     = "OK"
	statusData   = "DATA"
	statusDone   = "DONE"
	fallbackMime = "text/plain"
)

// Extension implements backend.TerminalExtension for Kitty's secure clipboard
// transfer protocol (OSC 5522).
type Extension struct {
	out              io.Writer
	password         string
	mime             string
	buffer           []byte
	enabled          bool
	receivedResponse bool
}

var _ backend.TerminalExtension = (*Extension)(nil)

// NewExtension creates a new Kitty extension.
func NewExtension() *Extension {
	return &Extension{}
}

// Init enables Kitty's secure clipboard transfer mode.
func (e *Extension) Init(out io.Writer) {
	e.out = out
	e.enable()
}

func (e *Extension) enable() {
	// Enable DEC mode 5522 (Secure clipboard transfer).
	slog.Info("KITTY: Sending Enable sequence", "mode", ModeSecureClipboard)
	e.out.Write([]byte(fmt.Sprintf("\x1b[?%dh", ModeSecureClipboard)))
	e.enabled = true
}

// HandleEvent processes OSC 5522 sequences to perform the multi-step paste handshake.
func (e *Extension) HandleEvent(raw event.RawEvent) (bool, event.Event) {
	// Re-enable until we get any 5522 response or a paste event.
	if !e.receivedResponse && e.out != nil {
		e.enable()
	}

	var data string
	var code int

	switch ev := raw.(type) {
	case *event.RawBracketedPaste:
		// If we see a standard paste, it means the terminal ignored our 5522 enable.
		e.receivedResponse = true
	case *event.RawOscEvent:
		code = ev.Code
		data = ev.Data
		if code == OSCSecureClipboard {
			e.receivedResponse = true
		}
	case *event.RawUnknownEvent:
		// Attempt to parse 5522 from raw strings (e.g. from uv.UnknownOscEvent)
		codeStr := fmt.Sprintf("%d", OSCSecureClipboard)
		if s, ok := ev.Payload.(string); ok && strings.Contains(s, codeStr) {
			e.receivedResponse = true
			slog.Debug("KITTY: Detected handshake in unknown event", "code", codeStr)
			// Format might be "5522;..." or "UNK:5522;..."
			parts := strings.SplitN(s, codeStr, 2)
			if len(parts) == 2 {
				code = OSCSecureClipboard
				data = strings.TrimPrefix(parts[1], ";")
				data = strings.TrimPrefix(data, ":")
			}
		}
	}

	if code != OSCSecureClipboard {
		return false, nil
	}

	slog.Info("KITTY: Processing handshake", "data", data)
	params := parseParams(data)
	typ := params[paramType]
	status := params[paramStatus]

	if typ != typeRead {
		return true, nil
	}

	switch status {
	case statusOK:
		// Step 1: Terminal sends password.
		e.password = params[paramPassword]

	case statusData:
		mime := params[paramMime]
		if mime == "" {
			break
		}

		// DATA status is used for both notification and chunk delivery.
		if strings.Contains(data, ";") {
			// Step 3: Chunk delivery (status=DATA:mime=...;<base64_chunk>).
			parts := strings.SplitN(data, ";", 2)
			if len(parts) == 2 {
				chunk, _ := base64.StdEncoding.DecodeString(parts[1])
				e.buffer = append(e.buffer, chunk...)
			}
		} else {
			// Step 2: Notification of available MIME type.
			// Request the data for this MIME type using the captured password.
			e.mime = mime
			if e.password != "" {
				seq := fmt.Sprintf("\x1b]%d;%s=%s:%s=%s:%s=%s\x1b\\",
					OSCSecureClipboard,
					paramType, typeRead,
					paramMime, e.mime,
					paramPassword, e.password)
				e.out.Write([]byte(seq))
				e.buffer = nil // Reset buffer for new transfer.
			}
		}

	case statusDone:
		// Step 4: Transfer complete.
		ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste, nil)
		mimeTypeBytes, _ := base64.StdEncoding.DecodeString(e.mime)
		mimeType := string(mimeTypeBytes)
		if mimeType == "" {
			mimeType = fallbackMime
		}
		ce.Items[mimeType] = e.buffer
		e.buffer = nil
		return true, ce
	}

	return true, nil
}

func parseParams(data string) map[string]string {
	res := make(map[string]string)
	// Typical format: key1=val1:key2=val2...
	// Note: value might contain more colons if it's the last one, but Kitty
	// params usually don't.
	pairs := strings.Split(data, ":")
	for _, p := range pairs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}
