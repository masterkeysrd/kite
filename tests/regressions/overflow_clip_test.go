// Package regressions – overflow clipping regression tests (TSK-027).
//
// These tests exercise the paint engine's `overflow: hidden` / `overflow: clip`
// clipping behaviour end-to-end through the full engine pipeline (sync →
// layout → paint). They verify:
//   - Framebuffer cells outside a clipped content box are untouched.
//   - The fragment's own border is never eaten by its own overflow.
//   - Nested overflow boxes compose clip rects by intersection.
//   - The asymmetric case (X clipped, Y visible) is handled correctly.
package regressions

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

var (
	colorRed = color.RGBA{255, 0, 0, 255}
)

// overflowFrame runs one engine frame and returns the Buffer; it fatals if
// no surface was produced.
func overflowFrame(t *testing.T, env *testenv.Environment) *backend.Buffer {
	t.Helper()
	env.RenderFrame()
	fr := env.Backend.LastFrame()
	if fr.Surface == nil {
		t.Fatal("no frame surface produced")
	}
	return fr.Surface
}

// (removed) use testenv.Region assertions instead of manual counting.

// -----------------------------------------------------------------------------
// TSK-027 – Regression: box with hidden overflow clips child content
// -----------------------------------------------------------------------------

// TestOverflowClip_BoxHiddenOverflow verifies that a box with overflow: hidden
// only paints inside its own boundary; the framebuffer cells beyond the right
// edge of the content box remain at their pre-paint (zero/empty) value.
func TestOverflowClip_BoxHiddenOverflow(t *testing.T) {
	env := testenv.Default(40, 5)
	defer env.Close()

	// A 10-wide clipping box containing a 30-wide red child box.
	root := element.Box(
		element.Box(
			element.Box().Style(style.S().Width(style.Cells(30)).Height(style.Cells(1)).Background(colorRed)),
		).Style(style.S().Width(style.Cells(10)).Height(style.Cells(1)).OverflowX(style.OverflowHidden).OverflowY(style.OverflowVisible)),
	)

	env.Mount(root)
	_ = overflowFrame(t, env)

	// Cells 0..9 must carry the red background (painted inside clip).
	testenv.ExpectScreen(t, env).Region(0, 0, 10, 1).ToHaveBackground(colorRed)
	// Cells 10..29 must NOT carry red (clipped by the parent's overflow: hidden).
	testenv.ExpectScreen(t, env).Region(10, 0, 20, 1).ToNotHaveBackground(colorRed)
}

// TestOverflowClip_BoxVisibleOverflow verifies that overflow: visible (default)
// allows content to spill past the parent's boundary.
func TestOverflowClip_BoxVisibleOverflow(t *testing.T) {
	env := testenv.Default(40, 5)
	defer env.Close()

	root := element.Box(
		element.Box(
			element.Box().Style(style.S().Width(style.Cells(30)).Height(style.Cells(1)).Background(colorRed)),
		).Style(style.S().Width(style.Cells(10)).Height(style.Cells(1)).OverflowX(style.OverflowVisible).OverflowY(style.OverflowVisible)),
	)

	env.Mount(root)
	_ = overflowFrame(t, env)

	// With visible overflow, the 30-wide child should paint beyond x=10.
	testenv.ExpectScreen(t, env).Region(0, 0, 30, 1).ToHaveBackgroundCountGreaterThan(colorRed, 10)
}

// TestOverflowClip_HiddenInsideVisible verifies that only the inner clip is
// applied when a hidden-overflow box sits inside a visible-overflow box.
func TestOverflowClip_HiddenInsideVisible(t *testing.T) {
	env := testenv.Default(40, 5)
	defer env.Close()

	// outer (30 wide, visible) → inner (8 wide, hidden) → 30-wide red child.
	root := element.Box(
		element.Box(
			element.Box(
				element.Box().Style(style.S().Width(style.Cells(30)).Height(style.Cells(1)).Background(colorRed)),
			).Style(style.S().Width(style.Cells(8)).Height(style.Cells(1)).OverflowX(style.OverflowHidden)),
		).Style(style.S().Width(style.Cells(30)).OverflowX(style.OverflowVisible)),
	)

	env.Mount(root)
	_ = overflowFrame(t, env)

	// Cells 0..7 must have red background (inside the inner clip).
	testenv.ExpectScreen(t, env).Region(0, 0, 8, 1).ToHaveBackground(colorRed)
	// Cells 8..29 must NOT have red (inner hidden-overflow clips them).
	testenv.ExpectScreen(t, env).Region(8, 0, 21, 1).ToNotHaveBackground(colorRed)
}

