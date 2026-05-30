package paint

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/backend"
)

func setCell(pe *PaintEngine, fb Surface, x, y int, c Cell) {
	pe.rootSurface = fb
	if len(pe.clipStack) == 0 {
		pe.clipStack = append(pe.clipStack, fb.Bounds())
	}
	pe.setCell(x, y, c)
}

func TestResolveBorders(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)
	pe := NewPaintEngine()

	// Create a horizontal line at y=2
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	}

	// Create a vertical line at x=2
	for y := range 5 {
		setCell(pe, fb, 2, y, Cell{Cell: backend.Cell{Content: "│"}, BorderStyle: BorderSingle})
	}

	pe.resolveBorders(fb)

	// Check the intersection at (2, 2)
	c := fb.CellAt(2, 2)
	if c.Content != "┼" {
		t.Errorf("Expected intersection at (2,2) to be ┼, got %q", c.Content)
	}
}

func TestResolveBorders_Styles(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	pe := NewPaintEngine()

	// Double horizontal line
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "═"}, BorderStyle: BorderDouble})
	}

	// Double vertical line
	for y := range 5 {
		setCell(pe, fb, 2, y, Cell{Cell: backend.Cell{Content: "║"}, BorderStyle: BorderDouble})
	}

	pe.resolveBorders(fb)

	c := fb.CellAt(2, 2)
	if c.Content != "╬" {
		t.Errorf("Expected double intersection at (2,2) to be ╬, got %q", c.Content)
	}
}

func TestResolveBorders_MixedStyles(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	pe := NewPaintEngine()

	// Thick horizontal line
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "━"}, BorderStyle: BorderThick})
	}

	// Single vertical line
	for y := range 5 {
		setCell(pe, fb, 2, y, Cell{Cell: backend.Cell{Content: "│"}, BorderStyle: BorderSingle})
	}

	pe.resolveBorders(fb)

	// Rule 3: Only borders of the same type can be merged.
	// Since Horizontal is Thick and Vertical is Single, they don't merge.
	// (2, 2) was last painted as Single Vertical, so it remains "│"
	c := fb.CellAt(2, 2)
	if c.Content != "│" {
		t.Errorf("Expected mixed intersection NOT to merge (remain │), got %q", c.Content)
	}
}

func TestResolveBorders_RoundedTee(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	pe := NewPaintEngine()

	// Rounded horizontal line (top edge of a box)
	setCell(pe, fb, 1, 1, Cell{Cell: backend.Cell{Content: "╭"}, BorderStyle: BorderRounded})
	setCell(pe, fb, 2, 1, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderRounded})
	setCell(pe, fb, 3, 1, Cell{Cell: backend.Cell{Content: "╮"}, BorderStyle: BorderRounded})

	// Vertical lines to make them actual corners
	setCell(pe, fb, 1, 2, Cell{Cell: backend.Cell{Content: "│"}, BorderStyle: BorderRounded})
	setCell(pe, fb, 3, 2, Cell{Cell: backend.Cell{Content: "│"}, BorderStyle: BorderRounded})

	// Vertical line hitting the middle of the rounded horizontal line
	// Note: this is BorderSingle, while horizontal is BorderRounded.
	setCell(pe, fb, 2, 2, Cell{Cell: backend.Cell{Content: "│"}, BorderStyle: BorderSingle})

	pe.resolveBorders(fb)

	// The corner (1, 1) and (3, 1) should remain rounded if they are corners
	if fb.CellAt(1, 1).Content != "╭" {
		t.Errorf("Expected (1,1) to be ╭, got %q", fb.CellAt(1, 1).Content)
	}
	if fb.CellAt(3, 1).Content != "╮" {
		t.Errorf("Expected (3,1) to be ╮, got %q", fb.CellAt(3, 1).Content)
	}

	// Rule 3: Different types don't merge. (2,1) is Rounded, its neighbor (2,2) is Single.
	c := fb.CellAt(2, 1)
	if c.Content != "─" {
		t.Errorf("Expected junction (2,1) NOT to merge (remain ─), got %q", c.Content)
	}
}

