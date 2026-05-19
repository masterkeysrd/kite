package render

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

func TestRegression_InheritancePropagation(t *testing.T) {
	view := NewRenderView()
	view.SetViewportSize(layout.Size{Width: 80, Height: 24})

	parent := NewBlock(nil, nil)
	parent.SetRawStyle(style.Style{
		Foreground: style.Some[color.Color](color.White),
	})
	view.InsertChild(parent, nil)

	child := NewBlock(nil, nil)
	// child does not set foreground, should inherit
	parent.InsertChild(child, nil)

	resolver := style.NewResolver()
	style.ResolveTree(resolver, view)

	if child.ComputedStyle().Foreground != color.White {
		t.Fatalf("Child should inherit white, got %v", child.ComputedStyle().Foreground)
	}

	// Change parent foreground
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	parent.SetRawStyle(style.Style{
		Foreground: style.Some[color.Color](red),
	})

	// Resolve again
	style.ResolveTree(resolver, view)

	if child.ComputedStyle().Foreground != red {
		t.Fatalf("Child should inherit red, got %v", child.ComputedStyle().Foreground)
	}
}

func TestRegression_FlexLayoutAfterDRY(t *testing.T) {
	view := NewRenderView()
	view.SetViewportSize(layout.Size{Width: 80, Height: 24})

	flex := NewBox(nil, nil)
	flex.SetRawStyle(style.Style{
		Display: style.Some(style.DisplayFlex),
		Width:   style.Some(style.Percent(100)),
		Height:  style.Some(style.Percent(100)),
	})
	view.InsertChild(flex, nil)

	child1 := NewBlock(nil, nil)
	child1.SetRawStyle(style.Style{
		Flex: style.Some(style.Flex(1, 1, style.Auto)),
	})
	flex.InsertChild(child1, nil)

	child2 := NewBlock(nil, nil)
	child2.SetRawStyle(style.Style{
		Flex: style.Some(style.Flex(1, 1, style.Auto)),
	})
	flex.InsertChild(child2, nil)

	resolver := style.NewResolver()
	style.ResolveTree(resolver, view)

	viewport := view.ViewportSize()
	LayoutPhase(view, viewport)

	if flex.Fragment() == nil {
		t.Fatal("Flex fragment is nil")
	}
	if len(flex.Fragment().Children) != 2 {
		t.Fatalf("Expected 2 children in flex fragment, got %d", len(flex.Fragment().Children))
	}

	// Verify they are positioned side-by-side (default Row)
	c1 := flex.Fragment().Children[0]
	c2 := flex.Fragment().Children[1]

	if c1.Offset.X == c2.Offset.X && c1.Fragment.Size.Width > 0 {
		t.Errorf("Flex children should not overlap horizontally in Row. c1.X=%d, c2.X=%d", c1.Offset.X, c2.Offset.X)
	}

	// Also check they are added in order
	if c1.Fragment.Node != (layout.Node)(child1) {
		t.Errorf("First child in fragment should be child1")
	}
	if c2.Fragment.Node != (layout.Node)(child2) {
		t.Errorf("Second child in fragment should be child2")
	}
}

func TestRegression_MultipleChildrenBlock(t *testing.T) {
	view := NewRenderView()
	view.SetViewportSize(layout.Size{Width: 80, Height: 24})

	block := NewBlock(nil, nil)
	view.InsertChild(block, nil)

	child1 := NewBlock(nil, nil)
	child1.SetRawStyle(style.Style{Height: style.Some(style.Cells(1))})
	block.InsertChild(child1, nil)

	child2 := NewBlock(nil, nil)
	child2.SetRawStyle(style.Style{Height: style.Some(style.Cells(1))})
	block.InsertChild(child2, nil)

	style.ResolveTree(style.NewResolver(), view)
	LayoutPhase(view, view.ViewportSize())

	if len(block.Fragment().Children) != 2 {
		t.Fatalf("Expected 2 children in block, got %d", len(block.Fragment().Children))
	}

	c1 := block.Fragment().Children[0]
	c2 := block.Fragment().Children[1]

	if c1.Offset.Y == c2.Offset.Y {
		t.Errorf("Block children should not overlap vertically. c1.Y=%d, c2.Y=%d", c1.Offset.Y, c2.Offset.Y)
	}
}
