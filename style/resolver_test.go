package style_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// fakeNode — minimal style.StyleNode implementation for tests
// ---------------------------------------------------------------------------

type fakeNode struct {
	rawStyle            style.Style
	elementDefaultStyle style.Style // optional per-element-type default
	computedStyle       *style.Computed
	dirtyStyle          bool
	dirtyChild          bool
	parent              *fakeNode
	firstChild          *fakeNode
	nextSibling         *fakeNode
	visited             bool // set by tests to track ResolveTree visits
}

func (n *fakeNode) RawStyle() style.Style              { return n.rawStyle }
func (n *fakeNode) ElementDefaultStyle() style.Style   { return n.elementDefaultStyle }
func (n *fakeNode) ComputedStyle() *style.Computed     { return n.computedStyle }
func (n *fakeNode) SetComputedStyle(c *style.Computed) { n.computedStyle = c; n.visited = true }
func (n *fakeNode) IsDirtyStyle() bool                 { return n.dirtyStyle }
func (n *fakeNode) HasDirtyStyleChild() bool           { return n.dirtyChild }
func (n *fakeNode) ClearDirtyStyle()                   { n.dirtyStyle = false }
func (n *fakeNode) ClearChildNeedsStyle()              { n.dirtyChild = false }

func (n *fakeNode) StyleParent() style.StyleNode {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *fakeNode) StyleFirstChild() style.StyleNode {
	if n.firstChild == nil {
		return nil
	}
	return n.firstChild
}
func (n *fakeNode) StyleNextSibling() style.StyleNode {
	if n.nextSibling == nil {
		return nil
	}
	return n.nextSibling
}

// markDirty simulates MarkDirty(DirtyStyle): sets dirtyStyle on self and
// dirtyChild on ancestors, like the real propagation in render.BaseRender.
func (n *fakeNode) markDirty() {
	n.dirtyStyle = true
	for p := n.parent; p != nil; p = p.parent {
		p.dirtyChild = true
	}
}

// appendChild links child as n's first child (simple single-child helper).
func (n *fakeNode) appendChild(child *fakeNode) {
	child.parent = n
	if n.firstChild == nil {
		n.firstChild = child
	} else {
		// Walk to end of sibling list.
		cur := n.firstChild
		for cur.nextSibling != nil {
			cur = cur.nextSibling
		}
		cur.nextSibling = child
	}
}

var _ style.StyleNode = (*fakeNode)(nil)

// ---------------------------------------------------------------------------
// TestResolver_DefaultsApplied
// ---------------------------------------------------------------------------

// TestResolver_DefaultsApplied verifies that an element with no author style
// receives the DefaultStyle baseline.
func TestResolver_DefaultsApplied(t *testing.T) {
	r := style.NewResolver()
	node := &fakeNode{dirtyStyle: true}

	got := r.Resolve(node, nil)
	want := style.DefaultStyle()

	if got.Display != want.Display {
		t.Errorf("Display = %v, want %v", got.Display, want.Display)
	}
	if got.Foreground != want.Foreground {
		t.Errorf("Foreground = %v, want %v", got.Foreground, want.Foreground)
	}
	if got.TextWrap != want.TextWrap {
		t.Errorf("TextWrap = %v, want %v", got.TextWrap, want.TextWrap)
	}
	if got.OverflowX != want.OverflowX {
		t.Errorf("OverflowX = %v, want %v", got.OverflowX, want.OverflowX)
	}
}

// ---------------------------------------------------------------------------
// TestResolver_InheritsColor
// ---------------------------------------------------------------------------

// TestResolver_InheritsColor verifies that Foreground (an inheritable
// property) flows from the parent Computed to the child when the child does
// not set it explicitly.
func TestResolver_InheritsColor(t *testing.T) {
	r := style.NewResolver()

	parentFG := color.RGBA{R: 100, G: 200, B: 50, A: 255}
	parentComputed := &style.Computed{
		Foreground: parentFG,
		Background: color.Transparent,
		TextWrap:   style.TextWrapWord,
	}

	child := &fakeNode{dirtyStyle: true}
	got := r.Resolve(child, parentComputed)

	if got.Foreground != parentFG {
		t.Errorf("Foreground = %v, want inherited %v", got.Foreground, parentFG)
	}
}

// ---------------------------------------------------------------------------
// TestResolver_DoesNotInheritWidth
// ---------------------------------------------------------------------------