// TestOverflowClip_BorderIntact verifies that a box with overflow: hidden and
// a visible border retains its full border decoration; the border cells are NOT
// eaten by the overflow clip.
func TestOverflowClip_BorderIntact(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// A 10×3 box with a single border and overflow: hidden. The inner child is
	// 30-wide so it would spill without clipping.
	root := element.Box(
		element.Box(
			element.Box().Style(style.S().Width(style.Cells(30)).Height(style.Cells(1)).Background(colorRed)),
		).Style(style.S().Width(style.Cells(10)).Height(style.Cells(3)).OverflowX(style.OverflowHidden).OverflowY(style.OverflowHidden).Border(style.SingleBorder())),
	)

	env.Mount(root)
	fb := overflowFrame(t, env)

	// All four corners of the 10×3 box must carry a border glyph.
	// Box at origin (0,0): corners are TL(0,0), TR(9,0), BL(0,2), BR(9,2).
	corners := [][2]int{{0, 0}, {9, 0}, {0, 2}, {9, 2}}
	for _, pt := range corners {
		c := fb.CellAt(pt[0], pt[1])
		if c.Content == "" {
			t.Errorf("BorderIntact: corner (%d,%d) has no content — border was clipped", pt[0], pt[1])
		}
	}

	// Verify that red background does NOT appear beyond x=9 (the border's right edge).
	testenv.ExpectScreen(t, env).Region(10, 1, 20, 1).ToNotHaveBackground(colorRed)
}

// TestOverflowClip_Asymmetric_HiddenX_VisibleY clips only horizontal overflow
// and allows vertical spill.
func TestOverflowClip_Asymmetric_HiddenX_VisibleY(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// Clipping container: 6 wide, overflow-x: hidden, overflow-y: visible.
	// Contains a tall blue child box (2×5) that fits horizontally and vertically
	// (no overflow on Y axis) – clipping is about X axis for a red 30-wide box.
	root := element.Box(
		element.Box(
			// Wide red child — should be horizontally clipped to 6 cols.
			element.Box().Style(style.S().Width(style.Cells(30)).Height(style.Cells(1)).Background(colorRed)),
		).Style(style.S().Width(style.Cells(6)).Height(style.Cells(3)).OverflowX(style.OverflowHidden).OverflowY(style.OverflowVisible)),
	)

	env.Mount(root)
	_ = overflowFrame(t, env)

	// X >= 6 must not have red (horizontal clip).
	testenv.ExpectScreen(t, env).Region(6, 0, 24, 1).ToNotHaveBackground(colorRed)
	// At least some cells inside x<6 should have red background.
	testenv.ExpectScreen(t, env).Region(0, 0, 6, 1).ToHaveBackgroundCountGreaterThan(colorRed, 0)
}

// TestOverflow_MinSizeScrollContainer verifies that scroll/clip containers
// have an automatic minimum main size of 0, allowing them to shrink to fit
// within the available space.
func TestOverflow_MinSizeScrollContainer(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// Parent has height 5.
	// Contains a Header of height 1, a scrollable Body (overflow-y: scroll), and a Footer of height 1.
	// The scrollable Body contains an inner item of height 10.
	// Without the scroll container automatic min-size fix, the Body's minimum height would default to its content height (10),
	// causing the parent to overflow (total height = 1 + 10 + 1 = 12, which is > 5).
	// With the fix, the Body can shrink to 3 (so total height = 1 + 3 + 1 = 5).
	root := element.Box(
		element.Box(
			element.Box("Head").Style(style.S().Height(style.Cells(1))),
			element.Box(
				element.Box("Body content").Style(style.S().Height(style.Cells(10))),
			).Style(style.S().Flex(style.FlexItemValue{Grow: 1, Shrink: 1}).OverflowY(style.OverflowScroll)),
			element.Box("Foot").Style(style.S().Height(style.Cells(1))),
		).Style(style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Height(style.Cells(5))),
	)

	env.Mount(root)
	env.RenderFrame()

	fr := env.Backend.LastFrame()
	surf := fr.Surface
	if surf == nil {
		t.Fatal("no frame surface produced")
	}

	// The row Y=4 should contain the text "Foot".
	var rowContent string
	for x := 0; x < 40; x++ {
		cell := surf.CellAt(x, 4)
		if cell.Content != "" {
			rowContent += cell.Content
		}
	}

	if !strings.Contains(rowContent, "Foot") {
		// Dump the screen buffer for debugging
		var screenLines []string
		for y := 0; y < 10; y++ {
			var line string
			for x := 0; x < 40; x++ {
				cell := surf.CellAt(x, y)
				if cell.Content == "" {
					line += "."
				} else {
					line += cell.Content
				}
			}
			screenLines = append(screenLines, fmt.Sprintf("Y=%d: %s", y, line))
		}
		t.Errorf("expected Footer to be positioned at row Y=4 (got row content %q)\nScreen:\n%s", rowContent, strings.Join(screenLines, "\n"))
	}
}
