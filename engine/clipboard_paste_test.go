package engine

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"
)

type fakeMultiMIMEClipboard struct {
	items map[string][]byte
}

var _ terminal.Clipboard = (*fakeMultiMIMEClipboard)(nil)

func (f *fakeMultiMIMEClipboard) ReadText() *promise.Promise[string] {
	return promise.Resolved(string(f.items["text/plain"]))
}

func (f *fakeMultiMIMEClipboard) WriteText(text string) *promise.Promise[struct{}] {
	f.items["text/plain"] = []byte(text)
	return promise.Resolved(struct{}{})
}

func (f *fakeMultiMIMEClipboard) ListFormats() *promise.Promise[[]string] {
	var out []string
	for k := range f.items {
		out = append(out, k)
	}
	return promise.Resolved(out)
}

func (f *fakeMultiMIMEClipboard) Read(mime string) *promise.Promise[[]byte] {
	return promise.Resolved(f.items[mime])
}

func (f *fakeMultiMIMEClipboard) Write(mime string, data []byte) *promise.Promise[struct{}] {
	f.items[mime] = append([]byte(nil), data...)
	return promise.Resolved(struct{}{})
}

func newTestPasteEngine(t *testing.T, cb terminal.Clipboard) (*Engine, *element.TextAreaElement) {
	t.Helper()
	b := mock.New(80, 24)
	eng := New(b, Options{})
	doc := eng.Document()
	if cb != nil {
		doc.SetClipboardProvider(cb)
	}
	ta := element.TextArea("")
	doc.AppendChild(ta)
	eng.focusManager.SetFocus(ta, focus.ReasonProgrammatic)
	return eng, ta
}

func drainEngineTasks(eng *Engine) {
	for i := 0; i < 8; i++ {
		eng.scheduler.drainMicrotasks()
	}
}

func TestEngine_KeyDownPreventDefaultCancelsPaste(t *testing.T) {
	cb := &fakeMultiMIMEClipboard{
		items: map[string][]byte{"text/plain": []byte("should not paste")},
	}
	eng, ta := newTestPasteEngine(t, cb)

	pasteCount := 0
	ta.AddEventListener(event.EventKeyDown, func(e event.Event) {
		e.PreventDefault()
	})
	ta.AddEventListener(event.EventPaste, func(e event.Event) {
		pasteCount++
	})

	eng.processRawEvent(&backend.RawKeyEvent{Key: key.Key{Code: 'v', Mod: key.ModCtrl}})
	drainEngineTasks(eng)

	if pasteCount != 0 {
		t.Fatalf("paste listener fired %d times, want 0", pasteCount)
	}
	if got := ta.Value(); got != "" {
		t.Fatalf("textarea Value() = %q, want empty", got)
	}
}

func TestEngine_CtrlV_PreloadsMultiMIME_AndPreventDefaultCancelsText(t *testing.T) {
	cb := &fakeMultiMIMEClipboard{
		items: map[string][]byte{
			"image/png":  {0x89, 'P', 'N', 'G'},
			"text/plain": []byte("screenshot.png"),
		},
	}
	eng, ta := newTestPasteEngine(t, cb)

	var gotImage []byte
	var gotText string
	ta.AddEventListener(event.EventPaste, func(e event.Event) {
		ce, ok := e.(*event.ClipboardEvent)
		if !ok {
			return
		}
		gotImage = ce.Items["image/png"]
		gotText = ce.Text()
		if len(gotImage) > 0 {
			ce.PreventDefault()
		}
	})

	eng.processRawEvent(&backend.RawKeyEvent{Key: key.Key{Code: 'v', Mod: key.ModCtrl}})
	drainEngineTasks(eng)

	if string(gotImage) != "\x89PNG" {
		t.Fatalf("got image/png = %q, want PNG header", gotImage)
	}
	if gotText != "screenshot.png" {
		t.Fatalf("got text/plain = %q, want %q", gotText, "screenshot.png")
	}
	if got := ta.Value(); got != "" {
		t.Fatalf("textarea Value() = %q, want empty because PreventDefault was called", got)
	}
}

func TestEngine_BracketedPaste_EnrichesMultiMIME_AndPreventDefaultCancelsText(t *testing.T) {
	cb := &fakeMultiMIMEClipboard{
		items: map[string][]byte{
			"image/png":  {0x89, 'P', 'N', 'G'},
			"text/plain": []byte("screenshot.png"),
		},
	}
	eng, ta := newTestPasteEngine(t, cb)

	var gotImage []byte
	var gotText string
	ta.AddEventListener(event.EventPaste, func(e event.Event) {
		ce, ok := e.(*event.ClipboardEvent)
		if !ok {
			return
		}
		gotImage = ce.Items["image/png"]
		gotText = ce.Text()
		if len(gotImage) > 0 {
			ce.PreventDefault()
		}
	})

	eng.processRawEvent(&backend.RawBracketedPaste{Text: "screenshot.png"})
	drainEngineTasks(eng)

	if string(gotImage) != "\x89PNG" {
		t.Fatalf("got image/png = %q, want PNG header", gotImage)
	}
	if gotText != "screenshot.png" {
		t.Fatalf("got text/plain = %q, want %q", gotText, "screenshot.png")
	}
	if got := ta.Value(); got != "" {
		t.Fatalf("textarea Value() = %q, want empty because PreventDefault was called", got)
	}
}