func TestResolveBorders_NoMangleText(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 3, 3)

	pe := NewPaintEngine()

	// Border horizontal line at y=1
	setCell(pe, fb, 0, 1, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	setCell(pe, fb, 1, 1, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	setCell(pe, fb, 2, 1, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})

	// User text "|" at (1, 0) and (1, 2) - NO BorderStyle
	setCell(pe, fb, 1, 0, Cell{Cell: backend.Cell{Content: "|"}})
	setCell(pe, fb, 1, 2, Cell{Cell: backend.Cell{Content: "|"}})

	pe.resolveBorders(fb)

	// The center (1, 1) should remain "─" because its vertical neighbors don't have a BorderStyle
	c := fb.CellAt(1, 1)
	if c.Content != "─" {
		t.Errorf("Expected (1,1) to remain ─, got %q", c.Content)
	}
}

func TestResolveBorders_ParallelNoMerge(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	pe := NewPaintEngine()

	// Line 1: y=1
	for x := range 5 {
		setCell(pe, fb, x, 1, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	}
	// Line 2: y=2
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	}

	pe.resolveBorders(fb)

	// They should remain "─" and not become "┬" or "┴" or "┼"
	for x := range 5 {
		if fb.CellAt(x, 1).Content != "─" {
			t.Errorf("Expected (x=%d, y=1) to remain ─, got %q", x, fb.CellAt(x, 1).Content)
		}
		if fb.CellAt(x, 2).Content != "─" {
			t.Errorf("Expected (x=%d, y=2) to remain ─, got %q", x, fb.CellAt(x, 2).Content)
		}
	}
}

func TestResolveBorders_SameBackgroundOnly(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	blue := color.RGBA{0, 0, 255, 255}
	red := color.RGBA{255, 0, 0, 255}

	pe := NewPaintEngine()

	// Horizontal line: y=2, Blue background
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "─", Bg: blue}, BorderStyle: BorderSingle})
	}

	// Vertical line: x=2, Red background
	for y := range 5 {
		setCell(pe, fb, 2, y, Cell{Cell: backend.Cell{Content: "│", Bg: red}, BorderStyle: BorderSingle})
	}

	pe.resolveBorders(fb)

	// Since they have different backgrounds, they shouldn't merge.
	c := fb.CellAt(2, 2)
	if c.Content == "┼" {
		t.Errorf("Expected intersection at (2,2) NOT to be ┼ due to different backgrounds")
	}
}

func TestResolveBorders_SameTypeOnly(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	pe := NewPaintEngine()

	// Horizontal line: Single
	for x := range 5 {
		setCell(pe, fb, x, 2, Cell{Cell: backend.Cell{Content: "─"}, BorderStyle: BorderSingle})
	}

	// Vertical line: Double
	for y := range 5 {
		setCell(pe, fb, 2, y, Cell{Cell: backend.Cell{Content: "║"}, BorderStyle: BorderDouble})
	}

	pe.resolveBorders(fb)

	c := fb.CellAt(2, 2)
	if c.Content == "╪" || c.Content == "╫" || c.Content == "╬" {
		t.Errorf("Expected intersection at (2,2) NOT to merge due to different types, got %q", c.Content)
	}
}

func TestResolveBorders_SameColorOnly(t *testing.T) {
	fb := NewFrameBuffer(0, 0, 5, 5)

	blue := color.RGBA{0, 0, 255, 255}
	red := color.RGBA{255, 0, 0, 255}

	// Horizontal line: y=2, Blue foreground
	for x := range 5 {
		fb.Set(x, 2, Cell{Cell: backend.Cell{Content: "─", Fg: blue}, BorderStyle: BorderSingle})
	}

	// Vertical line: x=2, Red foreground
	for y := range 5 {
		fb.Set(2, y, Cell{Cell: backend.Cell{Content: "│", Fg: red}, BorderStyle: BorderSingle})
	}

	pe := NewPaintEngine()
	pe.resolveBorders(fb)

	c := fb.CellAt(2, 2)
	if c.Content == "┼" {
		t.Errorf("Expected intersection at (2,2) NOT to be ┼ due to different colors")
	}
}
