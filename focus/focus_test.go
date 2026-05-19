package focus_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testObject is a lightweight dom.Node for focus tests.
type testObject struct {
	event.Target
	parent    *testObject
	children  []*testObject
	focusable bool
	disabled  bool
	display   style.Display
	render    *testRender
}

type testRender struct {
	node  *testObject
	dirty int
}

// newFocusable returns a testObject that can receive focus (display=Block).
func newFocusable() *testObject {
	obj := &testObject{focusable: true, display: style.DisplayBlock}
	obj.render = &testRender{node: obj}
	return obj
}

// newNonFocusable returns a testObject that cannot receive focus.
func newNonFocusable() *testObject {
	obj := &testObject{focusable: false, display: style.DisplayBlock}
	obj.render = &testRender{node: obj}
	return obj
}

// link appends child to parent's children list (DOM order).
func link(parent, child *testObject) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// --- dom.Node interface ------------------------------------------------------

func (o *testObject) Kind() dom.Kind   { return dom.KindElement }
func (o *testObject) NodeName() string { return "test" }
func (o *testObject) Parent() dom.Node {
	if o.parent == nil {
		return nil
	}
	return o.parent
}
func (o *testObject) ParentElement() dom.Element { return nil }
func (o *testObject) NextSibling() dom.Node {
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
func (o *testObject) PreviousSibling() dom.Node {
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
func (o *testObject) OwnerDocument() dom.Document   { return nil }
func (o *testObject) IsConnected() bool             { return true }
func (o *testObject) RenderObject() render.Object   { return o.render }
func (o *testObject) SetRenderObject(render.Object) {}
func (o *testObject) AppendChild(n dom.Node) dom.Node {
	o.children = append(o.children, n.(*testObject))
	n.(*testObject).parent = o
	return n
}
func (o *testObject) InsertBefore(n, ref dom.Node) dom.Node             { return nil }
func (o *testObject) RemoveChild(n dom.Node) dom.Node                   { return nil }
func (o *testObject) ReplaceChild(newChild, oldChild dom.Node) dom.Node { return nil }
func (o *testObject) FirstChild() dom.Node {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[0]
}
func (o *testObject) LastChild() dom.Node {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[len(o.children)-1]
}
func (o *testObject) HasChildNodes() bool { return len(o.children) > 0 }
func (o *testObject) Contains(n dom.Node) bool {
	for cur := n; cur != nil; cur = cur.Parent() {
		if cur == o {
			return true
		}
	}
	return false
}
func (o *testObject) ChildNodes() iter.Seq[dom.Node] {
	return func(yield func(dom.Node) bool) {
		for _, c := range o.children {
			if !yield(c) {
				return
			}
		}
	}
}
func (o *testObject) Unwrap() dom.Node        { return nil }
func (o *testObject) TextContent() string     { return "" }
func (o *testObject) CloneNode(bool) dom.Node { return nil }
func (o *testObject) NeedsSync() bool         { return false }
func (o *testObject) ChildNeedsSync() bool    { return false }
func (o *testObject) MarkNeedsSync()          {}
func (o *testObject) ClearSyncFlags()         {}

// --- dom.Focusable and dom.Disableable ---------------------------------------

func (o *testObject) IsFocusable() bool { return o.focusable }
func (o *testObject) Focus()            {}
func (o *testObject) Blur()             {}
func (o *testObject) IsDisabled() bool  { return o.disabled }
func (o *testObject) SetDisabled(v bool) {
	o.disabled = v
}

// --- render.Object interface (testRender) ------------------------------------

func (r *testRender) EventTarget() event.EventTarget { return r.node }
func (r *testRender) Parent() render.Object {
	if r.node.parent != nil {
		return r.node.parent.render
	}
	return nil
}
func (r *testRender) FirstChild() render.Object {
	if len(r.node.children) > 0 {
		return r.node.children[0].render
	}
	return nil
}
func (r *testRender) LastChild() render.Object {
	if len(r.node.children) > 0 {
		return r.node.children[len(r.node.children)-1].render
	}
	return nil
}
func (r *testRender) NextSibling() render.Object {
	if ns := r.node.NextSibling(); ns != nil {
		return ns.RenderObject()
	}
	return nil
}
func (r *testRender) PreviousSibling() render.Object {
	if ps := r.node.PreviousSibling(); ps != nil {
		return ps.RenderObject()
	}
	return nil
}
func (r *testRender) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {
		for _, c := range r.node.children {
			if !yield(c.render) {
				return
			}
		}
	}
}
func (r *testRender) InsertChild(child, before render.Object) {}
func (r *testRender) RemoveChild(child render.Object)         {}
func (r *testRender) ComputedStyle() *style.Computed {
	return &style.Computed{Display: r.node.display}
}
func (r *testRender) SetComputedStyle(*style.Computed)     {}
func (r *testRender) Flags() render.DirtyFlag              { return 0 }
func (r *testRender) MarkDirty(f render.DirtyFlag)         { r.dirty++ }
func (r *testRender) ClearDirty(render.DirtyFlag)          {}
func (r *testRender) MarkChildrenDirty()                   {}
func (r *testRender) ClearDirtyRecursive(render.DirtyFlag) {}
func (r *testRender) IsDetached() bool                     { return false }

func (r *testRender) RawStyle() style.Style              { return style.Style{} }
func (r *testRender) SetRawStyle(style.Style)            {}
func (r *testRender) ElementDefaultStyle() style.Style   { return style.Style{} }
func (r *testRender) SetElementDefaultStyle(style.Style) {}
func (r *testRender) IsDirtyStyle() bool                 { return false }
func (r *testRender) HasDirtyStyleChild() bool           { return false }
func (r *testRender) ClearDirtyStyle()                   {}
func (r *testRender) ClearChildNeedsStyle()              {}
func (r *testRender) StyleParent() style.StyleNode       { return r.Parent() }
func (r *testRender) StyleFirstChild() style.StyleNode   { return r.FirstChild() }
func (r *testRender) StyleNextSibling() style.StyleNode  { return r.NextSibling() }

// layout.Node implementation
func (r *testRender) Style() *style.Computed { return r.ComputedStyle() }
func (r *testRender) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		for _, c := range r.node.children {
			if !yield(c.render) {
				return
			}
		}
	}
}
func (r *testRender) IsDirtyLayout() bool                                      { return false }
func (r *testRender) ClearDirtyLayout()                                        {}
func (r *testRender) Fragment() *layout.Fragment                               { return nil }
func (r *testRender) CachedLayout(layout.ConstraintSpace) *layout.Fragment     { return nil }
func (r *testRender) SetCachedLayout(layout.ConstraintSpace, *layout.Fragment) {}
func (r *testRender) CachedMinMaxSizes() (layout.MinMaxSizes, bool) {
	return layout.MinMaxSizes{}, false
}
func (r *testRender) SetCachedMinMaxSizes(layout.MinMaxSizes) {}
func (r *testRender) LogicalNode() any                        { return r.node }

var _ dom.Node = (*testObject)(nil)
var _ render.Object = (*testRender)(nil)

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
