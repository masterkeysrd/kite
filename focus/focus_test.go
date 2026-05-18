package focus_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testObject is a lightweight render.Object for focus tests. It records
// MarkDirty calls and exposes full focusable/disabled/display control.
type testObject struct {
	event.Target
	parent    *testObject
	children  []*testObject
	focusable bool
	disabled  bool
	display   style.Display
	dirty     int
}

// newFocusable returns a testObject that can receive focus (display=Block).
func newFocusable() *testObject {
	return &testObject{focusable: true, display: style.DisplayBlock}
}

// newNonFocusable returns a testObject that cannot receive focus.
func newNonFocusable() *testObject {
	return &testObject{focusable: false, display: style.DisplayBlock}
}

// link appends child to parent's children list (DOM order).
func link(parent, child *testObject) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// --- render.Object interface -------------------------------------------------

func (o *testObject) Parent() render.Object {
	if o.parent == nil {
		return nil
	}
	return o.parent
}

func (o *testObject) FirstChild() render.Object {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[0]
}

func (o *testObject) LastChild() render.Object {
	if n := len(o.children); n > 0 {
		return o.children[n-1]
	}
	return nil
}

func (o *testObject) NextSibling() render.Object {
	if o.parent == nil {
		return nil
	}
	for i, c := range o.parent.children {
		if c == o && i+1 < len(o.parent.children) {
			return o.parent.children[i+1]
		}
	}
	return nil
}

func (o *testObject) PreviousSibling() render.Object {
	if o.parent == nil {
		return nil
	}
	for i, c := range o.parent.children {
		if c == o && i > 0 {
			return o.parent.children[i-1]
		}
	}
	return nil
}

func (o *testObject) EventTarget() event.EventTarget { return o }

func (o *testObject) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {
		for _, c := range o.children {
			if !yield(c) {
				return
			}
		}
	}
}

func (o *testObject) LogicalNode() any                        { return nil }
func (o *testObject) MarkDetached()                           {}
func (o *testObject) IsDetached() bool                        { return false }
func (o *testObject) MarkChildrenDirty()                      {}
func (o *testObject) InsertChild(child, before render.Object) {}
func (o *testObject) RemoveChild(child render.Object)         {}
func (o *testObject) RawStyle() style.Style                   { return style.Style{} }
func (o *testObject) SetRawStyle(_ style.Style)               {}
func (o *testObject) Flags() render.DirtyFlag                 { return 0 }
func (o *testObject) MarkDirty(_ render.DirtyFlag)            { o.dirty++ }
func (o *testObject) ClearDirty(_ render.DirtyFlag)           {}
func (o *testObject) ClearDirtyRecursive(_ render.DirtyFlag)  {}
func (o *testObject) IsDirtySet(_ render.DirtyFlag) bool      { return false }
func (o *testObject) IsDirtyStyle() bool                      { return false }
func (o *testObject) IsDirtyLayout() bool                     { return false }
func (o *testObject) IsDirtyPaint() bool                      { return false }
func (o *testObject) IsDirtyScroll() bool                     { return false }
func (o *testObject) IsDirtyStructure() bool                  { return false }
func (o *testObject) Focusable() bool                         { return o.focusable }
func (o *testObject) SetFocusable(v bool)                     { o.focusable = v }
func (o *testObject) Disabled() bool                          { return o.disabled }
func (o *testObject) SetDisabled(v bool)                      { o.disabled = v }

func (o *testObject) Style() *style.Computed {
	return &style.Computed{Display: o.display}
}
func (o *testObject) ComputedStyle() *style.Computed {
	return o.Style()
}

func (o *testObject) SetComputedStyle(*style.Computed) {}

// StyleNode implementation
func (o *testObject) ElementDefaultStyle() style.Style  { return style.Style{} }
func (o *testObject) HasDirtyStyleChild() bool          { return false }
func (o *testObject) ClearDirtyStyle()                  {}
func (o *testObject) ClearChildNeedsStyle()             {}
func (o *testObject) StyleParent() style.StyleNode      { return o.Parent() }
func (o *testObject) StyleFirstChild() style.StyleNode  { return o.FirstChild() }
func (o *testObject) StyleNextSibling() style.StyleNode { return o.NextSibling() }

// layout.Node implementation
func (o *testObject) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {}
}
func (o *testObject) ClearDirtyLayout()                                        {}
func (o *testObject) Fragment() *layout.Fragment                               { return nil }
func (o *testObject) CachedLayout(layout.ConstraintSpace) *layout.Fragment     { return nil }
func (o *testObject) SetCachedLayout(layout.ConstraintSpace, *layout.Fragment) {}
func (o *testObject) CachedMinMaxSizes() (layout.MinMaxSizes, bool) {
	return layout.MinMaxSizes{}, false
}
func (o *testObject) SetCachedMinMaxSizes(layout.MinMaxSizes) {}

