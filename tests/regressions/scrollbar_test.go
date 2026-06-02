package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestPaint_ScrollbarRendering(t *testing.T) {
	env := testenv.Default(20, 10)
	defer env.Close()

	// Create a scrollable box with a scrollbar.
	// We use a large inner box to ensure overflow.
	box := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(20)),
		}),
	).Style(style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(5)),
		OverflowY: style.Some(style.OverflowScroll),
	}).ScrollbarY(true)

	env.Mount(box)
	env.Flush()

	// Initial state: thumb should be at the top.
	// The track is at X=9 (width 10, index 9).
	testenv.ExpectScreen(t, env).
		CellAt(9, 0).ToHaveContent("┃").
		CellAt(9, 1).ToHaveContent("│").
		CellAt(9, 2).ToHaveContent("│").
		CellAt(9, 3).ToHaveContent("│").
		CellAt(9, 4).ToHaveContent("│")

	// Scroll down.
	env.ScrollTo(box, 0, 15)
	env.Flush()

	// After scroll: thumb should be at the bottom.
	testenv.ExpectScreen(t, env).
		CellAt(9, 0).ToHaveContent("│").
		CellAt(9, 1).ToHaveContent("│").
		CellAt(9, 2).ToHaveContent("│").
		CellAt(9, 3).ToHaveContent("│").
		CellAt(9, 4).ToHaveContent("┃")
}

func TestPaint_HorizontalScrollbarRendering(t *testing.T) {
	env := testenv.Default(20, 10)
	defer env.Close()

	// Create a horizontally scrollable box.
	box := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(20)),
			Height: style.Some(style.Cells(5)),
		}),
	).Style(style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(5)),
		OverflowX: style.Some(style.OverflowScroll),
	}).ScrollbarX(true)

	env.Mount(box)
	env.Flush()

	// Initial state: thumb at the left.
	// The track is at Y=4 (height 5, index 4).
	// viewWidth = 10, extentWidth = 20 -> thumbWidth = 10*10/20 = 5.
	testenv.ExpectScreen(t, env).
		CellAt(0, 4).ToHaveContent("━").
		CellAt(1, 4).ToHaveContent("━").
		CellAt(2, 4).ToHaveContent("━").
		CellAt(3, 4).ToHaveContent("━").
		CellAt(4, 4).ToHaveContent("━").
		CellAt(5, 4).ToHaveContent("─")

	// Scroll right.
	env.ScrollTo(box, 10, 0)
	env.Flush()

	// After scroll: thumb at the right.
	testenv.ExpectScreen(t, env).
		CellAt(4, 4).ToHaveContent("─").
		CellAt(5, 4).ToHaveContent("━").
		CellAt(6, 4).ToHaveContent("━").
		CellAt(7, 4).ToHaveContent("━").
		CellAt(8, 4).ToHaveContent("━").
		CellAt(9, 4).ToHaveContent("━")
}

func TestPaint_ScrollbarAutoHidden(t *testing.T) {
	env := testenv.Default(20, 10)
	defer env.Close()

	// Create a box with OverflowAuto and ScrollbarY=true, but NO overflow.
	box := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(2)),
		}),
	).Style(style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(5)),
		OverflowY: style.Some(style.OverflowAuto),
	}).ScrollbarY(true)

	env.Mount(box)
	env.Flush()

	// Should NOT have a scrollbar at X=9.
	// We check if it is empty.
	testenv.ExpectScreen(t, env).
		CellAt(9, 0).ToHaveContent("")
}

func TestPaint_ScrollbarAutoShown(t *testing.T) {
	env := testenv.Default(20, 10)
	defer env.Close()

	// Create a box with OverflowAuto and ScrollbarY=true, WITH overflow.
	box := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(10)),
		}),
	).Style(style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(5)),
		OverflowY: style.Some(style.OverflowAuto),
	}).ScrollbarY(true)

	env.Mount(box)
	env.Flush()

	// SHOULD have a scrollbar at X=9.
	testenv.ExpectScreen(t, env).
		CellAt(9, 0).ToHaveContent("┃")
}

func TestPaint_ScrollbarWithBorderPadding(t *testing.T) {
	env := testenv.Default(20, 10)
	defer env.Close()

	// Create a box with border, padding and overflow.
	box := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(20)),
		}),
	).Style(style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(5)),
		Border:    style.SingleBorder().Some(),
		Padding:   style.Some(style.Edges(1)),
		OverflowY: style.Some(style.OverflowAuto),
	}).ScrollbarY(true)

	env.Mount(box)
	env.Flush()

	// Fragment width = 10.
	// Border width = 1 on each side.
	// trackX = origin.X + 10 - 1 (right border) - 1 = 8.
	// The track should be at X=8.
	testenv.ExpectScreen(t, env).
		CellAt(8, 1).ToHaveContent("┃") // Y=1 because Y=0 is top border
}
