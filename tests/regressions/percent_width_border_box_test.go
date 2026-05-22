// Regression tests for layout percent resolution — covers TSK-041.
//
// Verifies that KindPercent widths resolve against the parent's border-box
// (ContainingSpace), not the content-box, matching ADR-018 semantics.
package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

var colorGreen = color.RGBA{0, 200, 0, 255}

// TestRegression_PercentWidthBorderBox verifies that a child with width:50%
// resolves against the parent's border-box (not the content-box).
//
// Layout tree (viewport 40×5):
//
//	outer  – width:20, border:single(1px each)  → border-box=20, content=18
//	  inner – width:50%                          → must be 10 (50% of 20)
//
// inner is placed at x=1 (left border), y=1 (top border).
// Green cells [1..10] at row 1 confirm the width is 10.
func TestRegression_PercentWidthBorderBox(t *testing.T) {
	env := testenv.Default(40, 5)
	defer env.Close()

	inner := element.Box().Style(style.Style{
		Width:      style.Some(style.Percent(50)),
		Height:     style.Some(style.Cells(1)),
		Background: style.Some[color.Color](colorGreen),
	})

	outer := element.Box(inner).Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Border: style.Some(style.SingleBorder()),
	})

	env.Mount(element.Box(outer))
	env.RenderFrame()

	// inner starts at x=1 (border), y=1 (border); width=10 → cols 1..10 are green.
	testenv.ExpectScreen(t, env).Region(1, 1, 10, 1).ToHaveBackground(colorGreen)
	// cols 11..18 inside the content area must NOT be green.
	testenv.ExpectScreen(t, env).Region(11, 1, 8, 1).ToNotHaveBackground(colorGreen)
}

// TestRegression_NestedPercentResolution verifies correct cascading of
// ContainingSpace across three nesting levels (ADR-018).
//
// Layout tree (viewport 60×10):
//
//	level1 – width:40, no border/padding  → border-box=40, content=40
//	  level2 – width:50%                  → must be 20 (50% of 40)
//	    level3 – width:50%                → must be 10 (50% of 20)
//
// level3 is painted green starting at x=0 (no border/padding anywhere).
// Green cells [0..9] at row 0 confirm the width is 10.
func TestRegression_NestedPercentResolution(t *testing.T) {
	env := testenv.Default(60, 5)
	defer env.Close()

	level3 := element.Box().Style(style.Style{
		Width:      style.Some(style.Percent(50)),
		Height:     style.Some(style.Cells(1)),
		Background: style.Some[color.Color](colorGreen),
	})

	level2 := element.Box(level3).Style(style.Style{
		Width:  style.Some(style.Percent(50)),
		Height: style.Some(style.Cells(1)),
	})

	level1 := element.Box(level2).Style(style.Style{
		Width:  style.Some(style.Cells(40)),
		Height: style.Some(style.Cells(1)),
	})

	env.Mount(element.Box(level1))
	env.RenderFrame()

	// level2 = 20, level3 = 10; both start at x=0 (no borders/padding).
	testenv.ExpectScreen(t, env).Region(0, 0, 10, 1).ToHaveBackground(colorGreen)
	testenv.ExpectScreen(t, env).Region(10, 0, 10, 1).ToNotHaveBackground(colorGreen)
}
