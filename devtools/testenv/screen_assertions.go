package testenv

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/paint"
)

// ScreenAssertion provides fluent assertions over a rendered framebuffer.
type ScreenAssertion struct {
	t  *testing.T
	fb paint.Surface
}

// ExpectScreen starts an assertion chain for the rendered framebuffer.
func ExpectScreen(t *testing.T, env *Environment) *ScreenAssertion {
	t.Helper()
	fb := env.Backend.LastFrame().Surface
	if fb == nil {
		t.Fatal("No frame was produced by testenv")
	}
	return &ScreenAssertion{t: t, fb: fb}
}

// CellAssertion scopes assertions to a specific cell coordinate.
type CellAssertion struct {
	screen *ScreenAssertion
	x, y   int
	cell   paint.Cell
}

// CellAt scopes the next assertions to a specific X,Y coordinate.
func (s *ScreenAssertion) CellAt(x, y int) *CellAssertion {
	s.t.Helper()
	cell := s.fb.CellAt(x, y)
	return &CellAssertion{screen: s, x: x, y: y, cell: cell}
}

// ToHaveContent checks the text/rune inside the cell and returns to the ScreenAssertion for chaining.
func (c *CellAssertion) ToHaveContent(expected string) *ScreenAssertion {
	c.screen.t.Helper()

	if c.cell.Content != expected {
		c.screen.t.Errorf("Expected cell at (%d,%d) to be %q, got %q", c.x, c.y, expected, c.cell.Content)
	}
	return c.screen
}

// ToHaveAttribute asserts that the cell has the given attribute.
func (c *CellAssertion) ToHaveAttribute(attr paint.CellAttrs) *ScreenAssertion {
	c.screen.t.Helper()
	if c.cell.Attrs&attr == 0 {
		c.screen.t.Errorf("cell at (%d,%d) expected to have attribute %v, got %v", c.x, c.y, attr, c.cell.Attrs)
	}
	return c.screen
}

// ToNotHaveAttribute asserts that the cell does NOT have the given attribute.
func (c *CellAssertion) ToNotHaveAttribute(attr paint.CellAttrs) *ScreenAssertion {
	c.screen.t.Helper()
	if c.cell.Attrs&attr != 0 {
		c.screen.t.Errorf("cell at (%d,%d) expected NOT to have attribute %v, but it did", c.x, c.y, attr)
	}
	return c.screen
}

// RegionAssertion scopes assertions to a rectangular region on the framebuffer.
type RegionAssertion struct {
	screen *ScreenAssertion
	rect   geom.Rect
}

// RegionRect scopes assertions to a geom.Rect region.
func (s *ScreenAssertion) RegionRect(r geom.Rect) *RegionAssertion {
	s.t.Helper()
	return &RegionAssertion{screen: s, rect: r}
}

// Region is a convenience that accepts raw coordinates and size.
func (s *ScreenAssertion) Region(x, y, w, h int) *RegionAssertion {
	s.t.Helper()
	return s.RegionRect(geom.Rect{Origin: geom.Point{X: x, Y: y}, Size: geom.Size{Width: w, Height: h}})
}

// ToHaveBackground asserts that EVERY cell in the region has the specified background.
func (r *RegionAssertion) ToHaveBackground(expected color.Color) *ScreenAssertion {
	r.screen.t.Helper()
	fb := r.screen.fb
	for cy := r.rect.Origin.Y; cy < r.rect.Origin.Y+r.rect.Size.Height; cy++ {
		for cx := r.rect.Origin.X; cx < r.rect.Origin.X+r.rect.Size.Width; cx++ {
			actual := fb.CellAt(cx, cy).BG
			if actual != expected {
				r.screen.t.Errorf("cell (%d,%d) expected bg %v, got %v", cx, cy, expected, actual)
			}
		}
	}
	return r.screen
}

// ToNotHaveBackground asserts that NO cell in the region has the specified background.
func (r *RegionAssertion) ToNotHaveBackground(unexpected color.Color) *ScreenAssertion {
	r.screen.t.Helper()
	fb := r.screen.fb
	for cy := r.rect.Origin.Y; cy < r.rect.Origin.Y+r.rect.Size.Height; cy++ {
		for cx := r.rect.Origin.X; cx < r.rect.Origin.X+r.rect.Size.Width; cx++ {
			actual := fb.CellAt(cx, cy).BG
			if actual == unexpected {
				r.screen.t.Errorf("cell (%d,%d) was clipped but had unexpected bg %v", cx, cy, actual)
			}
		}
	}
	return r.screen
}

// ToHaveBackgroundCountGreaterThan asserts that the number of cells in the region
// carrying the specified background color is strictly greater than `min`.
func (r *RegionAssertion) ToHaveBackgroundCountGreaterThan(expected color.Color, min int) *ScreenAssertion {
	r.screen.t.Helper()
	fb := r.screen.fb
	count := 0
	for cy := r.rect.Origin.Y; cy < r.rect.Origin.Y+r.rect.Size.Height; cy++ {
		for cx := r.rect.Origin.X; cx < r.rect.Origin.X+r.rect.Size.Width; cx++ {
			if fb.CellAt(cx, cy).BG == expected {
				count++
			}
		}
	}
	if count <= min {
		r.screen.t.Errorf("expected more than %d cells with bg %v, got %d", min, expected, count)
	}
	return r.screen
}
