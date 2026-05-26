package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestGridAlgorithm_Layout(t *testing.T) {
	newNode := func(width, height int) *mockNode {
		return &mockNode{
			style: &style.Computed{
				Width:  style.Cells(width),
				Height: style.Cells(height),
			},
		}
	}

	t.Run("2x2 1fr tracks", func(t *testing.T) {
		c4 := newNode(5, 1)
		c3 := newNode(5, 1)
		c2 := newNode(5, 1)
		c1 := newNode(5, 1)

		c1.nextSibling = c2
		c2.nextSibling = c3
		c3.nextSibling = c4

		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateColumns: style.Repeat(2, style.Fr(1)),
				GridTemplateRows:    style.Repeat(2, style.Fr(1)),
				Width:               style.Cells(20),
				Height:              style.Cells(10),
			},
			firstChild: c1,
		}

		space := NewConstraintSpaceBuilder(Size{20, 10}).
			SetContainingSpace(Size{20, 10}).
			SetContainerSpace(Size{20, 10}).
			SetIsFixedInlineSize(true).
			SetIsFixedBlockSize(true).
			ToConstraintSpace()

		algo := GetAlgorithm(parent)
		ctx := &Context{}
		frag := algo.Layout(ctx, parent, space)

		if frag.Size.Width != 20 || frag.Size.Height != 10 {
			t.Errorf("expected parent size 20x10, got %dx%d", frag.Size.Width, frag.Size.Height)
		}

		if len(frag.Children) != 4 {
			t.Fatalf("expected 4 children, got %d", len(frag.Children))
		}

		// Each child should be 10x5 (20/2 x 10/2)
		for i, child := range frag.Children {
			if child.Fragment.Size.Width != 10 || child.Fragment.Size.Height != 5 {
				t.Errorf("child %d: expected size 10x5, got %dx%d", i, child.Fragment.Size.Width, child.Fragment.Size.Height)
			}
		}

		// Check offsets
		expectedOffsets := []Point{
			{0, 0}, {10, 0},
			{0, 5}, {10, 5},
		}
		for i, child := range frag.Children {
			if child.Offset != expectedOffsets[i] {
				t.Errorf("child %d: expected offset %v, got %v", i, expectedOffsets[i], child.Offset)
			}
		}
	})

	t.Run("auto track expansion", func(t *testing.T) {
		c1 := newNode(15, 2)
		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateColumns: []style.GridTrackSize{style.Auto},
				Width:               style.Content,
				Height:              style.Auto,
			},
			firstChild: c1,
		}

		space := NewConstraintSpaceBuilder(Size{100, 100}).
			SetContainerSpace(Size{100, 100}).
			ToConstraintSpace()

		algo := GetAlgorithm(parent)
		ctx := &Context{}
		frag := algo.Layout(ctx, parent, space)

		// Parent should expand to fit c1 (15x2)
		if frag.Size.Width != 15 || frag.Size.Height != 2 {
			t.Errorf("expected parent size 15x2, got %dx%d", frag.Size.Width, frag.Size.Height)
		}
	})

	t.Run("gaps", func(t *testing.T) {
		c1 := newNode(5, 5)
		c2 := newNode(5, 5)
		c1.nextSibling = c2

		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateColumns: style.Repeat(2, style.Cells(5)),
				GridColumnGap:       2,
				Width:               style.Content, // Use Content to test shrink-wrap
				Height:              style.Auto,
			},
			firstChild: c1,
		}

		space := NewConstraintSpaceBuilder(Size{100, 100}).
			SetContainerSpace(Size{100, 100}).
			ToConstraintSpace()

		algo := GetAlgorithm(parent)
		ctx := &Context{}
		frag := algo.Layout(ctx, parent, space)

		// Width should be 5 + 2 + 5 = 12
		if frag.Size.Width != 12 {
			t.Errorf("expected parent width 12, got %d", frag.Size.Width)
		}

		if frag.Children[1].Offset.X != 7 {
			t.Errorf("expected second child offset X=7, got %d", frag.Children[1].Offset.X)
		}
	})

	t.Run("Width:Auto fills available width", func(t *testing.T) {
		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateColumns: []style.GridTrackSize{style.Fr(1)},
				Width:               style.Auto,
			},
		}
		space := NewConstraintSpaceBuilder(Size{85, 54}).
			SetContainerSpace(Size{85, 54}).
			ToConstraintSpace()
		algo := GetAlgorithm(parent)
		frag := algo.Layout(&Context{}, parent, space)
		if frag.Size.Width != 85 {
			t.Errorf("expected width 85 (fill available), got %d", frag.Size.Width)
		}
	})

	t.Run("Indefinite height with 100% height child", func(t *testing.T) {
		// Child with 100% height
		c1 := &mockNode{
			style: &style.Computed{
				Display: style.DisplayBlock,
				Height:  style.Percent(100),
				Width:   style.Percent(100),
			},
		}
		// Content of child (in reality another node, but we use IntrinsicBlockSize)
		// We'll give it a mockable intrinsic size if we can,
		// but mockNode doesn't support setting IntrinsicBlockSize easily without code changes.
		// Wait, IntrinsicBlockSize calls Layout on the node.
		// Our mockNode returns its cached fragment.

		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateRows:    []style.GridTrackSize{style.Auto},
				GridTemplateColumns: []style.GridTrackSize{style.Fr(1)},
				Height:              style.Auto,
			},
			firstChild: c1,
		}

		space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
		algo := GetAlgorithm(parent)

		// This should NOT panic or hang even if the child wants 100% of an indefinite height.
		frag := algo.Layout(&Context{}, parent, space)

		// Size should be at least 0.
		if frag.Size.Height < 0 {
			t.Errorf("expected non-negative height, got %d", frag.Size.Height)
		}
	})

	t.Run("Mixed tracks: Fixed, Auto, Fr", func(t *testing.T) {
		// c1 in Fixed(10)
		c1 := newNode(5, 1)
		// c2 in Auto (should expand to 15)
		c2 := newNode(15, 1)
		// c3 in Fr(1) (should take remaining)
		c3 := newNode(5, 1)

		c1.nextSibling = c2
		c2.nextSibling = c3

		parent := &mockNode{
			style: &style.Computed{
				Display:             style.DisplayGrid,
				GridTemplateColumns: []style.GridTrackSize{style.Cells(10), style.Auto, style.Fr(1)},
				Width:               style.Cells(40),
			},
			firstChild: c1,
		}

		space := NewConstraintSpaceBuilder(Size{40, 10}).ToConstraintSpace()
		algo := GetAlgorithm(parent)
		frag := algo.Layout(&Context{}, parent, space)

		// colWidths:
		// 1: Fixed 10
		// 2: Auto (min-content 15) -> 15
		// 3: Fr(1) -> remaining (40 - 10 - 15 = 15)

		if len(frag.Children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(frag.Children))
		}

		widths := []int{10, 15, 15}
		offsets := []int{0, 10, 25}

		for i, child := range frag.Children {
			if child.Fragment.Size.Width != widths[i] {
				t.Errorf("child %d: expected width %d, got %d", i, widths[i], child.Fragment.Size.Width)
			}
			if child.Offset.X != offsets[i] {
				t.Errorf("child %d: expected offset X %d, got %d", i, offsets[i], child.Offset.X)
			}
		}
	})
}

