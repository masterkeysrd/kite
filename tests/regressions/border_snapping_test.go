package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
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
		element.Box().Style(style.S().Width(style.Cells(10)).Height(style.Cells(3)).Border(style.SingleBorder())),
		element.Box().Style(style.S().Width(style.Cells(10)).Height(style.Cells(3)).Border(style.SingleBorder()).Margin(-1, 0, 0, 0)),
	).Style(style.S().Padding(1))

	e.Mount(root)
	e.RenderFrame()

	testenv.ExpectScreen(t, e).
		CellAt(1, 3).ToHaveContent("├").
		CellAt(10, 3).ToHaveContent("┤")
}
