package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

type mockTextSource struct {
	data string
}

func (m *mockTextSource) Data() string { return m.data }

func TestFlexColumnGrowingHeightWithWrappingChildren(t *testing.T) {
	// Container: Width 34, Padding 2. Available content width = 30.
	// 3 items, each with Padding 2.
	// Text "Column Item 1 (Stays Right) ..." (long).
	// Container has 30 width available. Items have 26 content width.
	// Items will wrap to multiple lines.

	container := render.NewBlock(nil, nil)
	s := style.DefaultStyle()
	s.Display = style.DisplayFlex
	s.FlexDirection = style.FlexColumn
	s.Width = style.Cells(34)
	s.Height = style.Auto
	s.Padding = style.EdgeValues[int]{Top: 1, Bottom: 1, Left: 2, Right: 2}
	s.Gap = style.Gap(1, 0) // Row gap 1
	s.AlignItems = style.AlignEnd
	container.SetComputedStyle(&s)

	for i := 1; i <= 3; i++ {
		textData := "Column Item 1 (Stays Right) and some more text to force wrapping to multiple lines"
		textRender := render.NewText(&mockTextSource{data: textData}, nil)
		st := style.DefaultStyle()
		st.Display = style.DisplayInline
		st.WhiteSpace = style.WhiteSpaceNormal
		textRender.SetComputedStyle(&st)

		childBox := render.NewBlock(nil, nil)
		st = style.DefaultStyle()
		st.Display = style.DisplayBlock
		st.Padding = style.EdgeValues[int]{Left: 2, Right: 2}
		st.Width = style.Auto
		st.Height = style.Auto
		childBox.SetComputedStyle(&st)
		childBox.InsertChild(textRender, nil)
		container.InsertChild(childBox, nil)
	}

	space := layout.NewConstraintSpaceBuilder(geom.Size{Width: 80, Height: 24}).
		SetContainerSpace(geom.Size{Width: 80, Height: 24}).
		SetContainingSpace(geom.Size{Width: 80, Height: 24}).
		ToConstraintSpace()

	algo := layout.GetAlgorithm(container)
	frag := algo.Layout(nil, container, space)

	// Each item should have height > 1.
	for i, childLink := range frag.Children {
		if childLink.Fragment.Size.Height <= 1 {
			t.Errorf("Item %d: expected height > 1, got %d", i+1, childLink.Fragment.Size.Height)
		}
	}

	// Container height should be padding + sum(items) + gaps.
	// Minimum expected height if items wrap to 2 lines each: 1 + 2 + 1 + 2 + 1 + 2 + 1 = 10.
	expectedMinHeight := 10
	if frag.Size.Height < expectedMinHeight {
		t.Errorf("Expected container height at least %d, got %d", expectedMinHeight, frag.Size.Height)
	}
}
