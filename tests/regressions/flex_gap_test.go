package regressions

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

type mockNode struct {
	layout.Node
	style    *style.Computed
	children []layout.Node
}

func (m *mockNode) Style() *style.Computed {
	return m.style
}

func (m *mockNode) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		for _, child := range m.children {
			if !yield(child) {
				return
			}
		}
	}
}

func (m *mockNode) LogicalNode() any                                                    { return nil }
func (m *mockNode) IsDirtyLayout() bool                                                 { return false }
func (m *mockNode) ClearDirtyLayout()                                                   {}
func (m *mockNode) Fragment() *layout.Fragment                                          { return nil }
func (m *mockNode) CachedLayout(space layout.ConstraintSpace) *layout.Fragment          { return nil }
func (m *mockNode) SetCachedLayout(space layout.ConstraintSpace, frag *layout.Fragment) {}
func (m *mockNode) CachedMinMaxSizes() (layout.MinMaxSizes, bool)                       { return layout.MinMaxSizes{}, false }
func (m *mockNode) SetCachedMinMaxSizes(sizes layout.MinMaxSizes)                       {}
func (m *mockNode) SetOffset(p layout.Point)                                            {}

func TestFlexGapInColumn(t *testing.T) {
	// Create a column flex container with 2 children and a gap.
	// Container height should be sum of children heights + gap.

	child1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 1, Basis: style.Auto},
		},
	}
	child2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 1, Basis: style.Auto},
		},
	}

	container := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayFlex,
			FlexDirection: style.FlexColumn,
			Gap:           style.Gap(1, 0), // Row gap 1, Column gap 0
			Height:        style.Auto,
			Width:         style.Cells(20),
		},
		children: []layout.Node{child1, child2},
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 20, Height: 100}).ToConstraintSpace()
	algo := layout.NewAlgorithm(container, space)
	frag := algo.Layout()

	// Expected height: 1 (child1) + 1 (gap) + 1 (child2) = 3
	if frag.Size.Height != 3 {
		t.Errorf("Expected height 3, got %d", frag.Size.Height)
	}

	// Check child offsets
	if len(frag.Children) != 2 {
		t.Fatalf("Expected 2 children fragments, got %d", len(frag.Children))
	}

	if frag.Children[0].Offset.Y != 0 {
		t.Errorf("Child 1 offset.Y should be 0, got %d", frag.Children[0].Offset.Y)
	}
	if frag.Children[1].Offset.Y != 2 {
		t.Errorf("Child 2 offset.Y should be 2 (1 child + 1 gap), got %d", frag.Children[1].Offset.Y)
	}
}

func TestFlexJustifyBetweenNegativeSpace(t *testing.T) {
	// Create a row flex container with 2 children and NO space left.
	// JustifyBetween should fallback to JustifyStart and use minimum gaps.

	child1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}
	child2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}

	container := &mockNode{
		style: &style.Computed{
			Display:        style.DisplayFlex,
			FlexDirection:  style.FlexRow,
			Gap:            style.Gap(0, 2), // Row gap 0, Column gap 2
			Height:         style.Cells(1),
			Width:          style.Cells(15), // Total needed: 10 + 2 + 10 = 22. Overflow!
			JustifyContent: style.JustifyBetween,
		},
		children: []layout.Node{child1, child2},
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 15, Height: 1}).ToConstraintSpace()
	algo := layout.NewAlgorithm(container, space)
	frag := algo.Layout()

	// Check child offsets
	if len(frag.Children) != 2 {
		t.Fatalf("Expected 2 children fragments, got %d", len(frag.Children))
	}

	if frag.Children[0].Offset.X != 0 {
		t.Errorf("Child 1 offset.X should be 0, got %d", frag.Children[0].Offset.X)
	}
	// Expected offset for Child 2: 10 (Child 1) + 2 (Gap) = 12.
	if frag.Children[1].Offset.X != 12 {
		t.Errorf("Child 2 offset.X should be 12 (10 child + 2 gap), got %d", frag.Children[1].Offset.X)
	}
}

func TestFlexSqueezedHeight(t *testing.T) {
	// Create a row flex container with 1 item of height 1.
	// Container is squeezed to height 0.
	// It should NOT fragment and return empty, but instead return the item (overflowing).

	child := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 1, Basis: style.Auto},
		},
	}

	container := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayFlex,
			FlexDirection: style.FlexRow,
			Height:        style.Auto,
			Width:         style.Cells(10),
		},
		children: []layout.Node{child},
	}

	// Squeezed height = 0
	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 10, Height: 0}).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	algo := layout.NewAlgorithm(container, space)
	frag := algo.Layout()

	// Check if child is present
	if len(frag.Children) == 0 {
		t.Fatalf("Expected at least one child fragment even when squeezed, got 0")
	}
}

func TestFlexColumnGap(t *testing.T) {
	// Create a column flex container with 3 items and a gap of 1.
	// Each item is 2 high.
	// Total height should be 2*3 + 1*2 = 8.

	child1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(2),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}
	child2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(2),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}
	child3 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(2),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}

	container := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayFlex,
			FlexDirection: style.FlexColumn,
			Gap:           style.Gap(1, 0), // Row gap 1
			Height:        style.Auto,
			Width:         style.Cells(10),
		},
		children: []layout.Node{child1, child2, child3},
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 10, Height: 100}).ToConstraintSpace()
	algo := layout.NewAlgorithm(container, space)
	frag := algo.Layout()

	if frag.Size.Height != 8 {
		t.Errorf("Expected height 8, got %d", frag.Size.Height)
	}

	if len(frag.Children) != 3 {
		t.Fatalf("Expected 3 children, got %d", len(frag.Children))
	}

	// Item 1: Y=0
	// Item 2: Y=3 (2+1)
	// Item 3: Y=6 (3+2+1)
	if frag.Children[1].Offset.Y != 3 {
		t.Errorf("Child 2 offset.Y should be 3, got %d", frag.Children[1].Offset.Y)
	}
	if frag.Children[2].Offset.Y != 6 {
		t.Errorf("Child 3 offset.Y should be 6, got %d", frag.Children[2].Offset.Y)
	}
}

func TestFlexNestedHeight(t *testing.T) {
	// Create a column flex container with an inner column flex container.
	// Inner container has 2 items of height 1 and gap 1. Height should be 3.
	// Outer container should see the inner one as 3 high.

	child1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}
	child2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Height:  style.Cells(1),
			Width:   style.Cells(10),
			Flex:    style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
	}

	inner := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayFlex,
			FlexDirection: style.FlexColumn,
			Gap:           style.Gap(1, 0), // Row gap 1
			Height:        style.Auto,
			Width:         style.Cells(10),
			Flex:          style.FlexItemValue{Grow: 0, Shrink: 0, Basis: style.Auto},
		},
		children: []layout.Node{child1, child2},
	}

	outer := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayFlex,
			FlexDirection: style.FlexColumn,
			Height:        style.Auto,
			Width:         style.Cells(20),
		},
		children: []layout.Node{inner},
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 20, Height: 100}).ToConstraintSpace()
	algo := layout.NewAlgorithm(outer, space)
	frag := algo.Layout()

	// Inner should be 3 high. Outer should be 3 high.
	if frag.Size.Height != 3 {
		t.Errorf("Expected outer height 3, got %d", frag.Size.Height)
	}

	if len(frag.Children) != 1 {
		t.Fatalf("Expected 1 child in outer, got %d", len(frag.Children))
	}

	innerFrag := frag.Children[0].Fragment
	if innerFrag.Size.Height != 3 {
		t.Errorf("Expected inner height 3, got %d", innerFrag.Size.Height)
	}
}
