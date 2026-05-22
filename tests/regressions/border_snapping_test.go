package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func TestBorderSnapping_Integration(t *testing.T) {
	e := testenv.Default(20, 10)
	defer e.Close()

	// Create two boxes flush against each other.
	// Box 1: (0,0) to (10, 5)
	// Box 2: (0,5) to (10, 10)
	// They share the horizontal line at y=5 (relative to parent)
	// But in Kite, they are block elements, so Box 2 starts after Box 1.
	root := element.Box(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(3)),
			Border: style.SingleBorder().Some(),
		}),
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(10)),
			Height: style.Some(style.Cells(3)),
			Border: style.SingleBorder().Some(),
			Margin: style.Some(style.Edges(-1, 0, 0, 0)), // Overlap the border
		}),
	).Style(style.Style{
		Padding: style.Some(style.Edges(1)),
	})

	e.Mount(root)
	e.RenderFrame()

	testenv.ExpectScreen(t, e).
		CellAt(1, 3).ToHaveContent("├").
		CellAt(10, 3).ToHaveContent("┤")
}
