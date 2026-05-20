package paint

import (
	"testing"
)

func TestResolveBorders(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Create a horizontal line at y=2
	for x := 0; x < 5; x++ {
		fb.Set(x, 2, Cell{Content: "─", BorderStyle: BorderSingle})
	}

	// Create a vertical line at x=2
	for y := 0; y < 5; y++ {
		fb.Set(2, y, Cell{Content: "│", BorderStyle: BorderSingle})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// Check the intersection at (2, 2)
	c := fb.CellAt(2, 2)
	if c.Content != "┼" {
		t.Errorf("Expected intersection at (2,2) to be ┼, got %q", c.Content)
	}
}

func TestResolveBorders_Styles(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Double horizontal line
	for x := 0; x < 5; x++ {
		fb.Set(x, 2, Cell{Content: "═", BorderStyle: BorderDouble})
	}

	// Double vertical line
	for y := 0; y < 5; y++ {
		fb.Set(2, y, Cell{Content: "║", BorderStyle: BorderDouble})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	c := fb.CellAt(2, 2)
	if c.Content != "╬" {
		t.Errorf("Expected double intersection at (2,2) to be ╬, got %q", c.Content)
	}
}

func TestResolveBorders_MixedStyles(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Thick horizontal line
	for x := 0; x < 5; x++ {
		fb.Set(x, 2, Cell{Content: "━", BorderStyle: BorderThick})
	}

	// Single vertical line
	for y := 0; y < 5; y++ {
		fb.Set(2, y, Cell{Content: "│", BorderStyle: BorderSingle})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// Heaviest style wins: Thick should win over Single
	c := fb.CellAt(2, 2)
	if c.Content != "╋" {
		t.Errorf("Expected mixed intersection (Thick wins) at (2,2) to be ╋, got %q", c.Content)
	}
}

func TestResolveBorders_RoundedTee(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	// Rounded horizontal line (top edge of a box)
	fb.Set(1, 1, Cell{Content: "╭", BorderStyle: BorderRounded})
	fb.Set(2, 1, Cell{Content: "─", BorderStyle: BorderRounded})
	fb.Set(3, 1, Cell{Content: "╮", BorderStyle: BorderRounded})

	// Vertical lines to make them actual corners
	fb.Set(1, 2, Cell{Content: "│", BorderStyle: BorderRounded})
	fb.Set(3, 2, Cell{Content: "│", BorderStyle: BorderRounded})

	// Vertical line hitting the middle of the rounded horizontal line
	fb.Set(2, 2, Cell{Content: "│", BorderStyle: BorderSingle})

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// The corner (1, 1) and (3, 1) should remain rounded if they are corners
	if fb.CellAt(1, 1).Content != "╭" {
		t.Errorf("Expected (1,1) to be ╭, got %q", fb.CellAt(1, 1).Content)
	}
	if fb.CellAt(3, 1).Content != "╮" {
		t.Errorf("Expected (3,1) to be ╮, got %q", fb.CellAt(3, 1).Content)
	}

	// The junction (2, 1) is now a T-junction. It should fall back to Single style (┬)
	// because Rounded doesn't have T-junctions.
	c := fb.CellAt(2, 1)
	if c.Content != "┬" {
		t.Errorf("Expected junction (2,1) to fall back to ┬, got %q", c.Content)
	}
}

func TestResolveBorders_NoMangleText(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 3, 3)

	// Border horizontal line at y=1
	fb.Set(0, 1, Cell{Content: "─", BorderStyle: BorderSingle})
	fb.Set(1, 1, Cell{Content: "─", BorderStyle: BorderSingle})
	fb.Set(2, 1, Cell{Content: "─", BorderStyle: BorderSingle})

	// User text "|" at (1, 0) and (1, 2) - NO BorderStyle
	fb.Set(1, 0, Cell{Content: "|"})
	fb.Set(1, 2, Cell{Content: "|"})

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	// The center (1, 1) should remain "─" because its vertical neighbors don't have a BorderStyle
	c := fb.CellAt(1, 1)
	if c.Content != "─" {
		t.Errorf("Expected (1,1) to remain ─, got %q", c.Content)
	}
}
