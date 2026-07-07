package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestRegression_SidebarContentDynamicLayout(t *testing.T) {
	// Total width 40, height 10.
	env := testenv.Default(40, 10)
	defer env.Close()

	colorSidebar := color.RGBA{100, 100, 100, 255}
	colorContent := color.RGBA{50, 50, 50, 255}
	colorInner := color.RGBA{0, 0, 200, 255}

	// Inner box inside content: has single border and solid background.
	innerBox := element.Box(
		element.Box("Some Text").Style(style.S().Foreground(color.RGBA{255, 255, 255, 255})),
	).Style(
		style.S().
			Width(style.Percent(100)).
			Height(style.Cells(4)).
			Border(style.SingleBorder()).
			Background(colorInner),
	).WithID("inner-box")

	// Content area on the right.
	contentArea := element.Box(innerBox).Style(
		style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Flex(1, 1, style.Auto).
			Background(colorContent),
	).WithID("content-area")

	// Sidebar on the left.
	sidebar := element.Box().Style(
		style.S().
			Width(style.Cells(10)).
			Height(style.Percent(100)).
			Background(colorSidebar),
	).WithID("sidebar")

	// Root container: flex row.
	root := element.Box(sidebar, contentArea).Style(
		style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Width(style.Percent(100)).
			Height(style.Percent(100)),
	)

	env.Mount(root)
	env.RenderFrame()

	// 1. Verify initial layout state (sidebar visible).
	// Sidebar is at columns 0..9.
	testenv.ExpectScreen(t, env).Region(0, 0, 10, 10).ToHaveBackground(colorSidebar)
	// Content area is at columns 10..39.
	// Inner box is inside content area. With width:100% and contentArea having flex:1 (consuming remaining 30 cells),
	// innerBox should be 30 cells wide, starting at col 10.
	roInner := env.RenderObject(innerBox)
	if roInner.Fragment().Size.Width != 30 {
		t.Fatalf("Expected inner box width 30, got %d", roInner.Fragment().Size.Width)
	}

	// 2. Collapse Sidebar: Set sidebar width to 0.
	sidebar.Style(
		style.S().
			Width(style.Cells(0)).
			Height(style.Percent(100)).
			Background(colorSidebar),
	)

	env.RenderFrame()

	// After collapse, contentArea consumes all 40 cells.
	// Inner box should expand to width 40, starting at col 0.
	roInnerAfter := env.RenderObject(innerBox)
	if roInnerAfter.Fragment().Size.Width != 40 {
		t.Fatalf("After collapse: Expected inner box width 40, got %d", roInnerAfter.Fragment().Size.Width)
	}

	// Verify that the inner box's borders and background are intact at col 0 and col 39.
	// Every row of the inner box (height 4, y=0..3) must have correct background color inside the borders.
	// The borders are at:
	// Top: y=0 (all x)
	// Bottom: y=3 (all x)
	// Left: x=0 (all y)
	// Right: x=39 (all y)
	// The background region (x=1..38, y=1..2) must have colorInner.
	testenv.ExpectScreen(t, env).Region(1, 1, 38, 2).ToHaveBackground(colorInner)

	// The borders should contain the correct border characters.
	// Let's assert single border horizontal and vertical corners/glyphs.
	topLineRunes := []rune("┌" + repeatString("─", 38) + "┐")
	bottomLineRunes := []rune("└" + repeatString("─", 38) + "┘")

	for x := 0; x < 40; x++ {
		expectedTopChar := string(topLineRunes[x])
		expectedBottomChar := string(bottomLineRunes[x])

		testenv.ExpectScreen(t, env).CellAt(x, 0).ToHaveContent(expectedTopChar)
		testenv.ExpectScreen(t, env).CellAt(x, 3).ToHaveContent(expectedBottomChar)
	}

	// Check vertical borders at cols 0 and 39 (rows 1 and 2).
	for y := 1; y <= 2; y++ {
		testenv.ExpectScreen(t, env).CellAt(0, y).ToHaveContent("│")
		testenv.ExpectScreen(t, env).CellAt(39, y).ToHaveContent("│")
	}
}

func repeatString(s string, count int) string {
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func TestRegression_OverlayJunctionBug(t *testing.T) {
	env := testenv.Default(10, 5)
	defer env.Close()

	// Outer box with border.
	outer := element.Box().Style(style.S().Width(style.Cells(10)).Height(style.Cells(5)).Border(style.SingleBorder()))

	env.Mount(outer)

	// Add an overlay.
	overlay := element.Box().Style(style.S().Width(style.Cells(2)).Height(style.Cells(1)).Background(color.RGBA{255, 0, 0, 255}).Margin(2, 2, 0, 0))
	env.ShowOverlay(overlay, 1)

	env.RenderFrame()

	// The top-left corner at (0,0) must be resolved to "┌".
	testenv.ExpectScreen(t, env).CellAt(0, 0).ToHaveContent("┌")
}
