package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

// Regression test for FlexWrap — covers issue where wrapping flex containers
// don't shrink because their Min intrinsic size is calculated as sum of children.
func TestFlexWrapMinSize(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// Create a wrapping flex container with 5 items, each 10 wide.
	// Total width = 50 + 4 gaps (if gap is 1) = 54.
	// Terminal width is 40. It SHOULD wrap.

	tools := element.Box().WithID("tools").Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Gap(1),
	)

	for i := 0; i < 5; i++ {
		tools.AddChild(element.Box().Style(style.S().
			Width(style.Cells(10)).
			Height(style.Cells(1)),
		))
	}

	// Wrap it in another box to see if it's constrained.
	root := element.Box().Style(style.S().
		Width(style.Percent(100)).
		Height(style.Percent(100)),
	).AddChild(tools)

	env.Mount(root)
	env.Flush()

	// The tools box should have wrapped.
	node := env.GetNodeByID("tools")
	rect, ok := node.(dom.Element).GetBoundingClientRect()
	if !ok {
		t.Fatal("could not get bounding client rect for tools")
	}

	t.Logf("Tools Box Rect: %+v", rect)

	if rect.Size.Width > 40 {
		t.Errorf("Tools box overflowed: width %d > 40", rect.Size.Width)
	}

	if rect.Size.Height <= 1 {
		t.Errorf("Tools box did not wrap: height %d <= 1", rect.Size.Height)
	}
}

func TestFlexWrapMinSizeFail(t *testing.T) {
	env := testenv.Default(40, 20)
	defer env.Close()

	// Root is a FlexRow (reproducing L14 in the dump).
	root := element.Box().Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		Width(style.Cells(40)),
	)

	// Inner is a FlexColumn (reproducing L274).
	inner := element.Box().Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexColumn).
		Flex(0, 1),
	)

	// Tools box inside Inner.
	tools := element.Box().WithID("tools").Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Gap(1),
	)
	for i := 0; i < 5; i++ {
		tools.AddChild(element.Box().Style(style.S().
			Width(style.Cells(10)).
			Height(style.Cells(1)),
		))
	}
	inner.AddChild(tools)
	root.AddChild(inner)

	env.Mount(root)
	env.Flush()

	node := env.GetNodeByID("tools")
	rect, ok := node.(dom.Element).GetBoundingClientRect()
	if !ok {
		t.Fatal("could not get bounding client rect for tools")
	}

	t.Logf("Tools Box Rect: %+v", rect)

	if rect.Size.Width > 40 {
		t.Errorf("Tools box overflowed: width %d > 40", rect.Size.Width)
	}

	if rect.Size.Height <= 1 {
		t.Errorf("Tools box did not wrap: height %d <= 1", rect.Size.Height)
	}
}
