package element_test

// Unit tests for TSK-025: TextAreaElement on UA Shadow Subtree.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
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
	if !is.OverflowY.IsSet() || is.OverflowY.Value() != style.OverflowAuto {
		t.Errorf("IntrinsicStyle.OverflowY = %v, want OverflowAuto", is.OverflowY)
	}
	if !is.OverflowWrap.IsSet() || is.OverflowWrap.Value() != style.OverflowWrapBreakWord {
		t.Errorf("IntrinsicStyle.OverflowWrap = %v, want OverflowWrapBreakWord", is.OverflowWrap)
	}
	if !is.OverflowX.IsSet() || is.OverflowX.Value() != style.OverflowClip {
		t.Errorf("IntrinsicStyle.OverflowX = %v, want OverflowClip", is.OverflowX)
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
		OverflowY: style.Some(style.OverflowVisible), // must lose to intrinsic Auto
	})

	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	ro := txa.RenderObject()
	if ro == nil {
		t.Fatal("no render object")
	}
	cs := ro.ComputedStyle()
	// overflow-y: auto must always be forced by the intrinsic style.
	if cs.OverflowY != style.OverflowAuto {
		t.Errorf("OverflowY = %v, want OverflowAuto (intrinsic must win)", cs.OverflowY)
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

func dispatchMouseDownTextArea(target event.EventTarget, x, y int) {
	ev := event.NewMouseEvent(event.EventMouseDown, layout.Point{X: x, Y: y}, event.ButtonLeft, 0)
	ev.Local = layout.Point{X: x, Y: y}
	path := []event.EventTarget{target}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

func TestTextArea_MouseDown_SetsCursor(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "line1\nline2")
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Click on line 1, 'l' (offset 0)
	dispatchMouseDownTextArea(txa, 0, 0)
	if off := txa.Buffer().ByteOffset(); off != 0 {
		t.Errorf("Click at (0,0) expected offset 0, got %d", off)
	}

	// Click on line 1, 'i' (offset 1)
	dispatchMouseDownTextArea(txa, 1, 0)
	if off := txa.Buffer().ByteOffset(); off != 1 {
		t.Errorf("Click at (1,0) expected offset 1, got %d", off)
	}

	// Click on line 2, 'l' (offset 6)
	// 'line1\n' is 6 bytes.
	dispatchMouseDownTextArea(txa, 0, 1)
	if off := txa.Buffer().ByteOffset(); off != 6 {
		t.Errorf("Click at (0,1) expected offset 6, got %d", off)
	}

	// Click on a third (empty) line - should be clamped to end of buffer
	dispatchMouseDownTextArea(txa, 0, 2)
	if off := txa.Buffer().ByteOffset(); off != 11 {
		t.Errorf("Click at (0,2) expected offset 11, got %d", off)
	}
}

func TestTextArea_WheelScroll_DoesNotSnapBack(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	defer eng.Stop()

	// Create a textarea with enough content to scroll.
	// Height 5, 10 lines of text.
	text := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	txa := element.NewTextArea(eng.Document(), text)
	// Put cursor at the start (line 0)
	txa.Buffer().SetOffset(0)
	root := element.Box(txa)
	eng.Mount(root)

	eng.Frame()

	// Precondition: scroll is at 0
	if _, y := txa.Scroll(); y != 0 {
		t.Fatalf("Initial scroll.Y should be 0, got %d", y)
	}

	// Focus the textarea
	fm := focus.NewManager(eng.Document(), event.NewDispatcher())
	fm.Focus(txa, 0)
	eng.Frame()

	// Simulate wheel scroll down (deltaY > 0)
	// Process wheel event directly on target
	ev := event.NewWheelEvent(layout.Point{X: 0, Y: 0}, 0, 2, 0)
	txa.OnWheel(ev)

	// Run another frame.
	eng.Frame()

	// Check if scroll is still at 2, or if it snapped back to 0.
	if _, y := txa.Scroll(); y != 2 {
		t.Errorf("After wheel scroll down, scroll.Y should be 2, got %d (likely snapped back by ScrollCursorIntoView)", y)
	}

	// Now type a character. This SHOULD trigger ScrollCursorIntoView and snap back to 0
	// (since cursor is at 0).
	dispatchKeyDownTextArea(txa, key.Key{Code: 'a', Text: "a"})
	eng.Frame()

	if _, y := txa.Scroll(); y != 0 {
		t.Errorf("After typing, scroll.Y should have snapped back to 0 to show cursor, got %d", y)
	}
}
