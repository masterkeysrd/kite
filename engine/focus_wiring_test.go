package engine

// Unit tests for the two focus-wiring fixes:
//
//  1. Mousedown-to-focus: dispatchMouseEvent must call focusManager.Focus when
//     a mousedown lands on a focusable node and DefaultPrevented is false.
//
//  2. First-key auto-focus: dispatchKeyEvent must call focusManager.Next when
//     nothing is focused so the very first keystroke lands on the first
//     focusable element in DOM tree order rather than being silently dropped.
//
// Tests are in package engine (not engine_test) to access unexported fields
// (focusManager, dispatchKeyEvent, dispatchMouseEvent, processRawEvent),
// following the same pattern as cursor_test.go.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// focusableWiringElement is the minimal focusable DOM element stub reused
// across all focus-wiring unit tests.
type focusableWiringElement struct {
	dom.Element
}

func (f *focusableWiringElement) IsFocusable() bool { return true }
func (f *focusableWiringElement) Focus()            {}
func (f *focusableWiringElement) Blur()             {}
func (f *focusableWiringElement) Unwrap() dom.Node  { return f.Element }

// plantFocusable creates a focusable element owned by the engine's document,
// gives it a render object with a valid ComputedStyle and fragment, and
// positions it in the renderView's fragment tree at (x=0, y=yOffset) with
// size 10×1. The HitTest works correctly for (0, yOffset) after this call.
//
// The element is appended to the document as `fe` (not the inner el), so
// walkPreOrder / collectFocusable finds it via FirstChild/NextSibling and
// the dom.Focusable check passes.
func plantFocusable(eng *Engine, tag string, yOffset int) (*focusableWiringElement, *render.Box) {
	fe := &focusableWiringElement{}
	el := eng.document.CreateElement(tag, fe)
	fe.Element = el

	ro := render.NewBox(fe, fe)
	ro.SetComputedStyle(&style.Computed{Display: style.DisplayBlock})

	frag := &layout.Fragment{
		Node: ro,
		Size: geom.Size{Width: 10, Height: 1},
	}
	ro.SetCachedLayout(layout.ConstraintSpace{}, frag)

	el.SetRenderObject(ro)
	eng.renderView.InsertChild(ro, nil)

	// Append fe (not el) so that FirstChild() returns fe and the Focusable
	// interface check in IsFocusable succeeds.
	eng.document.AppendChild(fe)

	// Extend the renderView's cached fragment to include this child.
	prev := eng.renderView.Fragment()
	var prevChildren []layout.FragmentLink
	if prev != nil {
		prevChildren = prev.Children
	}
	newChildren := append(prevChildren, layout.FragmentLink{
		Offset:   geom.Point{X: 0, Y: yOffset},
		Fragment: frag,
	})
	eng.renderView.SetCachedLayout(layout.ConstraintSpace{}, &layout.Fragment{
		Node:     eng.renderView,
		Size:     geom.Size{Width: 80, Height: 24},
		Children: newChildren,
	})

	return fe, ro
}

// ---------------------------------------------------------------------------
// Bug 1 — Mousedown-to-focus
// ---------------------------------------------------------------------------

// TestDispatchMouseEvent_Mousedown_FocusesHitTarget verifies that a mousedown
// on a focusable element moves keyboard focus to that element with
// ReasonPointer. This is the primary regression test for Bug 1.
func TestDispatchMouseEvent_Mousedown_FocusesHitTarget(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	fe, _ := plantFocusable(eng, "input", 0)

	if eng.focusManager.Current() != nil {
		t.Fatal("precondition: nothing should be focused before the mousedown")
	}

	// A mousedown at (0,0) hits the element planted at y=0.
	eng.processRawEvent(&event.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})

	if eng.focusManager.Current() != fe {
		t.Errorf("after mousedown: focused = %v, want the pressed element", eng.focusManager.Current())
	}
	if eng.focusManager.Reason() != focus.ReasonPointer {
		t.Errorf("after mousedown: Reason() = %v, want ReasonPointer", eng.focusManager.Reason())
	}
}

// TestDispatchMouseEvent_NonFocusable_LeavesExistingFocus verifies that
// pressing a non-focusable element (a plain box with no dom.Focusable) does
// not steal focus from the currently focused element.
func TestDispatchMouseEvent_NonFocusable_LeavesExistingFocus(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	feInput, _ := plantFocusable(eng, "input", 0)

	// Plain non-focusable element at y=2.
	plainEl := eng.document.CreateElement("div", nil)
	plainRO := render.NewBox(plainEl, plainEl)
	plainRO.SetComputedStyle(&style.Computed{Display: style.DisplayBlock})
	plainFrag := &layout.Fragment{Node: plainRO, Size: geom.Size{Width: 10, Height: 1}}
	plainRO.SetCachedLayout(layout.ConstraintSpace{}, plainFrag)
	plainEl.SetRenderObject(plainRO)
	eng.renderView.InsertChild(plainRO, nil)
	eng.document.AppendChild(plainEl)
	prev := eng.renderView.Fragment()
	eng.renderView.SetCachedLayout(layout.ConstraintSpace{}, &layout.Fragment{
		Node: eng.renderView,
		Size: geom.Size{Width: 80, Height: 24},
		Children: append(prev.Children, layout.FragmentLink{
			Offset:   geom.Point{X: 0, Y: 2},
			Fragment: plainFrag,
		}),
	})

	// Focus the input first.
	if !eng.focusManager.Focus(feInput, focus.ReasonProgrammatic) {
		t.Fatal("precondition: could not focus input element")
	}

	// Mousedown on the plain div at y=2.
	eng.processRawEvent(&event.RawMouseEvent{X: 0, Y: 2, Button: event.ButtonLeft, Up: false})

	if eng.focusManager.Current() != feInput {
		t.Errorf("after mousedown on non-focusable: focused = %v, want original input element",
			eng.focusManager.Current())
	}
}

