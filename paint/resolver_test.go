package paint

import (
	"testing"
)

func TestResolveBorders(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Create a horizontal line at y=2
	for x := 0; x < 5; x++ {
		fb.Set(x, 2, Cell{Content: "─", Attrs: FlagIsBorder})
	}

	// Create a vertical line at x=2
	for y := 0; y < 5; y++ {
		fb.Set(2, y, Cell{Content: "│", Attrs: FlagIsBorder})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// Check the intersection at (2, 2)
	c := fb.CellAt(2, 2)
	if c.Content != "┼" {
		t.Errorf("Expected intersection at (2,2) to be ┼, got %q", c.Content)
	}

	// Check some other points
	if fb.CellAt(2, 0).Content != "│" {
		t.Errorf("Expected (2,0) to be │, got %q", fb.CellAt(2, 0).Content)
	}
	if fb.CellAt(0, 2).Content != "─" {
		t.Errorf("Expected (0,2) to be ─, got %q", fb.CellAt(0, 2).Content)
	}
}

func TestResolveBorders_Styles(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Double horizontal line
	for x := 0; x < 5; x++ {
		fb.Set(x, 2, Cell{Content: "═", Attrs: FlagIsBorder})
	}

	// Double vertical line
	for y := 0; y < 5; y++ {
		fb.Set(2, y, Cell{Content: "║", Attrs: FlagIsBorder})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	c := fb.CellAt(2, 2)
	if c.Content != "╬" {
		t.Errorf("Expected double intersection at (2,2) to be ╬, got %q", c.Content)
	}
}

func TestResolveBorders_NoMangleText(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 3, 3)

	// Border horizontal line at y=1
	fb.Set(0, 1, Cell{Content: "─", Attrs: FlagIsBorder})
	fb.Set(1, 1, Cell{Content: "─", Attrs: FlagIsBorder})
	fb.Set(2, 1, Cell{Content: "─", Attrs: FlagIsBorder})

	// User text "|" at (1, 0) and (1, 2) - NO FlagIsBorder
	fb.Set(1, 0, Cell{Content: "|"})
	fb.Set(1, 2, Cell{Content: "|"})

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// The center (1, 1) should remain "─" because its vertical neighbors don't have FlagIsBorder
	c := fb.CellAt(1, 1)
	if c.Content != "─" {
		t.Errorf("Expected (1,1) to remain ─, got %q", c.Content)
	}
}
