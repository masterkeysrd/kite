package spatial_test

import (
	"iter"
	"testing"

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

// spatialObj is a lightweight render.Object that lets tests set exact bounds
// and focusability.
type spatialObj struct {
	parent        *spatialObj
	children      []*spatialObj
	focusable     bool
	disabled      bool
	display       style.Display
	bounds        layout.Rect
	computedStyle *style.Computed // cached to avoid per-call allocs
}

// newFocusable creates a focusable spatialObj at the given bounds.
func newFocusable(b layout.Rect) *spatialObj {
	return &spatialObj{
		focusable: true,
		display:   style.DisplayBlock,
		bounds:    b,
	}
}

// newContainer creates a non-focusable spatialObj (container / root).
func newContainer() *spatialObj {
	return &spatialObj{display: style.DisplayBlock}
}

// link appends child under parent in DOM order.
func link(parent, child *spatialObj) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// --- render.Object interface ---

func (o *spatialObj) Parent() render.Object {
	if o.parent == nil {
		return nil
	}
	return o.parent
}

func (o *spatialObj) FirstChild() render.Object {
	if len(o.children) == 0 {
		return nil
	}
	return o.children[0]
}

func (o *spatialObj) LastChild() render.Object {
	if n := len(o.children); n > 0 {
		return o.children[n-1]
	}
	return nil
}

func (o *spatialObj) NextSibling() render.Object {
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

func (o *spatialObj) PreviousSibling() render.Object {
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

func (o *spatialObj) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {
		for _, c := range o.children {
			if !yield(c) {
				return
			}
		}
	}
}

func (o *spatialObj) Bounds() layout.Rect                 { return o.bounds }
func (o *spatialObj) SetBounds(r layout.Rect)             { o.bounds = r }
func (o *spatialObj) LogicalNode() any                    { return nil }
func (o *spatialObj) MarkDetached()                       {}
func (o *spatialObj) IsDetached() bool                    { return false }
func (o *spatialObj) MarkChildrenDirty()                  {}
func (o *spatialObj) RawStyle() style.Style               { return style.Style{} }
func (o *spatialObj) SetRawStyle(_ style.Style)           {}
func (o *spatialObj) Flags() render.DirtyFlag             { return 0 }
func (o *spatialObj) MarkDirty(_ render.DirtyFlag)        {}
func (o *spatialObj) ClearDirty(_ render.DirtyFlag)       {}
func (o *spatialObj) IsDirtySet(_ render.DirtyFlag) bool  { return false }
func (o *spatialObj) IsDirtyStyle() bool                  { return false }
func (o *spatialObj) IsDirtyLayout() bool                 { return false }
func (o *spatialObj) IsDirtyPaint() bool                  { return false }
func (o *spatialObj) IsDirtyScroll() bool                 { return false }
func (o *spatialObj) IsDirtyStructure() bool              { return false }
func (o *spatialObj) LayoutFlags() render.LayoutFlag      { return 0 }
func (o *spatialObj) SetLayoutFlag(_ render.LayoutFlag)   {}
func (o *spatialObj) ClearLayoutFlag(_ render.LayoutFlag) {}
func (o *spatialObj) Focusable() bool                     { return o.focusable }
func (o *spatialObj) SetFocusable(v bool)                 { o.focusable = v }
func (o *spatialObj) Disabled() bool                      { return o.disabled }
func (o *spatialObj) SetDisabled(v bool)                  { o.disabled = v }

func (o *spatialObj) ComputedStyle() *style.Computed {
	if o.computedStyle == nil {
		o.computedStyle = &style.Computed{Display: o.display}
	}
	return o.computedStyle
}

func (o *spatialObj) SetComputedStyle(_ *style.Computed) {}

// compile-time check
var _ render.Object = (*spatialObj)(nil)

// ---------------------------------------------------------------------------
// Manager factory
// ---------------------------------------------------------------------------

// makeManager returns a focus.Manager with a no-op event dispatcher.
func makeManager(root *spatialObj) *focus.Manager {
	resolver := func(_ render.Object) *event.EventTarget { return nil }
	d := event.NewDispatcher(resolver)
	return focus.NewManager(root, d, resolver)
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
	if m.Current() != b {
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
	if m.Current() != right {
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
	if m.Current() != cur {
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
		if m.Current() != cur {
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
	if m.Current() != inside {
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
			if m.Current() != tc.want {
				t.Errorf("Navigate(%s): got %v, want %v", tc.name, m.Current(), tc.want)
			}
		})
	}
}
