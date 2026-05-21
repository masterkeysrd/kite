package element_test

// Unit tests for TSK-025: TextAreaElement on UA Shadow Subtree.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

func TestTextArea_PublicChildren_HidesUANode(t *testing.T) {
	txa := element.TextArea("")

	count := 0
	for range txa.ChildNodes() {
		count++
	}
	if count != 0 {
		t.Errorf("ChildNodes count = %d, want 0", count)
	}
}

func TestTextArea_IntrinsicStyle_Properties(t *testing.T) {
	txa := element.TextArea("")
	is := txa.IntrinsicStyle()

	if !is.Display.IsSet() || is.Display.Value() != style.DisplayInlineBlock {
		t.Errorf("IntrinsicStyle.Display = %v, want DisplayInlineBlock", is.Display)
	}
	if !is.OverflowY.IsSet() || is.OverflowY.Value() != style.OverflowScroll {
		t.Errorf("IntrinsicStyle.OverflowY = %v, want OverflowScroll", is.OverflowY)
	}
	if !is.OverflowWrap.IsSet() || is.OverflowWrap.Value() != style.OverflowWrapBreakWord {
		t.Errorf("IntrinsicStyle.OverflowWrap = %v, want OverflowWrapBreakWord", is.OverflowWrap)
	}
	// OverflowX and WhiteSpace are no longer forced by the textarea host;
	// line breaks are handled by <br> elements in the UA subtree (HTML model).
	if is.OverflowX.IsSet() {
		t.Errorf("IntrinsicStyle.OverflowX should not be forced, got %v", is.OverflowX)
	}
	if is.WhiteSpace.IsSet() {
		t.Errorf("IntrinsicStyle.WhiteSpace should not be forced, got %v", is.WhiteSpace)
	}
}

func TestTextArea_IntrinsicStyle_Wins(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.TextArea("")
	txa.Style(style.Style{
		OverflowY: style.Some(style.OverflowVisible), // must lose to intrinsic Scroll
	})

	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	ro := txa.RenderObject()
	if ro == nil {
		t.Fatal("no render object")
	}
	cs := ro.ComputedStyle()
	// overflow-y: scroll must always be forced by the intrinsic style.
	if cs.OverflowY != style.OverflowScroll {
		t.Errorf("OverflowY = %v, want OverflowScroll (intrinsic must win)", cs.OverflowY)
	}
}

func TestTextArea_CursorState_Initial(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "hello")
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Initial cursor is at the end of "hello" because editor.NewBuffer puts it at end.
	cs := txa.CursorState()
	if cs.X != 5 || cs.Y != 0 {
		t.Errorf("CursorState = (%d, %d), want (5, 0)", cs.X, cs.Y)
	}
}

func TestTextArea_CursorState_AfterEnter(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "hi")
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Press Enter
	dispatchKeyDownTextArea(txa, key.Key{Code: key.KeyEnter})

	eng.Frame()

	cs := txa.CursorState()
	// After "hi\n", cursor should be at (0, 1)
	if cs.X != 0 || cs.Y != 1 {
		t.Errorf("CursorState after enter = (%d, %d), want (0, 1)", cs.X, cs.Y)
	}
}

func dispatchKeyDownTextArea(txa *element.TextAreaElement, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)
	path := []event.EventTarget{txa}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}