func TestGridAlgorithm_ComputeMinMaxSizes(t *testing.T) {
	newNode := func(width, height int) *mockNode {
		return &mockNode{
			style: &style.Computed{
				Width:  style.Cells(width),
				Height: style.Cells(height),
			},
		}
	}

	c1 := newNode(10, 1)
	c2 := newNode(20, 1)
	c1.nextSibling = c2

	parent := &mockNode{
		style: &style.Computed{
			Display:             style.DisplayGrid,
			GridTemplateColumns: []style.GridTrackSize{style.Auto, style.Auto},
			GridColumnGap:       5,
		},
		firstChild: c1,
	}

	algo := &GridAlgorithm{}
	ctx := &Context{}
	sizes := algo.ComputeMinMaxSizes(ctx, parent)

	// Max should be 10 + 5 + 20 = 35
	if sizes.Max != 35 {
		t.Errorf("expected max-content 35, got %d", sizes.Max)
	}
}

func TestGridAlgorithm_Rendering(t *testing.T) {
	// Simulate a box with text "A" inside a grid
	c1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Percent(100),
			Height:  style.Percent(100),
		},
	}

	parent := &mockNode{
		style: &style.Computed{
			Display:             style.DisplayGrid,
			GridTemplateColumns: []style.GridTrackSize{style.Cells(10)},
			GridTemplateRows:    []style.GridTrackSize{style.Cells(5)},
			Padding:             style.Edges(1),
		},
		firstChild: c1,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	ctx := &Context{}
	frag := algo.Layout(ctx, parent, space)

	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	childFrag := frag.Children[0]
	// Insets = 1 (padding) + 0 (no border) = 1
	if childFrag.Offset.X != 1 || childFrag.Offset.Y != 1 {
		t.Errorf("expected child offset (1, 1), got %v", childFrag.Offset)
	}

	// Track is 10. Item is 100%. So child size should be 10.
	if childFrag.Fragment.Size.Width != 10 || childFrag.Fragment.Size.Height != 5 {
		t.Errorf("expected child size 10x5, got %dx%d", childFrag.Fragment.Size.Width, childFrag.Fragment.Size.Height)
	}
}

func TestGridAlgorithm_ExplicitPlacement(t *testing.T) {
	newNode := func(name string) *mockNode {
		return &mockNode{
			style: &style.Computed{
				Display: style.DisplayBlock,
			},
		}
	}

	// c1 at (2, 2) - 1-based coordinates in CSS
	c1 := newNode("c1")
	c1.style.GridColumn = style.GridPlacement{Start: 2}
	c1.style.GridRow = style.GridPlacement{Start: 2}

	parent := &mockNode{
		style: &style.Computed{
			Display:             style.DisplayGrid,
			GridTemplateColumns: []style.GridTrackSize{style.Cells(10), style.Cells(10)},
			GridTemplateRows:    []style.GridTrackSize{style.Cells(5), style.Cells(5)},
			GridColumnGap:       2,
			GridRowGap:          1,
		},
		firstChild: c1,
	}

	space := NewConstraintSpaceBuilder(Size{30, 20}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	ctx := &Context{}
	frag := algo.Layout(ctx, parent, space)

	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	childFrag := frag.Children[0]
	// Offset should be:
	// X = colWidths[0] + colGap = 10 + 2 = 12
	// Y = rowHeights[0] + rowGap = 5 + 1 = 6
	if childFrag.Offset.X != 12 || childFrag.Offset.Y != 6 {
		t.Errorf("expected child offset (12, 6), got %v", childFrag.Offset)
	}
}