// TestDispatchMouseEvent_PreventDefault_DoesNotFocus verifies that when a
// mousedown listener calls ev.PreventDefault(), the engine skips the focus
// transfer, giving widgets a standard opt-out mechanism.
func TestDispatchMouseEvent_PreventDefault_DoesNotFocus(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	fe, _ := plantFocusable(eng, "input", 0)

	// Cancel the mousedown default action.
	fe.AddEventListener(event.EventMouseDown, func(e event.Event) {
		e.PreventDefault()
	})

	eng.processRawEvent(&event.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})

	if eng.focusManager.Current() != nil {
		t.Errorf("PreventDefault on mousedown: focused = %v, want nil (opt-out honoured)",
			eng.focusManager.Current())
	}
}

// TestDispatchMouseEvent_SwitchesFocus_BetweenTwoInputs verifies that
// pressing a second focusable element steals focus from the first.
func TestDispatchMouseEvent_SwitchesFocus_BetweenTwoInputs(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	first, _ := plantFocusable(eng, "input-first", 0)
	second, _ := plantFocusable(eng, "input-second", 2)

	// Focus the first element.
	eng.processRawEvent(&event.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})
	if eng.focusManager.Current() != first {
		t.Fatalf("precondition: could not focus first element, got %v", eng.focusManager.Current())
	}

	// Now press the second element (at y=2).
	eng.processRawEvent(&event.RawMouseEvent{X: 0, Y: 2, Button: event.ButtonLeft, Up: false})

	if eng.focusManager.Current() != second {
		t.Errorf("after pressing second element: focused = %v, want second element",
			eng.focusManager.Current())
	}
}

// ---------------------------------------------------------------------------
// Bug 2 — First-key auto-focus
// ---------------------------------------------------------------------------

// TestDispatchKeyEvent_AutoFocusFirst_WhenNothingFocused verifies that when no
// element is focused and a key event arrives, the engine auto-focuses the first
// focusable element in DOM tree order before dispatching the key.
func TestDispatchKeyEvent_AutoFocusFirst_WhenNothingFocused(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	fe, _ := plantFocusable(eng, "input", 0)

	if eng.focusManager.Current() != nil {
		t.Fatal("precondition: nothing should be focused")
	}

	eng.dispatchKeyEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))

	if eng.focusManager.Current() != fe {
		t.Errorf("after key with no focus: focused = %v, want first focusable element",
			eng.focusManager.Current())
	}
}

// TestDispatchKeyEvent_AutoFocusFirst_OrderIsDOM verifies that with multiple
// focusable elements, auto-focus selects the one that comes first in DOM
// tree order (depth-first pre-order), not the last.
func TestDispatchKeyEvent_AutoFocusFirst_OrderIsDOM(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	first, _ := plantFocusable(eng, "input-first", 0)
	_, _ = plantFocusable(eng, "input-second", 2)

	eng.dispatchKeyEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))

	if eng.focusManager.Current() != first {
		t.Errorf("auto-focus DOM order: focused = %v, want first element", eng.focusManager.Current())
	}
}

// TestDispatchKeyEvent_NoAutoFocus_WhenAlreadyFocused verifies that when an
// element already has focus, a keydown does not move focus to the first element.
func TestDispatchKeyEvent_NoAutoFocus_WhenAlreadyFocused(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	_, _ = plantFocusable(eng, "input-first", 0)
	second, _ := plantFocusable(eng, "input-second", 2)

	if !eng.focusManager.Focus(second, focus.ReasonProgrammatic) {
		t.Fatal("precondition: could not focus second element")
	}

	eng.dispatchKeyEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))

	if eng.focusManager.Current() != second {
		t.Errorf("keydown with existing focus: focused = %v, want second (unchanged)",
			eng.focusManager.Current())
	}
}

// TestDispatchKeyEvent_KeyDeliveredToAutoFocusedElement verifies that after
// auto-focusing, the key event is delivered to the newly focused element's
// listeners — not silently dropped.
func TestDispatchKeyEvent_KeyDeliveredToAutoFocusedElement(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	fe, _ := plantFocusable(eng, "input", 0)

	var received event.EventType
	fe.AddEventListener(event.EventKeyDown, func(e event.Event) {
		received = e.Type()
	})

	eng.dispatchKeyEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))

	if received != event.EventKeyDown {
		t.Errorf("auto-focused element did not receive the keydown: got %q", received)
	}
}

// TestDispatchKeyEvent_NoFocusable_FallsBackToDocument verifies that when no
// focusable element exists, a keydown is dispatched to the document (original
// fallback) rather than being dropped.
func TestDispatchKeyEvent_NoFocusable_FallsBackToDocument(t *testing.T) {
	b := mock.New(80, 24)
	eng := New(b, Options{})
	defer eng.Stop()

	var docReceived bool
	eng.document.AddEventListener(event.EventKeyDown, func(e event.Event) {
		docReceived = true
	})

	eng.dispatchKeyEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))

	if !docReceived {
		t.Error("with no focusable elements: document listener was not invoked")
	}
	if eng.focusManager.Current() != nil {
		t.Errorf("with no focusable elements: focused = %v, want nil", eng.focusManager.Current())
	}
}
