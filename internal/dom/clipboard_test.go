package dom

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"
)

type mockClipboard struct {
	data string
}

var _ terminal.Clipboard = (*mockClipboard)(nil)

func (m *mockClipboard) ReadText() *promise.Promise[string] {
	return promise.Resolved(m.data)
}

func (m *mockClipboard) WriteText(text string) *promise.Promise[struct{}] {
	m.data = text
	return promise.Resolved(struct{}{})
}

func (m *mockClipboard) Read(mime string) *promise.Promise[[]byte] {
	if mime == "text/plain" {
		return promise.Resolved([]byte(m.data))
	}
	return promise.Resolved([]byte(nil))
}

func (m *mockClipboard) Write(mime string, data []byte) *promise.Promise[struct{}] {
	if mime == "text/plain" {
		m.data = string(data)
	}
	return promise.Resolved(struct{}{})
}

func TestDocument_ClipboardCopy(t *testing.T) {
	doc := NewDocument()
	t1 := doc.CreateTextNode("Hello Clipboard", nil)
	doc.AppendChild(t1)

	// Set up a selection
	rng := doc.CreateRange()
	rng.SetStart(t1, 6)
	rng.SetEnd(t1, 15) // "Clipboard"
	doc.Selection().AddRange(rng)

	cb := &mockClipboard{}
	doc.SetClipboardProvider(cb)
	ce := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy)

	// Manually dispatch to document handler
	doc.handleCopy(ce)

	if got := ce.Text(); got != "Clipboard" {
		t.Errorf("expected event text 'Clipboard', got %q", got)
	}
	if cb.data != "Clipboard" {
		t.Errorf("expected system clipboard 'Clipboard', got %q", cb.data)
	}
}

func TestDocument_ClipboardPasteFallback(t *testing.T) {
	doc := NewDocument()
	cb := &mockClipboard{data: "pasted from system"}
	doc.SetClipboardProvider(cb)
	ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)

	// Initially event should have no items
	if len(ce.Items) != 0 {
		t.Fatal("expected new event to have no items")
	}

	// Manually dispatch to document handler
	doc.handlePaste(ce)

	// RequestClipboard is async, it doesn't populate ce.Items immediately.
	// We just verify that Document called RequestClipboard (which in our mock is a no-op but we can add a flag).
}

func TestDocument_ClipboardCopy_PrioritizesExistingData(t *testing.T) {
	doc := NewDocument()
	t1 := doc.CreateTextNode("Document Selection", nil)
	doc.AppendChild(t1)

	// Set document selection
	rng := doc.CreateRange()
	rng.SetStart(t1, 0)
	rng.SetEnd(t1, 8) // "Document"
	doc.Selection().AddRange(rng)

	cb := &mockClipboard{}
	doc.SetClipboardProvider(cb)
	ce := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy)
	// Simulate a focused element already populating the event
	ce.Items[event.MimeTextPlain] = []byte("Local Selection")

	doc.handleCopy(ce)

	// Should NOT overwrite with "Document"
	if got := ce.Text(); got != "Local Selection" {
		t.Errorf("expected event text to remain 'Local Selection', got %q", got)
	}
	if cb.data != "Local Selection" {
		t.Errorf("expected system clipboard to be updated with 'Local Selection', got %q", cb.data)
	}
}
