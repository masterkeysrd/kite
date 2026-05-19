package spatial_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/focus/spatial"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Minimal render.Object implementation for spatial tests
// ---------------------------------------------------------------------------

// spatialObj is a lightweight dom.Node for spatial tests.
type spatialObj struct {
	event.Target
	parent        *spatialObj
	children      []*spatialObj
	focusable     bool
	disabled      bool
	display       style.Display
	bounds        layout.Rect
	computedStyle *style.Computed // cached to avoid per-call allocs
	render        *spatialRender
}

type spatialRender struct {
	node *spatialObj
}

// newFocusable creates a focusable spatialObj at the given bounds.
func newFocusable(b layout.Rect) *spatialObj {
	obj := &spatialObj{
		focusable: true,
		display:   style.DisplayBlock,
		bounds:    b,
	}
	obj.render = &spatialRender{node: obj}
	return obj
}

// newContainer creates a non-focusable spatialObj (container / root).
func newContainer() *spatialObj {
	obj := &spatialObj{display: style.DisplayBlock}
	obj.render = &spatialRender{node: obj}
	return obj
}

// link appends child under parent in DOM order.
func link(parent, child *spatialObj) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// --- dom.Node interface ---

func (o *spatialObj) Kind() dom.Kind   { return dom.KindElement }
func (o *spatialObj) NodeName() string { return "test" }
func (o *spatialObj) Parent() dom.Node {
	if o.parent == nil {
		return nil
	}
	return o.parent
}
func (o *spatialObj) ParentElement() dom.Element { return nil }
func (o *spatialObj) NextSibling() dom.Node {
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
func (o *spatialObj) PreviousSibling() dom.Node {
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
func (o *spatialObj) OwnerDocument() dom.Document   { return nil }
func (o *spatialObj) IsConnected() bool             { return true }
func (o *spatialObj) RenderObject() render.Object   { return o.render }
func (o *spatialObj) SetRenderObject(render.Object) {}
func (o *spatialObj) AppendChild(n dom.Node) dom.Node {
	o.children = append(o.children, n.(*spatialObj))
	n.(*spatialObj).parent = o
	return n
}
func (o *spatialObj) InsertBefore(n, ref dom.Node) dom.Node             { return nil }
func (o *spatialObj) RemoveChild(n dom.Node) dom.Node                   { return nil }
func (o *spatialObj) ReplaceChild(newChild, oldChild dom.Node) dom.Node { return nil }
func (o *spatialObj) FirstChild() dom.Node {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[0]
}
func (o *spatialObj) LastChild() dom.Node {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[len(o.children)-1]
}
func (o *spatialObj) HasChildNodes() bool { return len(o.children) > 0 }
func (o *spatialObj) Contains(n dom.Node) bool {
	for cur := n; cur != nil; cur = cur.Parent() {
		if cur == o {
			return true
		}
	}
	return false
}
func (o *spatialObj) ChildNodes() iter.Seq[dom.Node] {
	return func(yield func(dom.Node) bool) {
		for _, c := range o.children {
			if !yield(c) {
				return
			}
		}
	}
}
func (o *spatialObj) Unwrap() dom.Node        { return nil }
func (o *spatialObj) TextContent() string     { return "" }
func (o *spatialObj) CloneNode(bool) dom.Node { return nil }
func (o *spatialObj) NeedsSync() bool         { return false }
func (o *spatialObj) ChildNeedsSync() bool    { return false }
func (o *spatialObj) MarkNeedsSync()          {}
func (o *spatialObj) ClearSyncFlags()         {}

// --- dom.Focusable and dom.Disableable ---

func (o *spatialObj) IsFocusable() bool { return o.focusable }
func (o *spatialObj) Focus()            {}
func (o *spatialObj) Blur()             {}
func (o *spatialObj) IsDisabled() bool  { return o.disabled }
func (o *spatialObj) SetDisabled(v bool) {
	o.disabled = v
}

// --- render.Object interface (spatialRender) ---

func (r *spatialRender) EventTarget() event.EventTarget { return r.node }
func (r *spatialRender) Parent() render.Object {
	if r.node.parent != nil {
		return r.node.parent.render
	}
	return nil
}
func (r *spatialRender) FirstChild() render.Object {
	if len(r.node.children) > 0 {
		return r.node.children[0].render
	}
	return nil
}
func (r *spatialRender) LastChild() render.Object {
	if len(r.node.children) > 0 {
		return r.node.children[len(r.node.children)-1].render
	}
	return nil
}
func (r *spatialRender) NextSibling() render.Object {
	if ns := r.node.NextSibling(); ns != nil {
		return ns.RenderObject()
	}
	return nil
}
func (r *spatialRender) PreviousSibling() render.Object {
	if ps := r.node.PreviousSibling(); ps != nil {
		return ps.RenderObject()
	}
	return nil
}
func (r *spatialRender) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {
		for _, c := range r.node.children {
			if !yield(c.render) {
				return
			}
		}
	}
}
func (r *spatialRender) InsertChild(child, before render.Object) {}
func (r *spatialRender) RemoveChild(child render.Object)         {}
func (r *spatialRender) ComputedStyle() *style.Computed {
	if r.node.computedStyle == nil {
		r.node.computedStyle = &style.Computed{Display: r.node.display}
	}
	return r.node.computedStyle
}
func (r *spatialRender) SetComputedStyle(s *style.Computed)     { r.node.computedStyle = s }
func (r *spatialRender) Flags() render.DirtyFlag                { return 0 }
func (r *spatialRender) MarkDirty(_ render.DirtyFlag)           {}
func (r *spatialRender) ClearDirty(_ render.DirtyFlag)          {}
func (r *spatialRender) MarkChildrenDirty()                     {}
func (r *spatialRender) ClearDirtyRecursive(_ render.DirtyFlag) {}
func (r *spatialRender) IsDetached() bool                       { return false }
func (r *spatialRender) RawStyle() style.Style                  { return style.Style{} }
func (r *spatialRender) DefaultStyle() style.Style              { return style.Style{} }
func (r *spatialRender) IsDirtyStyle() bool                     { return false }
func (r *spatialRender) HasDirtyStyleChild() bool               { return false }
func (r *spatialRender) ClearDirtyStyle()                       {}
func (r *spatialRender) ClearChildNeedsStyle()                  {}
func (r *spatialRender) StyleParent() style.StyleNode           { return r.Parent() }
func (r *spatialRender) StyleFirstChild() style.StyleNode       { return r.FirstChild() }
func (r *spatialRender) StyleNextSibling() style.StyleNode      { return r.NextSibling() }

// layout.Node implementation
func (r *spatialRender) Style() *style.Computed { return r.ComputedStyle() }
func (r *spatialRender) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		for _, c := range r.node.children {
			if !yield(c.render) {
				return
			}
		}
	}
}
func (r *spatialRender) IsDirtyLayout() bool { return false }
func (r *spatialRender) ClearDirtyLayout()   {}
func (r *spatialRender) Fragment() *layout.Fragment {
	// Build a mock fragment tree recursively so that layout.AbsoluteBounds works.
	frag := &layout.Fragment{
		Node: r,
		Size: r.node.bounds.Size,
	}
	for _, c := range r.node.children {
		cFrag := c.render.Fragment()
		// Convert absolute bounds back to relative offsets for the mock tree.
		offset := layout.Point{
			X: c.bounds.Origin.X - r.node.bounds.Origin.X,
			Y: c.bounds.Origin.Y - r.node.bounds.Origin.Y,
		}
		frag.Children = append(frag.Children, layout.FragmentLink{
			Offset:   offset,
			Fragment: cFrag,
		})
	}
	return frag
}
func (r *spatialRender) CachedLayout(space layout.ConstraintSpace) *layout.Fragment {
	// For testing, mock a fragment using the manually set bounds.
	return &layout.Fragment{
		Node: r,
		Size: r.node.bounds.Size,
	}
}
func (r *spatialRender) SetCachedLayout(layout.ConstraintSpace, *layout.Fragment) {}
func (r *spatialRender) CachedMinMaxSizes() (layout.MinMaxSizes, bool) {
	return layout.MinMaxSizes{}, false
}
func (r *spatialRender) SetCachedMinMaxSizes(layout.MinMaxSizes) {}
func (r *spatialRender) LogicalNode() any                        { return r.node }

