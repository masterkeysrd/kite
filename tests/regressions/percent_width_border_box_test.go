// Regression tests for layout percent resolution — covers TSK-041.
//
// Verifies that KindPercent widths resolve against the parent's content-box
// (ContainerSpace), not the border-box. This matches standard CSS semantics:
// percentage widths always resolve against the containing block's content area.
package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

var colorGreen = color.RGBA{0, 200, 0, 255}

// TestRegression_PercentWidthContentBox verifies that a child with width:50%
// resolves against the parent's content-box, not the border-box.
//
// Layout tree (viewport 40×5):
//
//	outer  – width:20, border:single(1px each)  → border-box=20, content=18
//	  inner – width:50%                          → must be 9 (50% of content 18)
//
// inner is placed at x=1 (border), y=1 (border).
// Green cells [1..9] at row 1 confirm the width is 9 (content-box percent).
// Cells at x=10..17 inside the content area must NOT be green.
func TestRegression_PercentWidthContentBox(t *testing.T) {
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

	// content-box = 20 - 2 (border L+R) = 18 → 50% = 9
	// inner starts at x=1 (border), y=1 (border); width=9 → cols 1..9 are green.
	testenv.ExpectScreen(t, env).Region(1, 1, 9, 1).ToHaveBackground(colorGreen)
	// cols 10..18 inside the content area must NOT be green.
	testenv.ExpectScreen(t, env).Region(10, 1, 9, 1).ToNotHaveBackground(colorGreen)
}

// TestRegression_PercentWidth100_NoOverflow verifies that width:100% on a child
// fills exactly the parent's content area and does NOT overflow past the right border.
//
// This is the core regression from the kite-dump.json: title and flex containers
// with width:100% were overflowing the card's right border because percent was
// resolved against the border-box (55) instead of the content-box (49).
func TestRegression_PercentWidth100_NoOverflow(t *testing.T) {
	env := testenv.Default(40, 5)
	defer env.Close()

	// inner fills 100% of outer's content area.  With border=1 and padding=2 on each
	// side, the content area is 20-2-4=14.  Green must stop at x=14 (1 border + 14
	// content = x=1..14); the border at x=19 must still be a border glyph (not green).
	inner := element.Box().Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Cells(1)),
		Background: style.Some[color.Color](colorGreen),
	})

	outer := element.Box(inner).Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Border:  style.Some(style.SingleBorder()),
		Padding: style.Some(style.Edges(2)),
	})

	env.Mount(element.Box(outer))
	env.RenderFrame()

	// content = 20-2-4=14; inner at x=3 (border+padding), width=14 → cols 3..16 green.
	testenv.ExpectScreen(t, env).Region(3, 3, 14, 1).ToHaveBackground(colorGreen)
	// The right border column (x=19) must NOT be overwritten with green.
	testenv.ExpectScreen(t, env).Region(17, 3, 3, 1).ToNotHaveBackground(colorGreen)
}

// TestRegression_NestedPercentResolution verifies correct cascading of
// ContainerSpace across three nesting levels.
//
// Layout tree (viewport 60×5):
//
//	level1 – width:40, no border/padding  → content=40
//	  level2 – width:50%                  → must be 20 (50% of 40)
//	    level3 – width:50%                → must be 10 (50% of 20)
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

	// level2=20, level3=10; all start at x=0 (no borders/padding).
	testenv.ExpectScreen(t, env).Region(0, 0, 10, 1).ToHaveBackground(colorGreen)
	testenv.ExpectScreen(t, env).Region(10, 0, 10, 1).ToNotHaveBackground(colorGreen)
}
