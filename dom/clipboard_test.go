package dom

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
)

type mockClipboard struct {
	data string
}

func (m *mockClipboard) GetClipboard() string {
	return m.data
}

func (m *mockClipboard) SetClipboard(text string) {
	m.data = text
}

func (m *mockClipboard) RequestClipboard() {}

func TestDocument_ClipboardCopy(t *testing.T) {
	doc := NewDocument().(*document)
	t1 := doc.CreateTextNode("Hello Clipboard", nil)
	doc.AppendChild(t1)

	// Set up a selection
	rng := doc.CreateRange()
	rng.SetStart(t1, 6)
	rng.SetEnd(t1, 15) // "Clipboard"
	doc.Selection().AddRange(rng)

	cb := &mockClipboard{}
	ce := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy, cb)

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
	doc := NewDocument().(*document)
	cb := &mockClipboard{data: "pasted from system"}
	ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste, cb)

	// Initially event should have no items
	if len(ce.Items) != 0 {
		t.Fatal("expected new event to have no items")
	}

	// Manually dispatch to document handler
	doc.handlePaste(ce)

	if got := ce.Text(); got != "pasted from system" {
		t.Errorf("expected event text 'pasted from system', got %q", got)
	}
}

func TestDocument_ClipboardCopy_PrioritizesExistingData(t *testing.T) {
	doc := NewDocument().(*document)
	t1 := doc.CreateTextNode("Document Selection", nil)
	doc.AppendChild(t1)

	// Set document selection
	rng := doc.CreateRange()
	rng.SetStart(t1, 0)
	rng.SetEnd(t1, 8) // "Document"
	doc.Selection().AddRange(rng)

	cb := &mockClipboard{}
	ce := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy, cb)
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