var _ dom.Node = (*spatialObj)(nil)
var _ render.Object = (*spatialRender)(nil)

// ---------------------------------------------------------------------------
// Manager factory
// ---------------------------------------------------------------------------

// makeManager returns a focus.Manager with a no-op event dispatcher.
func makeManager(root *spatialObj) *focus.Manager {
	d := event.NewDispatcher()
	return focus.NewManager(root, d)
}

// rect is a convenience constructor for layout.Rect.
func rect(x, y, w, h int) layout.Rect {
	return layout.Rect{
		Origin: layout.Point{X: x, Y: y},
		Size:   layout.Size{Width: w, Height: h},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestNavigate_PicksClosestInDirection verifies that Navigate picks the
// nearest candidate (by primary axis distance) in the requested direction.
//
// Layout (each cell = 1):
//
//	[a]  [b]  [c]   — row at y=0..2
//	     [cur]      — current focus at y=5..7
func TestNavigate_PicksClosestInDirection(t *testing.T) {
	t.Parallel()

	root := newContainer()

	//         x   y   w  h
	cur := newFocusable(rect(4, 5, 3, 2))
	a := newFocusable(rect(0, 0, 3, 2))
	b := newFocusable(rect(4, 0, 3, 2)) // directly above, closest
	c := newFocusable(rect(8, 0, 3, 2))

	for _, n := range []*spatialObj{cur, a, b, c} {
		link(root, n)
	}

	m := makeManager(root)
	m.Focus(cur, focus.ReasonProgrammatic)

	if !spatial.Navigate(m, spatial.DirectionUp) {
		t.Fatal("Navigate returned false; expected true")
	}
	if m.Current() != dom.Node(b) {
		t.Errorf("Navigate Up: got %v, want b (directly above)", m.Current())
	}
}

// TestNavigate_OffAxisPenaltyAffectsTiebreaker verifies that when two
// candidates are equidistant on the primary axis, the one with less off-axis
// offset wins.
//
// Layout:
//
//	[left]          [right]    — both at y=0..2, primary distance = 3
//	      [cur]               — at y=5..7, x=5..7
//
// left  is far off-axis; right is close off-axis → right should win.
func TestNavigate_OffAxisPenaltyAffectsTiebreaker(t *testing.T) {
	t.Parallel()

	root := newContainer()

	cur := newFocusable(rect(5, 5, 2, 2))   // x=5..7, y=5..7
	left := newFocusable(rect(0, 0, 2, 2))  // x=0..2, y=0..2  — far off-axis
	right := newFocusable(rect(5, 0, 2, 2)) // x=5..7, y=0..2  — on-axis

	for _, n := range []*spatialObj{cur, left, right} {
		link(root, n)
	}

	m := makeManager(root)
	m.Focus(cur, focus.ReasonProgrammatic)

	if !spatial.Navigate(m, spatial.DirectionUp) {
		t.Fatal("Navigate returned false; expected true")
	}
	if m.Current() != dom.Node(right) {
		t.Errorf("off-axis penalty: got %v, want right (on-axis above)", m.Current())
	}
}

// TestNavigate_RejectsCandidatesBehind verifies that candidates behind the
// current focus (in the opposite direction) are excluded from the candidate
// set and Navigate returns false if they are the only options.
func TestNavigate_RejectsCandidatesBehind(t *testing.T) {
	t.Parallel()

	root := newContainer()

	cur := newFocusable(rect(5, 5, 2, 2))
	// Both candidates are below cur; navigating Up should find nothing.
	below1 := newFocusable(rect(0, 8, 2, 2))
	below2 := newFocusable(rect(5, 9, 2, 2))

	for _, n := range []*spatialObj{cur, below1, below2} {
		link(root, n)
	}

	m := makeManager(root)
	m.Focus(cur, focus.ReasonProgrammatic)

	if spatial.Navigate(m, spatial.DirectionUp) {
		t.Error("Navigate Up returned true; expected false (all candidates behind)")
	}
	if m.Current() != dom.Node(cur) {
		t.Errorf("focus should stay on cur; got %v", m.Current())
	}
}

// TestNavigate_NoCandidate_ReturnsFalse verifies that Navigate returns false
// and does not change focus when there are no focusable candidates in the
// requested direction.
func TestNavigate_NoCandidate_ReturnsFalse(t *testing.T) {
	t.Parallel()

	root := newContainer()

	// Only one focusable — navigating in any direction should fail.
	cur := newFocusable(rect(5, 5, 2, 2))
	link(root, cur)

	m := makeManager(root)
	m.Focus(cur, focus.ReasonProgrammatic)

	for _, dir := range []spatial.Direction{
		spatial.DirectionUp,
		spatial.DirectionDown,
		spatial.DirectionLeft,
		spatial.DirectionRight,
	} {
		moved := spatial.Navigate(m, dir)
		if moved {
			t.Errorf("Navigate(%v) returned true with a single focusable; expected false", dir)
		}
		if m.Current() != dom.Node(cur) {
			t.Errorf("focus changed unexpectedly; got %v, want cur", m.Current())
		}
	}
}

// TestNavigate_RespectsActiveScope verifies that Navigate only considers
// candidates within the active scope's subtree, ignoring focusables outside.
func TestNavigate_RespectsActiveScope(t *testing.T) {
	t.Parallel()

	root := newContainer()

	// outside is above cur but lives outside the modal scope.
	outside := newFocusable(rect(5, 0, 2, 2))

	modal := newContainer()
	cur := newFocusable(rect(5, 5, 2, 2))
	// inside is above cur and within the modal scope.
	inside := newFocusable(rect(5, 3, 2, 1))

	link(root, outside)
	link(root, modal)
	link(modal, cur)
	link(modal, inside)

	m := makeManager(root)
	// Push a scope rooted at modal so only modal subtree is considered.
	m.PushScope(&focus.Scope{Root: modal})
	m.Focus(cur, focus.ReasonProgrammatic)

	if !spatial.Navigate(m, spatial.DirectionUp) {
		t.Fatal("Navigate returned false; expected true (inside is above cur)")
	}
	if m.Current() != dom.Node(inside) {
		t.Errorf("Navigate Up: got %v, want inside (outside scope should be ignored)", m.Current())
	}
}

// TestNavigate_AllFourDirections exercises all four directions in a cross
// layout where each direction has exactly one candidate.
//
// Layout:
//
//	      [up]
//	[left][cur][right]
//	      [down]
func TestNavigate_AllFourDirections(t *testing.T) {
	t.Parallel()

	root := newContainer()

	cur := newFocusable(rect(4, 4, 2, 2))    // x=4..6, y=4..6
	up := newFocusable(rect(4, 0, 2, 3))     // x=4..6, y=0..3  — above
	down := newFocusable(rect(4, 7, 2, 3))   // x=4..6, y=7..10 — below
	leftN := newFocusable(rect(0, 4, 3, 2))  // x=0..3, y=4..6  — left
	rightN := newFocusable(rect(7, 4, 3, 2)) // x=7..10, y=4..6 — right

	for _, n := range []*spatialObj{cur, up, down, leftN, rightN} {
		link(root, n)
	}

	m := makeManager(root)

	cases := []struct {
		dir  spatial.Direction
		want *spatialObj
		name string
	}{
		{spatial.DirectionUp, up, "Up"},
		{spatial.DirectionDown, down, "Down"},
		{spatial.DirectionLeft, leftN, "Left"},
		{spatial.DirectionRight, rightN, "Right"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset focus to cur before each sub-test.
			m.Focus(cur, focus.ReasonProgrammatic)

			if !spatial.Navigate(m, tc.dir) {
				t.Fatalf("Navigate(%s) returned false; expected true", tc.name)
			}
			if m.Current() != dom.Node(tc.want) {
				t.Errorf("Navigate(%s): got %v, want %v", tc.name, m.Current(), tc.want)
			}
		})
	}
}