// TestResolver_DoesNotInheritWidth verifies that Width (a non-inheritable
// property) is not carried from parent to child; the child gets the default.
func TestResolver_DoesNotInheritWidth(t *testing.T) {
	r := style.NewResolver()

	parentWidth := style.Cells(80)
	parentComputed := &style.Computed{
		Width:      parentWidth,
		Foreground: style.TerminalDefault,
		Background: color.Transparent,
	}

	child := &fakeNode{dirtyStyle: true}
	got := r.Resolve(child, parentComputed)

	wantWidth := style.DefaultStyle().Width
	if got.Width != wantWidth {
		t.Errorf("Width = %v, want default %v (not inherited %v)", got.Width, wantWidth, parentWidth)
	}
}

// ---------------------------------------------------------------------------
// TestResolver_OverlayWins
// ---------------------------------------------------------------------------

// TestResolver_OverlayWins verifies that an element's explicit style wins
// over the inherited value from the parent.
func TestResolver_OverlayWins(t *testing.T) {
	r := style.NewResolver()

	parentFG := color.RGBA{R: 100, G: 200, B: 50, A: 255}
	parentComputed := &style.Computed{
		Foreground: parentFG,
		Background: color.Transparent,
	}

	childFG := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	child := &fakeNode{
		dirtyStyle: true,
		rawStyle:   style.Style{Foreground: style.Some[color.Color](childFG)},
	}
	got := r.Resolve(child, parentComputed)

	if got.Foreground != childFG {
		t.Errorf("Foreground = %v, want element override %v", got.Foreground, childFG)
	}
}

// ---------------------------------------------------------------------------
// TestResolver_TerminalDefaultPreserved
// ---------------------------------------------------------------------------

// TestResolver_TerminalDefaultPreserved verifies that the TerminalDefault
// sentinel color passes through the resolver unchanged (symbolic colors are
// not resolved at style time; that happens at paint time in the backend).
func TestResolver_TerminalDefaultPreserved(t *testing.T) {
	r := style.NewResolver()

	node := &fakeNode{
		dirtyStyle: true,
		rawStyle:   style.Style{Foreground: style.Some(style.TerminalDefault)},
	}
	got := r.Resolve(node, nil)

	if got.Foreground != style.TerminalDefault {
		t.Errorf("Foreground = %v, want TerminalDefault sentinel", got.Foreground)
	}
}

// ---------------------------------------------------------------------------
// TestResolveTree_PhaseGated
// ---------------------------------------------------------------------------

// TestResolveTree_PhaseGated verifies that a completely clean subtree is not
// visited during ResolveTree.
func TestResolveTree_PhaseGated(t *testing.T) {
	r := style.NewResolver()

	// Tree:  root (dirty) → dirty child + clean child
	root := &fakeNode{}
	dirty := &fakeNode{}
	clean := &fakeNode{}

	root.appendChild(dirty)
	root.appendChild(clean)
	dirty.markDirty() // propagates dirtyChild to root

	style.ResolveTree(r, root)

	if !dirty.visited {
		t.Error("dirty child should have been visited")
	}
	if clean.visited {
		t.Error("clean child should NOT have been visited")
	}
}

// ---------------------------------------------------------------------------
// TestResolveTree_ClearsDirtyStyle
// ---------------------------------------------------------------------------

// TestResolveTree_ClearsDirtyStyle verifies that ResolveTree clears the
// DirtyStyle flag on every visited node.
func TestResolveTree_ClearsDirtyStyle(t *testing.T) {
	r := style.NewResolver()

	root := &fakeNode{}
	child := &fakeNode{}
	root.appendChild(child)
	child.markDirty()

	style.ResolveTree(r, root)

	if child.IsDirtyStyle() {
		t.Error("DirtyStyle should be cleared after ResolveTree")
	}
	if root.HasDirtyStyleChild() {
		t.Error("ChildNeedsStyle should be cleared after ResolveTree")
	}
}

// ---------------------------------------------------------------------------
// TestResolveTree_SkipsAfterClean
// ---------------------------------------------------------------------------

// TestResolveTree_SkipsAfterClean verifies that a second call to ResolveTree
// with no dirty marks set is a complete no-op (no nodes visited).
func TestResolveTree_SkipsAfterClean(t *testing.T) {
	r := style.NewResolver()

	root := &fakeNode{}
	child := &fakeNode{}
	root.appendChild(child)
	child.markDirty()

	// First pass: resolves and clears dirty flags.
	style.ResolveTree(r, root)
	child.visited = false // reset tracker

	// Second pass: nothing should be dirty, so nothing visited.
	style.ResolveTree(r, root)

	if child.visited {
		t.Error("second ResolveTree with no dirty marks should be a no-op")
	}
}
