package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestBorderSnapping_Integration(t *testing.T) {
	b := mock.New(20, 10)
	eng := engine.New(b, engine.Options{})

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

	eng.Mount(root)
	eng.Frame()

	// The overlap happens at y=3 (1 cell padding + 3 cells height - 1 cell overlap)
	// Wait, Box 1 is at y=1 (padding). Height 3. Bottom border is at y=3.
	// Box 2 would normally start at y=4. With margin-top: -1, it starts at y=3.

	// Let's check the cells at the junction.
	// (1, 3) should be ├
	// (10, 3) should be ┤
	// Cells between (2, 3) and (9, 3) should be ─ (actually they are overlapped)

	fb := b.LastFrame().Surface
	if fb == nil {
		t.Fatal("No frame was produced")
	}

	// Junction on the left
	cLeft := fb.CellAt(1, 3)
	if cLeft.Content != "├" {
		t.Errorf("Expected left junction at (1,3) to be ├, got %q", cLeft.Content)
	}

	// Junction on the right
	cRight := fb.CellAt(10, 3)
	if cRight.Content != "┤" {
		t.Errorf("Expected right junction at (10,3) to be ┤, got %q", cRight.Content)
	}
}
