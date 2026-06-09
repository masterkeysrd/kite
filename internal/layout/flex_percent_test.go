package layout

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

func TestFlexLayout_PercentWidthItem(t *testing.T) {
	// Root container: Width 100
	// Child: Width 50%
	// Expected child width: 50

	childStyle := style.DefaultStyle()
	childStyle.Width = style.Percent(50)
	childStyle.Height = style.Cells(1)
	child := &mockNode{style: &childStyle}

	parentStyle := style.DefaultStyle()
	parentStyle.Display = style.DisplayFlex
	parentStyle.FlexDirection = style.FlexRow
	parentStyle.Width = style.Cells(100)
	parentStyle.Height = style.Auto

	parent := &mockNode{
		style:      &parentStyle,
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)

	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 50 {
		t.Errorf("expected child width 50, got %d", childFrag.Size.Width)
	}
}

func TestFlexLayout_ThreePerRowWithGap(t *testing.T) {
	// Root container: Width 90
	// 4 children: Width 30%
	// Gap: 2
	// 30% of 90 = 27
	// Row 1: 27 + 2 + 27 + 2 + 27 = 85. Fits!
	// Item 4: 85 + 2 + 27 = 114. Wraps.

	childStyle := style.DefaultStyle()
	childStyle.Width = style.Percent(30)
	childStyle.Height = style.Cells(1)

	c4 := &mockNode{style: &childStyle}
	c3 := &mockNode{style: &childStyle, nextSibling: c4}
	c2 := &mockNode{style: &childStyle, nextSibling: c3}
	c1 := &mockNode{style: &childStyle, nextSibling: c2}

	parentStyle := style.DefaultStyle()
	parentStyle.Display = style.DisplayFlex
	parentStyle.FlexDirection = style.FlexRow
	parentStyle.FlexWrap = style.FlexWrapOn
	parentStyle.Width = style.Cells(90)
	parentStyle.Gap = style.Gap(0, 2) // Column gap 2

	parent := &mockNode{
		style:      &parentStyle,
		firstChild: c1,
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)

	if len(frag.Children) != 4 {
		t.Fatalf("expected 4 children, got %d", len(frag.Children))
	}

	// X offsets: 0, 29 (27+2), 58 (29+29)
	expectedX := []int{0, 29, 58}
	for i := 0; i < 3; i++ {
		if frag.Children[i].Offset.X != expectedX[i] {
			t.Errorf("child %d: expected X offset %d, got %d", i, expectedX[i], frag.Children[i].Offset.X)
		}
		if frag.Children[i].Offset.Y != 0 {
			t.Errorf("child %d: expected Y offset 0, got %d", i, frag.Children[i].Offset.Y)
		}
	}

	if frag.Children[3].Offset.Y == 0 {
		t.Errorf("child 3: expected to wrap, but Y offset is 0")
	}
}
