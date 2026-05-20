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
	intrinsicStyle      style.Style // optional UA-forced intrinsic style
	computedStyle       *style.Computed
	dirtyStyle          bool
	dirtyChild          bool
	parent              *fakeNode
	firstChild          *fakeNode
	nextSibling         *fakeNode
	visited             bool // set by tests to track ResolveTree visits
}

func (n *fakeNode) RawStyle() style.Style              { return n.rawStyle }
func (n *fakeNode) DefaultStyle() style.Style          { return n.elementDefaultStyle }
func (n *fakeNode) IntrinsicStyle() style.Style        { return n.intrinsicStyle }
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

	// Prime the tree first so everyone has a computed style
	style.ResolveTree(r, root)
	dirty.visited = false
	clean.visited = false

	// Now mark dirty again and verify gating
	dirty.markDirty()
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

// ---------------------------------------------------------------------------
// TestResolver_FlexAndOrder
// ---------------------------------------------------------------------------

func TestResolver_FlexAndOrder(t *testing.T) {
	r := style.NewResolver()

	// 1. Default values
	node1 := &fakeNode{dirtyStyle: true}
	got1 := r.Resolve(node1, nil)
	if got1.Order != 0 {
		t.Errorf("Order = %v, want 0", got1.Order)
	}
	if got1.Flex.Grow != 0 {
		t.Errorf("Flex.Grow = %v, want 0", got1.Flex.Grow)
	}

	// 2. Explicit values
	node2 := &fakeNode{
		dirtyStyle: true,
		rawStyle: style.Style{
			Order: style.Some(5),
			Flex:  style.Some(style.Flex(2, 3)),
			Gap:   style.Some(style.Gap(1, 2)),
		},
	}
	got2 := r.Resolve(node2, nil)
	if got2.Order != 5 {
		t.Errorf("Order = %v, want 5", got2.Order)
	}
	if got2.Flex.Grow != 2 {
		t.Errorf("Flex.Grow = %v, want 2", got2.Flex.Grow)
	}
	if got2.Flex.Shrink != 3 {
		t.Errorf("Flex.Shrink = %v, want 3", got2.Flex.Shrink)
	}
	if got2.Gap.Row != 1 || got2.Gap.Column != 2 {
		t.Errorf("Gap = %+v, want {Row: 1, Column: 2}", got2.Gap)
	}
}

// ---------------------------------------------------------------------------
// TestResolver_BackgroundNotInherited
// ---------------------------------------------------------------------------

func TestResolver_BackgroundNotInherited(t *testing.T) {
	r := style.NewResolver()

	parentBG := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	parentComputed := &style.Computed{
		Background: parentBG,
	}

	child := &fakeNode{dirtyStyle: true}
	got := r.Resolve(child, parentComputed)

	if got.Background == parentBG {
		t.Errorf("Background should not be inherited")
	}
}

// ---------------------------------------------------------------------------
// TSK-022 — Intrinsic Style Layer unit tests (ADR-010)
// ---------------------------------------------------------------------------

// TestIntrinsicStyle_WinsOverAuthor verifies that a property set via
// IntrinsicStyle() overrides the same property set by the author's RawStyle.
// (intrinsic wins — test 4.1 bullet 1)
func TestIntrinsicStyle_WinsOverAuthor(t *testing.T) {
	r := style.NewResolver()

	// Author wants Block; UA forces InlineBlock.
	node := &fakeNode{
		dirtyStyle:     true,
		rawStyle:       style.Style{Display: style.Some(style.DisplayBlock)},
		intrinsicStyle: style.Style{Display: style.Some(style.DisplayInlineBlock)},
	}

	got := r.Resolve(node, nil)
	if got.Display != style.DisplayInlineBlock {
		t.Errorf("Display = %v, want DisplayInlineBlock (intrinsic must win over author)", got.Display)
	}
}

