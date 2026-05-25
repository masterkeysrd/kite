package kitty

import (
	"bytes"
	"testing"

	"github.com/masterkeysrd/kite/event"
)

func TestKittyExtension_Init(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	expected := "\x1b[?5522h\x1b[?5522$p"
	if out.String() != expected {
		t.Errorf("expected %q, got %q", expected, out.String())
	}
}

func TestKittyExtension_TmuxWrapping(t *testing.T) {
	t.Setenv("TMUX", "1")
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	// Wrapping: \x1bPtmux;\x1b + doubled ESC + \x1b\\
	expected := "\x1bPtmux;\x1b\x1b[?5522h\x1b\x1b[?5522$p\x1b\\"
	if out.String() != expected {
		t.Errorf("expected %q, got %q", expected, out.String())
	}
}

func TestKittyExtension_CapabilityReport(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	// Capability report should mark as initialized and be handled
	// Test various forms seen in the wild/tmux
	reports := []string{
		"CSI:?5522;1$y",
		"?5522;1$y",
		"\x1b[?5522;1$y",
		"5522;1$y",
		" ?5522;1$y ",         // With spaces
		"CSI:?5522;1$y\x1b\\", // With terminator
	}

	for _, r := range reports {
		ext.initialized = false
		handled, ev := ext.HandleEvent(&event.RawUnknownEvent{
			Payload: r,
		})

		if !handled {
			t.Errorf("expected capability report %q to be handled", r)
		}
		if ev != nil {
			t.Errorf("expected no event from capability report %q, got %v", r, ev)
		}
		if !ext.initialized {
			t.Errorf("expected initialized to be true for report %q", r)
		}
	}
}

func TestKittyExtension_Handshake(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)
	out.Reset()

	// 1. Initial password notification
	handled, ev := ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=OK:pw=cGFzc3dvcmQ=",
	})
	if !handled || ev != nil {
		t.Errorf("expected OK to be handled without event")
	}
	// Clear the lazy init sequences from the buffer
	out.Reset()

	// 2. MIME type notification
	handled, ev = ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=DATA:mime=dGV4dC9wbGFpbg==",
	})
	if !handled || ev != nil {
		t.Errorf("expected DATA (mime notification) to be handled without event")
	}

	// Check that read request was NOT YET sent (waiting for DONE)
	if out.Len() > 0 {
		t.Errorf("expected no read request yet, got %q", out.String())
	}

	// 3. DONE (finish mime listing -> trigger read request)
	handled, ev = ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=DONE",
	})
	if !handled || ev != nil {
		t.Fatalf("expected DONE to emit NO event yet, got handled=%v, ev=%v", handled, ev)
	}

	expectedReq := "\x1b]5522;type=read:id=kite:pw=cGFzc3dvcmQ=:name=UGFzdGUgZXZlbnQ=;dGV4dC9wbGFpbg==\x1b\\"
	if out.String() != expectedReq {
		t.Errorf("expected read request %q, got %q", expectedReq, out.String())
	}
	out.Reset()

	// 3b. Read request acknowledgment
	handled, ev = ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=OK",
	})
	if !handled || ev != nil {
		t.Errorf("expected OK acknowledgment to be handled without event")
	}
	if !ext.requesting {
		t.Error("expected ext.requesting to remain true after OK acknowledgment")
	}

	// 4. Chunk delivery
	handled, ev = ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=DATA:mime=dGV4dC9wbGFpbg==;aGVsbG8gd29ybGQ=",
	})
	if !handled || ev != nil {
		t.Errorf("expected DATA (chunk) to be handled without event")
	}

	// 5. Final DONE (finish transfer -> emit event)
	handled, ev = ext.HandleEvent(&event.RawOscEvent{
		Code: 5522,
		Data: "type=read:status=DONE",
	})
	if !handled || ev == nil {
		t.Fatalf("expected DONE to emit event, got handled=%v, ev=%v", handled, ev)
	}

	ce, ok := ev.(*event.ClipboardEvent)
	if !ok || ce.ClipType != event.ClipboardPaste {
		t.Fatalf("expected ClipboardEvent with ClipType ClipboardPaste, got %T", ev)
	}
	if string(ce.Items["text/plain"]) != "hello world" {
		t.Errorf("expected buffer 'hello world', got %q", ce.Items["text/plain"])
	}
}

func TestKittyExtension_BracketedPasteFallback(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	ext.HandleEvent(&event.RawBracketedPaste{Text: "fallback"})
	if !ext.initialized {
		t.Error("expected initialized to be true upon bracketed paste")
	}
}

func TestKittyExtension_UnknownEventParsing(t *testing.T) {
	ext := NewExtension()
	var out bytes.Buffer
	ext.Init(&out)

	handled, ev := ext.HandleEvent(&event.RawUnknownEvent{
		Payload: "OSC:5522;type=read:status=OK\x07",
	})
	if !handled || ev != nil {
		t.Errorf("expected OK via unknown event to be handled")
	}
	if !ext.initialized {
		t.Error("expected initialized to be true")
	}
}
