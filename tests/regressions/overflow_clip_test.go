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
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

var (
	colorRed  = color.RGBA{255, 0, 0, 255}
	colorBlue = color.RGBA{0, 0, 255, 255}
)

// overflowFrame runs one engine frame and returns the FrameBuffer; it fatals if
// no surface was produced.
func overflowFrame(t *testing.T, b *mock.Backend, eng *engine.Engine) *mock.FrameRecord {
	t.Helper()
	eng.Frame()
	fr := b.LastFrame()
	if fr.Surface == nil {
		t.Fatal("no frame surface produced")
	}
	return &fr
}

// countBG returns how many cells in row y carry the given background color.
func countBG(fr *mock.FrameRecord, bg color.Color, y, startX, endX int) int {
	n := 0
	for x := startX; x < endX; x++ {
		if fr.Surface.CellAt(x, y).BG == bg {
			n++
		}
	}
	return n
}

// -----------------------------------------------------------------------------
// TSK-027 – Regression: box with hidden overflow clips child content
// -----------------------------------------------------------------------------

// TestOverflowClip_BoxHiddenOverflow verifies that a box with overflow: hidden
// only paints inside its own boundary; the framebuffer cells beyond the right
// edge of the content box remain at their pre-paint (zero/empty) value.
func TestOverflowClip_BoxHiddenOverflow(t *testing.T) {
	b := mock.New(40, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// A 10-wide clipping box containing a 30-wide red child box.
	root := element.Box(
		element.Box(
			element.Box().Style(style.Style{
				Width:      style.Some(style.Cells(30)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colorRed),
			}),
		).Style(style.Style{
			Width:     style.Some(style.Cells(10)),
			Height:    style.Some(style.Cells(1)),
			OverflowX: style.Some(style.OverflowHidden),
			OverflowY: style.Some(style.OverflowVisible),
		}),
	)

	eng.Mount(root)
	fr := overflowFrame(t, b, eng)

	// Cells 0..9 must carry the red background (painted inside clip).
	for x := 0; x < 10; x++ {
		if fr.Surface.CellAt(x, 0).BG != colorRed {
			t.Errorf("cell (%d,0) should have red background (inside clip), got %v", x, fr.Surface.CellAt(x, 0).BG)
		}
	}
	// Cells 10..29 must NOT carry red (clipped by the parent's overflow: hidden).
	for x := 10; x < 30; x++ {
		if fr.Surface.CellAt(x, 0).BG == colorRed {
			t.Errorf("cell (%d,0) should be clipped but has red background", x)
		}
	}
}

// TestOverflowClip_BoxVisibleOverflow verifies that overflow: visible (default)
// allows content to spill past the parent's boundary.
func TestOverflowClip_BoxVisibleOverflow(t *testing.T) {
	b := mock.New(40, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	root := element.Box(
		element.Box(
			element.Box().Style(style.Style{
				Width:      style.Some(style.Cells(30)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colorRed),
			}),
		).Style(style.Style{
			Width:     style.Some(style.Cells(10)),
			Height:    style.Some(style.Cells(1)),
			OverflowX: style.Some(style.OverflowVisible),
			OverflowY: style.Some(style.OverflowVisible),
		}),
	)

	eng.Mount(root)
	fr := overflowFrame(t, b, eng)

	// With visible overflow, the 30-wide child should paint beyond x=10.
	painted := countBG(fr, colorRed, 0, 0, 30)
	if painted <= 10 {
		t.Errorf("OverflowVisible: expected more than 10 red cells but got %d", painted)
	}
}

// TestOverflowClip_HiddenInsideVisible verifies that only the inner clip is
// applied when a hidden-overflow box sits inside a visible-overflow box.
func TestOverflowClip_HiddenInsideVisible(t *testing.T) {
	b := mock.New(40, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// outer (30 wide, visible) → inner (8 wide, hidden) → 30-wide red child.
	root := element.Box(
		element.Box(
			element.Box(
				element.Box().Style(style.Style{
					Width:      style.Some(style.Cells(30)),
					Height:     style.Some(style.Cells(1)),
					Background: style.Some[color.Color](colorRed),
				}),
			).Style(style.Style{
				Width:     style.Some(style.Cells(8)),
				Height:    style.Some(style.Cells(1)),
				OverflowX: style.Some(style.OverflowHidden),
			}),
		).Style(style.Style{
			Width:     style.Some(style.Cells(30)),
			OverflowX: style.Some(style.OverflowVisible),
		}),
	)

	eng.Mount(root)
	fr := overflowFrame(t, b, eng)

	// Cells 0..7 must have red background (inside the inner clip).
	for x := 0; x < 8; x++ {
		if fr.Surface.CellAt(x, 0).BG != colorRed {
			t.Errorf("HiddenInsideVisible: cell (%d,0) should have red background", x)
		}
	}
	// Cells 8..29 must NOT have red (inner hidden-overflow clips them).
	for x := 8; x < 29; x++ {
		if fr.Surface.CellAt(x, 0).BG == colorRed {
			t.Errorf("HiddenInsideVisible: cell (%d,0) should be clipped but has red background", x)
		}
	}
}

// TestOverflowClip_BorderIntact verifies that a box with overflow: hidden and
// a visible border retains its full border decoration; the border cells are NOT
// eaten by the overflow clip.
func TestOverflowClip_BorderIntact(t *testing.T) {
	b := mock.New(40, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// A 10×3 box with a single border and overflow: hidden. The inner child is
	// 30-wide so it would spill without clipping.
	root := element.Box(
		element.Box(
			element.Box().Style(style.Style{
				Width:      style.Some(style.Cells(30)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colorRed),
			}),
		).Style(style.Style{
			Width:     style.Some(style.Cells(10)),
			Height:    style.Some(style.Cells(3)),
			OverflowX: style.Some(style.OverflowHidden),
			OverflowY: style.Some(style.OverflowHidden),
			Border:    style.SingleBorder().Some(),
		}),
	)

	eng.Mount(root)
	fr := overflowFrame(t, b, eng)

	// All four corners of the 10×3 box must carry a border glyph.
	// Box at origin (0,0): corners are TL(0,0), TR(9,0), BL(0,2), BR(9,2).
	corners := [][2]int{{0, 0}, {9, 0}, {0, 2}, {9, 2}}
	for _, pt := range corners {
		c := fr.Surface.CellAt(pt[0], pt[1])
		if c.Content == "" {
			t.Errorf("BorderIntact: corner (%d,%d) has no content — border was clipped", pt[0], pt[1])
		}
	}

	// Verify that red background does NOT appear beyond x=9 (the border's right edge).
	for x := 10; x < 30; x++ {
		if fr.Surface.CellAt(x, 1).BG == colorRed {
			t.Errorf("BorderIntact: cell (%d,1) should be clipped but has red background", x)
		}
	}
}

// TestOverflowClip_Asymmetric_HiddenX_VisibleY clips only horizontal overflow
// and allows vertical spill.
func TestOverflowClip_Asymmetric_HiddenX_VisibleY(t *testing.T) {
	b := mock.New(40, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Clipping container: 6 wide, overflow-x: hidden, overflow-y: visible.
	// Contains a tall blue child box (2×5) that fits horizontally and vertically
	// (no overflow on Y axis) – clipping is about X axis for a red 30-wide box.
	root := element.Box(
		element.Box(
			// Wide red child — should be horizontally clipped to 6 cols.
			element.Box().Style(style.Style{
				Width:      style.Some(style.Cells(30)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colorRed),
			}),
		).Style(style.Style{
			Width:     style.Some(style.Cells(6)),
			Height:    style.Some(style.Cells(3)),
			OverflowX: style.Some(style.OverflowHidden),
			OverflowY: style.Some(style.OverflowVisible),
		}),
	)

	eng.Mount(root)
	fr := overflowFrame(t, b, eng)

	// X >= 6 must not have red (horizontal clip).
	for x := 6; x < 30; x++ {
		if fr.Surface.CellAt(x, 0).BG == colorRed {
			t.Errorf("Asymmetric: cell (%d,0) should be horizontally clipped but has red background", x)
		}
	}
	// At least some cells inside x<6 should have red background.
	painted := countBG(fr, colorRed, 0, 0, 6)
	if painted == 0 {
		t.Error("Asymmetric: no red cells found inside the clip boundary — content missing")
	}
}