// TestIntrinsicStyle_EmptyDoesNotNullifyAuthor verifies that an empty
// IntrinsicStyle() does not nullify a property set by the author.
// (test 4.1 bullet 2)
func TestIntrinsicStyle_EmptyDoesNotNullifyAuthor(t *testing.T) {
	r := style.NewResolver()

	red := style.TerminalDefault // any non-zero color marker
	node := &fakeNode{
		dirtyStyle:     true,
		rawStyle:       style.Style{Foreground: style.Some(red)},
		intrinsicStyle: style.Style{}, // empty — does not force Foreground
	}

	got := r.Resolve(node, nil)
	if got.Foreground != red {
		t.Errorf("Foreground = %v, want %v (empty intrinsic must not nullify author)", got.Foreground, red)
	}
}

// TestIntrinsicStyle_InheritablePropertyCascadesToChild verifies that an
// inheritable property set via IntrinsicStyle() on the parent cascades into a
// child that has no explicit value for it.
// (test 4.1 bullet 3)
func TestIntrinsicStyle_InheritablePropertyCascadesToChild(t *testing.T) {
	r := style.NewResolver()

	// Parent forces WhiteSpace:PreWrap via intrinsic style.
	parent := &fakeNode{
		dirtyStyle:     true,
		intrinsicStyle: style.Style{WhiteSpace: style.Some(style.WhiteSpacePreWrap)},
	}
	parentComputed := r.Resolve(parent, nil)

	// Child has no explicit WhiteSpace — should inherit from parent.
	child := &fakeNode{dirtyStyle: true}
	got := r.Resolve(child, parentComputed)

	if got.WhiteSpace != style.WhiteSpacePreWrap {
		t.Errorf("WhiteSpace = %v, want WhiteSpacePreWrap (inherited from parent's intrinsic)", got.WhiteSpace)
	}
}

// TestIntrinsicStyle_LayerOrder verifies the full cascade precedence:
// DefaultStyle < RawStyle < IntrinsicStyle, each winning over the layer below.
// (test 4.1 bullet 4)
func TestIntrinsicStyle_LayerOrder(t *testing.T) {
	r := style.NewResolver()

	// DefaultStyle provides FlexRow; RawStyle overrides to FlexColumn;
	// IntrinsicStyle overrides to FlexRow again (UA forces it).
	node := &fakeNode{
		dirtyStyle:          true,
		elementDefaultStyle: style.Style{FlexDirection: style.Some(style.FlexRow)},
		rawStyle:            style.Style{FlexDirection: style.Some(style.FlexColumn)},
		intrinsicStyle:      style.Style{FlexDirection: style.Some(style.FlexRow)},
	}

	got := r.Resolve(node, nil)
	if got.FlexDirection != style.FlexRow {
		t.Errorf("FlexDirection = %v, want FlexRow (intrinsic must override author)", got.FlexDirection)
	}

	// Also verify that RawStyle wins over DefaultStyle when intrinsic is empty.
	node2 := &fakeNode{
		dirtyStyle:          true,
		elementDefaultStyle: style.Style{FlexDirection: style.Some(style.FlexRow)},
		rawStyle:            style.Style{FlexDirection: style.Some(style.FlexColumn)},
		intrinsicStyle:      style.Style{}, // empty
	}
	got2 := r.Resolve(node2, nil)
	if got2.FlexDirection != style.FlexColumn {
		t.Errorf("FlexDirection = %v, want FlexColumn (rawStyle must override defaultStyle)", got2.FlexDirection)
	}
}

// TestIntrinsicStyle_NoIntrinsicRegressionGuard verifies that elements that do
// not implement IntrinsicStyle (return empty Style{}) see no behavioral change.
// (test 4.1 bullet 5)
func TestIntrinsicStyle_NoIntrinsicRegressionGuard(t *testing.T) {
	r := style.NewResolver()

	// Plain node with no intrinsic style — behaviour must be identical to
	// before the intrinsic layer was introduced.
	node := &fakeNode{
		dirtyStyle: true,
		rawStyle: style.Style{
			Display: style.Some(style.DisplayFlex),
			Width:   style.Some(style.Cells(20)),
		},
	}

	got := r.Resolve(node, nil)
	if got.Display != style.DisplayFlex {
		t.Errorf("Display = %v, want DisplayFlex", got.Display)
	}
	if got.Width != style.Cells(20) {
		t.Errorf("Width = %v, want Cells(20)", got.Width)
	}
}
