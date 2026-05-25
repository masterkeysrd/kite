package kitty

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	paramType   = "type"
	paramStatus = "status"
	paramMime   = "mime"
	// pw is the one-time password key sent by kitty in the paste event.
	paramPw = "pw"
	// Protocol values.
	typeRead     = "read"
	statusOK     = "OK"
	statusData   = "DATA"
	statusDone   = "DONE"
	fallbackMime = "text/plain"
)

// Extension implements backend.TerminalExtension and event.ClipboardProvider
// for Kitty's secure clipboard transfer protocol (OSC 5522).
type Extension struct {
	out            io.Writer
	password       string
	mime           string
	buffer         []byte
	availableMimes []string // List of MIME types announced by Kitty
	requesting     bool     // True when we have requested data and are waiting for chunks
	started        bool     // True if we have sent the Init sequences
	initialized    bool     // True if we have received a response
}

const (
	tmuxId = "kite"
)

func (e *Extension) write(s string) {
	if e.out == nil {
		return
	}
	if os.Getenv("TMUX") != "" {
		// Wrap for tmux pass-through.
		// We double every ESC inside the sequence and wrap it in DCS tmux ; ... ST
		s = "\x1bPtmux;" + strings.ReplaceAll(s, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	e.out.Write([]byte(s)) //nolint:errcheck
}

var _ backend.TerminalExtension = (*Extension)(nil)
var _ event.ClipboardProvider = (*Extension)(nil)

// NewExtension creates a new Kitty extension.
func NewExtension() *Extension {
	return &Extension{}
}

func (e *Extension) Name() string { return "kitty" }

// SetClipboard is a no-op; the kitty protocol initiates writes via OSC 5522
// write packets, which is not yet implemented.
func (e *Extension) SetClipboard(text string) {}

// RequestClipboard initiates a rich clipboard read by asking for a MIME type list.
func (e *Extension) RequestClipboard() {
	slog.Info("KITTY: Requesting rich clipboard content list")
	// "Lg==" is base64 for "." (list all MIME types).
	// Documentation says: <OSC>5522;metadata;payload<ST>
	// metadata is type=read. payload is "." B64 encoded.
	e.write(fmt.Sprintf("\x1b]5522;type=read:id=%s;Lg==\x1b\\", tmuxId))
}

// Init enables Kitty's secure clipboard transfer mode and queries support.
func (e *Extension) Init(out io.Writer) {
	e.out = out
	// Enable Kitty's DEC private mode 5522 (secure clipboard / paste events)
	// and query the mode to confirm support.
	slog.Info("KITTY: Sending Enable and Query sequences during Init", "mode", ModeSecureClipboard)
	e.write(fmt.Sprintf("\x1b[?%dh\x1b[?%d$p", ModeSecureClipboard, ModeSecureClipboard))
	e.started = true
}

// HandleEvent processes OSC 5522 sequences to perform the multi-step paste handshake.
func (e *Extension) HandleEvent(raw event.RawEvent) (bool, event.Event) {
	if !e.started && e.out != nil {
		e.Init(e.out)
	}

	// slog.Debug("KITTY: Received event", "type", fmt.Sprintf("%T", raw), "val", fmt.Sprintf("%+v", raw))

	// Capture any event that might indicate we are initialized
	switch raw.(type) {
	case *event.RawClipboardEvent, *event.RawOscEvent, *event.RawBracketedPaste:
		if !e.initialized {
			slog.Info("KITTY: Extension initialized via terminal response", "type", fmt.Sprintf("%T", raw))
		}
		e.initialized = true
	}

	var data string
	var code int

	switch ev := raw.(type) {
	case *event.RawUnknownEvent:
		s := fmt.Sprintf("%v", ev.Payload)
		// slog.Debug("KITTY: Processing unknown event", "payload", s)

		// Detect the DECRQM capability report: CSI ? 5522 ; <status> $ y
		// This can come in several forms depending on the backend/multiplexer:
		// - "CSI:?5522;1$y"
		// - "?5522;1$y"
		// - "\x1b[?5522;1$y"
		// - "5522;1$y" (stripped by some filters)
		s_clean := strings.TrimSpace(s)
		if strings.Contains(s_clean, "5522") && (strings.HasSuffix(s_clean, "$y") || strings.Contains(s_clean, "$y")) {
			slog.Info("KITTY: Received DECRQM capability report, mode confirmed", "mode", ModeSecureClipboard, "raw", s)
			e.initialized = true
			return true, nil
		}

		// Fallback for OSC 5522 if backend didn't parse it as RawOscEvent
		if strings.Contains(s, "5522;") {
			if idx := strings.Index(s, "5522;"); idx != -1 {
				code = OSCSecureClipboard
				data = s[idx+len("5522;"):]
				data = strings.TrimPrefix(data, ":")
				data = strings.TrimRight(data, "\x1b\\\x07 ")
				e.initialized = true
			}
		}
	case *event.RawOscEvent:
		code = ev.Code
		data = ev.Data
		if code == OSCSecureClipboard {
			e.initialized = true
		}
	}

	if code != OSCSecureClipboard {
		return false, nil
	}

	slog.Info("KITTY: Processing handshake", "data", data)
	// Separate metadata from payload (payload follows the first ';')
	var metadata, payload string
	if parts := strings.SplitN(data, ";", 2); len(parts) == 2 {
		metadata = parts[0]
		payload = parts[1]
	} else {
		metadata = data
	}

	params := parseParams(metadata)
	typ := params[paramType]
	status := params[paramStatus]

	// Capture the one-time password sent by kitty in the paste notification.
	// Per the kitty spec the key is "pw" (not "password").
	if pw, ok := params[paramPw]; ok && pw != "" {
		e.password = pw
	}

	if typ != typeRead {
		return true, nil
	}

	switch status {
	case statusOK:
		// Step 1: Terminal indicates paste event or acknowledges read request.
		if e.requesting {
			// Acknowledgment of our read request; prepare buffer for chunks.
			e.buffer = nil
		} else {
			// Start of a paste notification/listing.
			e.availableMimes = nil
			e.requesting = false
			e.buffer = nil
		}
		return true, nil

	case statusData:
		mime := params[paramMime]
		if mime == "" {
			return true, nil
		}

		// DATA status is used for both chunk delivery and MIME-type notification.
		if payload != "" {
			if !e.requesting {
				// Step 2: MIME-type notification via requested listing (mime=".")
				// The payload is a space-separated list of base64-encoded MIME types.
				mimes := strings.Split(strings.TrimSpace(payload), " ")
				e.availableMimes = append(e.availableMimes, mimes...)
			} else {
				// Step 3: Chunk delivery — status=DATA:mime=<b64>;<b64_chunk>
				chunk, _ := base64.StdEncoding.DecodeString(payload)
				e.buffer = append(e.buffer, chunk...)
			}
		} else {
			// Step 2: MIME-type notification via unsolicited paste event (one per packet)
			e.availableMimes = append(e.availableMimes, mime)
		}
		return true, nil

	case statusDone:
		if !e.requesting && len(e.availableMimes) > 0 {
			// Step 2b: We received all available MIME types. Prioritize them.
			// We prefer rich content over plain text.
			priorities := []string{"image/png", "text/html", "text/plain"}
			var selectedMime string

			for _, pref := range priorities {
				for _, m := range e.availableMimes {
					mimeBytes, _ := base64.StdEncoding.DecodeString(m)
					if string(mimeBytes) == pref {
						selectedMime = m
						break
					}
				}
				if selectedMime != "" {
					break
				}
			}

			if selectedMime == "" && len(e.availableMimes) > 0 {
				selectedMime = e.availableMimes[0]
			}

			if selectedMime != "" {
				// Request the data using the selected MIME type and password.
				e.mime = selectedMime
				e.requesting = true

				// Standard metadata keys
				metadata := fmt.Sprintf("type=%s:id=%s", typeRead, tmuxId)
				if e.password != "" {
					// The spec requires setting the human name to "Paste event" (base64 encoded)
					// when using the one-time password from an unsolicited paste event.
					// Base64("Paste event") = UGFzdGUgZXZlbnQ=
					metadata += fmt.Sprintf(":pw=%s:name=UGFzdGUgZXZlbnQ=", e.password)
				}

				// The payload is the base64 encoded MIME type we want to read.
				// Documentation: <OSC>5522;type=read;<base 64 encoded space separated list of mime types to read><ST>
				e.write(fmt.Sprintf("\x1b]%d;%s;%s\x1b\\",
					OSCSecureClipboard,
					metadata,
					e.mime)) // e.mime is already base64 encoded
				return true, nil
			}
		}

		// Step 4: Transfer complete — emit a ClipboardEvent to the DOM.
		ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
		mimeTypeBytes, _ := base64.StdEncoding.DecodeString(e.mime)
		mimeType := string(mimeTypeBytes)
		if mimeType == "" {
			mimeType = fallbackMime
		}
		ce.Items[mimeType] = e.buffer

		// Reset state
		e.buffer = nil
		e.availableMimes = nil
		e.requesting = false

		return true, ce
	}

	return true, nil
}

func parseParams(data string) map[string]string {
	res := make(map[string]string)
	if data == "" {
		return res
	}
	pairs := strings.Split(data, ":")
	for _, p := range pairs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}