// compile-time interface check
var _ render.Object = (*testObject)(nil)

// ---------------------------------------------------------------------------
// Manager factory helpers
// ---------------------------------------------------------------------------

// makeManager returns a Manager and its event-capture infrastructure.
// root is the default scope root. captured accumulates dispatched focus
// event types in order.
func makeManager(root *testObject) (*focus.Manager, *[]event.EventType) {
	dispatcher := event.NewDispatcher()
	m := focus.NewManager(root, dispatcher)

	var captured []event.EventType
	for _, typ := range []event.EventType{
		event.EventFocus, event.EventBlur,
		event.EventFocusIn, event.EventFocusOut,
	} {
		t := typ
		root.AddEventListener(t, func(e event.Event) {
			captured = append(captured, e.Type())
		})
		root.AddEventListener(t, func(e event.Event) {
			captured = append(captured, e.Type())
		}, event.Capture())
	}
	return m, &captured
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestManager_SetFocus_EmitsFocusevent verifies that switching focus from one
// element to another emits blur/focusout on the old target and focus/focusin on
// the new target.
func TestManager_SetFocus_EmitsFocusevent(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	b := newFocusable()
	link(root, a)
	link(root, b)

	d := event.NewDispatcher()
	m := focus.NewManager(root, d)

	var fired []event.EventType
	for _, typ := range []event.EventType{event.EventFocusIn, event.EventFocusOut} {
		t2 := typ
		root.AddEventListener(t2, func(e event.Event) {
			fired = append(fired, e.Type())
		})
	}

	// Focus a → b: should emit focusout (bubbling) then focusin (bubbling).
	m.Focus(a, focus.ReasonProgrammatic)
	fired = nil // reset; we only care about the a→b transition

	m.Focus(b, focus.ReasonProgrammatic)

	hasFocusOut := false
	hasFocusIn := false
	for _, typ := range fired {
		if typ == event.EventFocusOut {
			hasFocusOut = true
		}
		if typ == event.EventFocusIn {
			hasFocusIn = true
		}
	}
	if !hasFocusOut {
		t.Error("expected focusout event on focus transition, got none")
	}
	if !hasFocusIn {
		t.Error("expected focusin event on focus transition, got none")
	}
}

// TestManager_Next_DOMOrderWithinScope verifies that Next walks focusable
// nodes in DOM tree order within the active scope.
func TestManager_Next_DOMOrderWithinScope(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	b := newFocusable()
	c := newFocusable()
	link(root, a)
	link(root, b)
	link(root, c)

	m, _ := makeManager(root)

	m.Next() // nil → a
	if m.Current() != a {
		t.Errorf("after 1st Next: got %v, want a", m.Current())
	}
	m.Next() // a → b
	if m.Current() != b {
		t.Errorf("after 2nd Next: got %v, want b", m.Current())
	}
	m.Next() // b → c
	if m.Current() != c {
		t.Errorf("after 3rd Next: got %v, want c", m.Current())
	}
	m.Next() // c → a (wrap)
	if m.Current() != a {
		t.Errorf("after wrap Next: got %v, want a", m.Current())
	}
}

// TestManager_Previous_ReverseOrder verifies that Previous walks focusable
// nodes in reverse DOM order, wrapping correctly.
func TestManager_Previous_ReverseOrder(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	b := newFocusable()
	c := newFocusable()
	link(root, a)
	link(root, b)
	link(root, c)

	m, _ := makeManager(root)

	m.Previous() // nil → c (last)
	if m.Current() != c {
		t.Errorf("after 1st Previous: got %v, want c", m.Current())
	}
	m.Previous() // c → b
	if m.Current() != b {
		t.Errorf("after 2nd Previous: got %v, want b", m.Current())
	}
	m.Previous() // b → a
	if m.Current() != a {
		t.Errorf("after 3rd Previous: got %v, want a", m.Current())
	}
	m.Previous() // a → c (wrap)
	if m.Current() != c {
		t.Errorf("after wrap Previous: got %v, want c", m.Current())
	}
}

// TestManager_SkipsNonFocusable verifies that non-focusable nodes are excluded
// from tab navigation.
func TestManager_SkipsNonFocusable(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	skip := newNonFocusable() // focusable == false
	b := newFocusable()
	link(root, a)
	link(root, skip)
	link(root, b)

	m, _ := makeManager(root)
	m.Focus(a, focus.ReasonProgrammatic)
	m.Next()

	if m.Current() != b {
		t.Errorf("Next should skip non-focusable; got %v, want b", m.Current())
	}
}

// TestManager_SkipsDisabled verifies that disabled nodes are excluded from
// tab navigation.
func TestManager_SkipsDisabled(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	disabled := newFocusable()
	disabled.disabled = true
	b := newFocusable()
	link(root, a)
	link(root, disabled)
	link(root, b)

	m, _ := makeManager(root)
	m.Focus(a, focus.ReasonProgrammatic)
	m.Next()

	if m.Current() != b {
		t.Errorf("Next should skip disabled node; got %v, want b", m.Current())
	}
}

// TestManager_SkipsDisplayNone verifies that display:none nodes are excluded
// from tab navigation.
func TestManager_SkipsDisplayNone(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	hidden := newFocusable()
	hidden.display = style.DisplayNone
	b := newFocusable()
	link(root, a)
	link(root, hidden)
	link(root, b)

	m, _ := makeManager(root)
	m.Focus(a, focus.ReasonProgrammatic)
	m.Next()

	if m.Current() != b {
		t.Errorf("Next should skip display:none node; got %v, want b", m.Current())
	}
}

// TestScope_PushCapturesPreviousFocus verifies that PushScope records the
// current focus in Scope.PreviousFocus.
func TestScope_PushCapturesPreviousFocus(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	link(root, a)

	modal := newNonFocusable()
	link(root, modal)

	m, _ := makeManager(root)
	m.Focus(a, focus.ReasonProgrammatic)

	s := &focus.Scope{Root: modal}
	m.PushScope(s)

	if s.PreviousFocus != a {
		t.Errorf("PushScope should capture previous focus; got %v, want a", s.PreviousFocus)
	}
}

// TestScope_PopRestoresFocus_ReasonRestore verifies that PopScope restores
// PreviousFocus with ReasonRestore.
func TestScope_PopRestoresFocus_ReasonRestore(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	link(root, a)

	modal := newNonFocusable()
	modalBtn := newFocusable()
	link(root, modal)
	link(modal, modalBtn)

	m, _ := makeManager(root)
	m.Focus(a, focus.ReasonProgrammatic)

	s := &focus.Scope{Root: modal}
	m.PushScope(s)
	m.Focus(modalBtn, focus.ReasonProgrammatic)

	m.PopScope()

	if m.Current() != a {
		t.Errorf("PopScope should restore previous focus; got %v, want a", m.Current())
	}
	if m.Reason() != focus.ReasonRestore {
		t.Errorf("PopScope should use ReasonRestore; got %v", m.Reason())
	}
}

// TestIsFocusVisible_OnlyKeyboardReason verifies that IsFocusVisible returns
// true only when focus was acquired via the keyboard.
func TestIsFocusVisible_OnlyKeyboardReason(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	a := newFocusable()
	link(root, a)

	m, _ := makeManager(root)

	m.Focus(a, focus.ReasonPointer)
	if m.IsFocusVisible(a) {
		t.Error("IsFocusVisible should be false for ReasonPointer")
	}

	m.Focus(a, focus.ReasonProgrammatic)
	if m.IsFocusVisible(a) {
		t.Error("IsFocusVisible should be false for ReasonProgrammatic")
	}

	m.Focus(a, focus.ReasonKeyboard)
	if !m.IsFocusVisible(a) {
		t.Error("IsFocusVisible should be true for ReasonKeyboard")
	}

	m.Focus(a, focus.ReasonRestore)
	if m.IsFocusVisible(a) {
		t.Error("IsFocusVisible should be false for ReasonRestore")
	}
}

// TestFocusFilter_RespectsScopeBoundary verifies that Focus returns false and
// Next skips objects outside the active scope.
func TestFocusFilter_RespectsScopeBoundary(t *testing.T) {
	t.Parallel()

	root := newNonFocusable()
	outside := newFocusable() // in root scope but not in inner scope
	modal := newNonFocusable()
	inside := newFocusable()
	link(root, outside)
	link(root, modal)
	link(modal, inside)

	m, _ := makeManager(root)
	// Focus outside first (in root scope).
	m.Focus(outside, focus.ReasonProgrammatic)

	// Push inner scope limited to modal subtree.
	s := &focus.Scope{Root: modal}
	m.PushScope(s)

	// Direct focus on outside should fail — it's not in the inner scope.
	ok := m.Focus(outside, focus.ReasonProgrammatic)
	if ok {
		t.Error("Focus on object outside active scope should return false")
	}

	// Next should only reach inside (the only focusable in the inner scope).
	m.Next()
	if m.Current() != inside {
		t.Errorf("Next within inner scope: got %v, want inside", m.Current())
	}
}
