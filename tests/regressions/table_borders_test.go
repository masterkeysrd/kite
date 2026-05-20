package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestTableBorders(t *testing.T) {
	b := mock.New(40, 10)
	eng := engine.New(b, engine.Options{})

	// Table with borders on every cell, no manual margins
	root := element.Box(
		element.Table(
			element.TR(
				element.TD("A").Style(style.Style{Border: style.SingleBorder().Some()}),
				element.TD("B").Style(style.Style{Border: style.SingleBorder().Some()}),
			),
			element.TR(
				element.TD("C").Style(style.Style{Border: style.SingleBorder().Some()}),
				element.TD("D").Style(style.Style{Border: style.SingleBorder().Some()}),
			),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Padding: style.Some(style.Edges(1)),
	})

	eng.Mount(root)
	eng.Frame()

	fb := b.LastFrame().Surface
	if fb == nil {
		t.Fatal("No frame produced")
	}

	// If junctions are working, the middle point should be ┼
	// We need to find where the middle point is.
	// Table at y=1 (padding). TR 1 height depends on content.
	// "A" is content. 1 cell high. Total TD height = 1 + border (1 top + 1 bottom) = 3 cells.
	// TR 1 height = 3.
	// TD 1 (A) at (1, 1), size (ColWidth, 3).
	// TD 2 (B) at (1+ColWidth, 1), size (ColWidth, 3).
	// Border of TD 1 bottom is at y = 1 + 3 - 1 = 3.
	// Border of TD 3 (C) top is at y = 1 + 3 = 4.
	// Wait, the TableAlgorithm doesn't seem to overlap borders.

	// Let's print the surface to see what's going on.
	for y := 0; y < 10; y++ {
		line := ""
		for x := 0; x < 40; x++ {
			c := fb.CellAt(x, y).Content
			if c == "" {
				line += " "
			} else {
				line += c
			}
		}
		t.Logf("%02d: %s", y, line)
	}
}
