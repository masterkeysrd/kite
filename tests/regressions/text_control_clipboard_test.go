// Regression tests for textControlBase clipboard mechanics — covers TSK-051.
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
)

func TestTextControl_Copy(t *testing.T) {
	inp := element.Input("hello world")
	// Select "hello"
	inp.SetSelectionRange(0, 5)

	ce := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy)
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ce, path)

	if got := ce.Text(); got != "hello" {
		t.Errorf("Copy text = %q, want %q", got, "hello")
	}
	if !ce.DefaultPrevented() {
		t.Error("expected DefaultPrevented to be true after Copy")
	}
}

func TestTextControl_Cut(t *testing.T) {
	inp := element.Input("hello world")
	// Select "hello "
	inp.SetSelectionRange(0, 6)

	inputFired := false
	inp.OnEvent(event.EventInput, func(e event.Event) {
		inputFired = true
		if got := e.(*event.InputEvent).Value; got != "world" {
			t.Errorf("InputEvent value = %q, want %q", got, "world")
		}
	})

	ce := event.NewClipboardEvent(event.EventCut, event.ClipboardCut)
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ce, path)

	if got := ce.Text(); got != "hello " {
		t.Errorf("Cut text = %q, want %q", got, "hello ")
	}
	if got := inp.Value(); got != "world" {
		t.Errorf("Value after Cut = %q, want %q", got, "world")
	}
	if !inputFired {
		t.Error("expected EventInput to fire after Cut")
	}
	if !ce.DefaultPrevented() {
		t.Error("expected DefaultPrevented to be true after Cut")
	}
}

func TestTextControl_Paste(t *testing.T) {
	inp := element.Input("hello ")
	inp.Buffer().MoveToEnd()
	inp.SyncBuffer()

	inputFired := false
	inp.OnEvent(event.EventInput, func(e event.Event) {
		inputFired = true
		if got := e.(*event.InputEvent).Value; got != "hello world" {
			t.Errorf("InputEvent value = %q, want %q", got, "hello world")
		}
	})

	ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
	ce.SetText("world")
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ce, path)

	if got := inp.Value(); got != "hello world" {
		t.Errorf("Value after Paste = %q, want %q", got, "hello world")
	}
	if !inputFired {
		t.Error("expected EventInput to fire after Paste")
	}
	if !ce.DefaultPrevented() {
		t.Error("expected DefaultPrevented to be true after Paste")
	}
}

func TestTextArea_Cut(t *testing.T) {
	txa := element.TextArea("line 1\nline 2")
	// Select "line 1\n"
	txa.SetSelectionRange(0, 7)

	ce := event.NewClipboardEvent(event.EventCut, event.ClipboardCut)
	path := []event.EventTarget{txa}
	d := event.NewDispatcher()
	d.Dispatch(ce, path)

	if got := ce.Text(); got != "line 1\n" {
		t.Errorf("Cut text = %q, want %q", got, "line 1\n")
	}
	if got := txa.Value(); got != "line 2" {
		t.Errorf("Value after Cut = %q, want %q", got, "line 2")
	}
}
