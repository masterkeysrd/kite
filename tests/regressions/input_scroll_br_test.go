package regressions

// Regression tests for input cursor scrolling and textarea <br> model.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

// --- Input scroll regression tests -------------------------------------------

// TestInput_Regression_ScrollWhenTextOverflows verifies that when the user
// types more characters than the input's visible width, the input scrolls
// so that the cursor remains visible.
func TestInput_Regression_ScrollWhenTextOverflows(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.Style{
		Width: style.Some(style.Cells(5)), // 5 cells wide, no border/padding
	})
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	// Type 8 characters — more than the 5-cell visible area.
	for _, ch := range "12345678" {
		dispatchKeyToInput(inp, key.Key{Code: ch, Text: string(ch)})
	}
	eng.Frame()

	// After typing, the cursor should be visible inside the input's content box.
	if !b.Cursor.Visible {
		t.Error("cursor should be visible when input is focused and has text")
	}

	// Cursor X must be within [0, 4] (content box = 5 cells, X in [0,4]).
	if b.Cursor.X < 0 || b.Cursor.X >= 5 {
		t.Errorf("cursor X = %d, want in range [0,4] (input width=5)", b.Cursor.X)
	}

	// The input scroll offset must be positive (content was scrolled).
	sx, _ := inp.Scroll()
	if sx <= 0 {
		t.Errorf("scroll X = %d, want > 0 (content should have scrolled)", sx)
	}
}

// TestInput_Regression_CursorState_AccountsForBorderAndPadding verifies that
// CursorState correctly adds border+padding inset to the IFC-local cursor X.
func TestInput_Regression_CursorState_AccountsForBorderAndPadding(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "abc")
	inp.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Border:  style.SingleBorder().Some(),   // left border = 1
		Padding: style.Some(style.Edges(0, 1)), // 1-cell padding left/right
	})
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	inp.Buffer().MoveToStart()
	inp.SyncBuffer()
	eng.Frame()

	cs := inp.CursorState()
	// border.Left=1, padding.Left=1 → insetLeft=2. Cursor at offset 0 → X=2.
	if cs.X != 2 {
		t.Errorf("CursorState.X = %d, want 2 (border+padding offset)", cs.X)
	}

	inp.Buffer().MoveToEnd()
	inp.SyncBuffer()
	eng.Frame()

	cs = inp.CursorState()
	// Cursor at end of "abc" (3 chars) → X = 2 + 3 = 5.
	if cs.X != 5 {
		t.Errorf("CursorState.X at end = %d, want 5", cs.X)
	}
}

// --- TextArea <br> model regression tests ------------------------------------

// TestTextArea_Regression_EnterInsertsBr verifies that pressing Enter in a
// textarea inserts a line break that is reflected as separate lines in the
// rendered fragment (the <br> model).
func TestTextArea_Regression_EnterInsertsBr(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "hi")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Press Enter — should insert \n and create a new line.
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyEnter})
	eng.Frame()

	cs := txa.CursorState()
	// After "hi\n", cursor should be at (0, 1) — start of new line.
	// Note: CursorState now returns (insetLeft+cx, insetTop+cy). With no
	// border or padding, insetLeft=0 insetTop=0, so direct cx/cy.
	if cs.X != 0 || cs.Y != 1 {
		t.Errorf("CursorState after Enter = (%d, %d), want (0, 1)", cs.X, cs.Y)
	}
}

// TestTextArea_Regression_MultilineNavigation verifies that Up/Down navigation
// works correctly with the <br>-based model.
func TestTextArea_Regression_MultilineNavigation(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "abc\ndef")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Initial cursor is at end of "abc\ndef" (offset 7) → should be at (3, 1).
	cs := txa.CursorState()
	if cs.X != 3 || cs.Y != 1 {
		t.Errorf("initial cursor = (%d, %d), want (3, 1)", cs.X, cs.Y)
	}

	// Press Up → cursor should move to line 0 at the same column (3).
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()
	cs = txa.CursorState()
	if cs.X != 3 || cs.Y != 0 {
		t.Errorf("cursor after Up = (%d, %d), want (3, 0)", cs.X, cs.Y)
	}

	// Press Down → back to line 1.
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
	eng.Frame()
	cs = txa.CursorState()
	if cs.X != 3 || cs.Y != 1 {
		t.Errorf("cursor after Down = (%d, %d), want (3, 1)", cs.X, cs.Y)
	}
}

// TestTextArea_Regression_PublicChildrenHideUA verifies that the UA subtree
// (including the ua-div and br elements) is not visible via ChildNodes().
func TestTextArea_Regression_PublicChildrenHideUA(t *testing.T) {
	txa := element.TextArea("hello\nworld")
	count := 0
	for range txa.ChildNodes() {
		count++
	}
	if count != 0 {
		t.Errorf("ChildNodes count = %d, want 0 (UA subtree must be hidden)", count)
	}
}

// --- helper ------------------------------------------------------------------

func dispatchKeyToInput(inp *element.InputElement, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

// dispatchKeyToTarget sends a key down event to an arbitrary EventTarget by
// building the ancestor path (root -> target) and dispatching through the
// event system. This mirrors testenv.Environment.DispatchKey for engine-based
// tests that don't use the higher-level test environment.
func dispatchKeyToTarget(target event.EventTarget, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)

	var path []event.EventTarget
	curr := target
	for curr != nil {
		path = append(path, curr)
		if n, ok := curr.(dom.Node); ok {
			p := n.Parent()
			if p == nil {
				break
			}
			curr = p
		} else {
			break
		}
	}

	// Reverse to root -> target
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}
