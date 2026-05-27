package layout

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

func TestLayout_ScrollbarSpaceReservation(t *testing.T) {
	// Root node with ScrollbarY enabled and OverflowScroll (forced reservation)
	rootStyle := style.Style{
		OverflowX: style.Some(style.OverflowAuto),
		OverflowY: style.Some(style.OverflowScroll),
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(10)),
	}.ScrollbarY(true)

	rootComp := rootStyle.Apply(style.DefaultStyle())
	root := &mockLayoutNode{
		style: &rootComp,
	}

	// Child node that wants 100% width
	childStyle := style.Style{
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Cells(1)),
	}
	childComp := childStyle.Apply(style.DefaultStyle())
	child := &mockLayoutNode{
		style: &childComp,
	}
	root.children = []Node{child}

	space := NewConstraintSpaceBuilder(geometry.Size{Width: 10, Height: 10}).
		SetIsFixedInlineSize(true).
		SetContainerSpace(geometry.Size{Width: 10, Height: 10}).
		ToConstraintSpace()

	algo := GetAlgorithm(root)
	frag := algo.Layout(nil, root, space)

	if !frag.HasScrollbarY {
		t.Errorf("expected HasScrollbarY to be true")
	}

	// The child should have width 9 (10 - 1 for scrollbar)
	if len(frag.Children) == 0 {
		t.Fatalf("expected child fragment")
	}
	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 9 {
		t.Errorf("expected child width 9, got %d", childFrag.Size.Width)
	}
}

func TestLayout_ScrollbarAutoHidden(t *testing.T) {
	// Root node with ScrollbarY enabled and OverflowAuto, but content fits.
	rootStyle := style.Style{
		OverflowX: style.Some(style.OverflowAuto),
		OverflowY: style.Some(style.OverflowAuto),
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(10)),
	}.ScrollbarY(true)

	rootComp := rootStyle.Apply(style.DefaultStyle())
	root := &mockLayoutNode{
		style: &rootComp,
	}

	// Child node that fits exactly
	childStyle := style.Style{
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Cells(5)),
	}
	childComp := childStyle.Apply(style.DefaultStyle())
	child := &mockLayoutNode{
		style: &childComp,
	}
	root.children = []Node{child}

	space := NewConstraintSpaceBuilder(geometry.Size{Width: 10, Height: 10}).
		SetIsFixedInlineSize(true).
		SetContainerSpace(geometry.Size{Width: 10, Height: 10}).
		ToConstraintSpace()

	algo := GetAlgorithm(root)
	frag := algo.Layout(nil, root, space)

	if frag.HasScrollbarY {
		t.Errorf("expected HasScrollbarY to be false for non-overflowing content with OverflowAuto")
	}

	// The child should have full width 10
	if len(frag.Children) == 0 {
		t.Fatalf("expected child fragment")
	}
	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 10 {
		t.Errorf("expected child width 10, got %d", childFrag.Size.Width)
	}
}

type mockLayoutNode struct {
	style    *style.Computed
	children []Node
	fragment *Fragment
}

func (m *mockLayoutNode) Style() *style.Computed { return m.style }
func (m *mockLayoutNode) FirstLayoutChild() Node {
	if len(m.children) == 0 {
		return nil
	}
	return m.children[0]
}
func (m *mockLayoutNode) NextLayoutSibling(child Node) Node {
	for i, c := range m.children {
		if c == child {
			if i+1 < len(m.children) {
				return m.children[i+1]
			}
			break
		}
	}
	return nil
}
func (m *mockLayoutNode) LogicalNode() any         { return nil }
func (m *mockLayoutNode) IsDirtyLayout() bool      { return true }
func (m *mockLayoutNode) IsDirtyPaint() bool       { return true }
func (m *mockLayoutNode) HasChildNeedsPaint() bool { return true }
func (m *mockLayoutNode) ClearDirtyLayout()        {}
func (m *mockLayoutNode) Fragment() *Fragment      { return m.fragment }
func (m *mockLayoutNode) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}
func (m *mockLayoutNode) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	m.fragment = frag
}
func (m *mockLayoutNode) CachedMinMaxSizes() (MinMaxSizes, bool) { return MinMaxSizes{}, false }
func (m *mockLayoutNode) SetCachedMinMaxSizes(sizes MinMaxSizes) {}
func (m *mockLayoutNode) SetOffset(p geometry.Point)             {}
func (m *mockLayoutNode) IsAnonymous() bool                      { return false }
