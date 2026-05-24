package kitty

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/event"
)

func TestKittyExtension_Handshake(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	// Check if Init sent the enable sequence
	expectedEnable := fmt.Sprintf("\x1b[?%dh", ModeSecureClipboard)
	if out.String() != expectedEnable {
		t.Errorf("expected enable sequence, got %q", out.String())
	}
	out.Reset()

	// Step 1: Terminal sends password
	ext.HandleEvent(&event.RawOscEvent{
		Code: OSCSecureClipboard,
		Data: fmt.Sprintf("%s=%s:%s=%s:%s=cGFzc3dvcmQ=", paramType, typeRead, paramStatus, statusOK, paramPassword), // "password"
	})
	if ext.password != "cGFzc3dvcmQ=" {
		t.Errorf("expected password to be set, got %q", ext.password)
	}
	out.Reset() // Clear the persistent enable sequence triggered by HandleEvent

	// Step 2: Terminal sends available MIME type
	mimeType := base64.StdEncoding.EncodeToString([]byte("text/plain"))
	ext.HandleEvent(&event.RawOscEvent{
		Code: OSCSecureClipboard,
		Data: fmt.Sprintf("%s=%s:%s=%s:%s=%s", paramType, typeRead, paramStatus, statusData, paramMime, mimeType),
	})

	// Check if extension requested data for this MIME type
	expectedReq := fmt.Sprintf("\x1b]%d;%s=%s:%s=%s:%s=cGFzc3dvcmQ=\x1b\\",
		OSCSecureClipboard, paramType, typeRead, paramMime, mimeType, paramPassword)
	if out.String() != expectedReq {
		t.Errorf("expected request sequence, got %q", out.String())
	}
	out.Reset()

	// Step 3: Terminal sends data chunks
	chunk1 := base64.StdEncoding.EncodeToString([]byte("hello "))
	ext.HandleEvent(&event.RawOscEvent{
		Code: OSCSecureClipboard,
		Data: fmt.Sprintf("%s=%s:%s=%s:%s=%s;%s", paramType, typeRead, paramStatus, statusData, paramMime, mimeType, chunk1),
	})
	chunk2 := base64.StdEncoding.EncodeToString([]byte("world"))
	ext.HandleEvent(&event.RawOscEvent{
		Code: OSCSecureClipboard,
		Data: fmt.Sprintf("%s=%s:%s=%s:%s=%s;%s", paramType, typeRead, paramStatus, statusData, paramMime, mimeType, chunk2),
	})

	// Step 4: Terminal sends DONE
	handled, ev := ext.HandleEvent(&event.RawOscEvent{
		Code: OSCSecureClipboard,
		Data: fmt.Sprintf("%s=%s:%s=%s", paramType, typeRead, paramStatus, statusDone),
	})

	if !handled {
		t.Fatal("expected event to be handled")
	}
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		t.Fatalf("expected ClipboardEvent, got %T", ev)
	}
	if ce.ClipType != event.ClipboardPaste {
		t.Errorf("expected Paste type, got %v", ce.ClipType)
	}
	if got := ce.Text(); got != "hello world" {
		t.Errorf("expected text 'hello world', got %q", got)
	}
}
