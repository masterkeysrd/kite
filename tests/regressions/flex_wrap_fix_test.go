package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestFlexWrapItemsVisible(t *testing.T) {
	ev := testenv.Default(80, 24)
	defer ev.Close()

	// 80 wide. items are 17 wide.
	// 4 items = 17*4 + 3 gaps = 68+3 = 71.
	// 5 items = 17*5 + 4 gaps = 85+4 = 89 (too wide).
	flexWrapContainerStyle := style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Width(style.Percent(100)).
		Padding(style.Edges(1, 2)).
		Gap(style.Gap(1, 1))

	flexWrapItemStyle := style.S().
		Padding(style.Edges(0, 2)).
		Border(style.SingleBorder())

	root := element.Box(
		element.Box(
			element.Box("Long Item 1").Style(flexWrapItemStyle),
			element.Box("Long Item 2").Style(flexWrapItemStyle),
			element.Box("Long Item 3").Style(flexWrapItemStyle),
			element.Box("Long Item 4").Style(flexWrapItemStyle),
			element.Box("Long Item 5").Style(flexWrapItemStyle),
			element.Box("Long Item 6").Style(flexWrapItemStyle),
		).Style(flexWrapContainerStyle),
	).Style(style.S().Width(style.Percent(100)).Height(style.Percent(100)))

	ev.Mount(root)
	ev.Flush()

	// Items 1-4 on first line. Item 5 on second line.
	// Item 1 top is 1 (padding). Height is 3 (1 text + 2 border).
	// Item 5 top should be 1 (padding) + 3 (line 1) + 1 (gap) = 5.
	// Check for "L" from "Long Item 5" at (2, 6) relative to container?
	// Wait, Border is at (2, 5), text "Long Item 5" is at (3, 6).
	// Actually, x=2 (container padding left) + 1 (item border left) = 3.
	// y=5 (as calculated above) + 1 (item border top) = 6.
	testenv.ExpectScreen(t, ev).CellAt(5, 6).ToHaveContent("L")
}

func TestFlexWrapStretchToLineHeight(t *testing.T) {
	ev := testenv.Default(80, 24)
	defer ev.Close()

	containerStyle := style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Width(style.Cells(20))

	itemStyle := style.S().
		Background(color.RGBA{R: 255, G: 0, B: 0, A: 255})

	root := element.Box(
		element.Box(
			element.Box("Tall").Style(itemStyle.Height(style.Cells(3))),
			element.Box("Short").Style(itemStyle),
		).Style(containerStyle),
	).Style(style.S().Width(style.Percent(100)).Height(style.Percent(100)))

	ev.Mount(root)
	ev.Flush()

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	// "Tall" is 3 high.
	testenv.ExpectScreen(t, ev).Region(0, 0, 4, 3).ToHaveBackground(red)
	// "Short" should stretch to 3 high.
	testenv.ExpectScreen(t, ev).Region(4, 0, 5, 3).ToHaveBackground(red)
}
