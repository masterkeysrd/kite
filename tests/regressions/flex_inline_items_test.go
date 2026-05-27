// Regression tests for Layout/Flex — covers inline-block elements not being grouped into AnonymousBlock.

package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

func TestFlexRowInlineChildrenDoNotGroup(t *testing.T) {
	// Create a row flex container (width 100).
	// Child 1: span element (DisplayInline), width 10 cells.
	// Child 2: input element (DisplayInlineBlock), flex 1 (should grow to consume the remaining 90 cells).

	container := render.NewBlock(nil, nil)
	s := style.DefaultStyle()
	s.Display = style.DisplayFlex
	s.FlexDirection = style.FlexRow
	s.Width = style.Cells(100)
	s.Height = style.Auto
	container.SetComputedStyle(&s)

	// Child 1: Span-like element (inline)
	span := render.NewBlock(nil, nil)
	sSpan := style.DefaultStyle()
	sSpan.Display = style.DisplayInline
	sSpan.Width = style.Cells(10)
	sSpan.Height = style.Cells(1)
	span.SetComputedStyle(&sSpan)
	container.InsertChild(span, nil)

	// Child 2: Input-like element (inline-block)
	input := render.NewBlock(nil, nil)
	sInput := style.DefaultStyle()
	sInput.Display = style.DisplayInlineBlock
	sInput.Flex = style.FlexItemValue{Grow: 1, Shrink: 1, Basis: style.Auto}
	sInput.Height = style.Cells(1)
	input.SetComputedStyle(&sInput)
	container.InsertChild(input, nil)

	space := layout.NewConstraintSpaceBuilder(geom.Size{Width: 100, Height: 24}).
		SetContainerSpace(geom.Size{Width: 100, Height: 24}).
		SetContainingSpace(geom.Size{Width: 100, Height: 24}).
		ToConstraintSpace()

	algo := layout.GetAlgorithm(container)
	frag := algo.Layout(nil, container, space)

	// Check that the container has exactly 2 direct children, not 1 AnonymousBlock.
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children, got %d (inline children might have been wrapped in AnonymousBlock)", len(frag.Children))
	}

	// First child (span) width should be 10.
	spanWidth := frag.Children[0].Fragment.Size.Width
	if spanWidth != 10 {
		t.Errorf("expected span width 10, got %d", spanWidth)
	}

	// Second child (input) width should be 90 (flex-grow fills the remaining space).
	inputWidth := frag.Children[1].Fragment.Size.Width
	if inputWidth != 90 {
		t.Errorf("expected input width 90 (100 - 10), got %d (flex-grow was likely ignored)", inputWidth)
	}
}
