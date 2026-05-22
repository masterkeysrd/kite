package testenv

import (
	"testing"

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
