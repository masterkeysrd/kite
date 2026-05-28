package regressions

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

// TestFlexWrapMinWidthAuto verifies that flex items with default min-width (auto)
// do not shrink below their content size and that rounding errors in space
// distribution are handled correctly.
func TestFlexWrapMinWidthAuto(t *testing.T) {
	// Total width 64
	env := testenv.Default(64, 20)
	defer env.Close()

	// 6 items with base-size 10 and min-size auto (which should resolve to content size).
	// Content size for "Flex Item X" is roughly 11 cells.
	// Total requested per line if 3 items: (11*3) + (gap 1 * 2) = 35.
	// 35 < 64, so they should fit.
	flexItems := make([]any, 0, 6)
	for i := 1; i <= 6; i++ {
		item := element.Box(fmt.Sprintf("Flex Item %d", i)).Style(style.Style{
			Width:  style.Some(style.Cells(13)), // Base size
			Height: style.Some(style.Cells(3)),
			Border: style.SingleBorder().Some(),
			Flex:   style.Some(style.Flex(1, 1, style.Cells(10))),
		}).WithID(fmt.Sprintf("item-%d", i))
		flexItems = append(flexItems, item)
	}

	root := element.Box(
		element.Box(
			flexItems...,
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRow),
			FlexWrap:      style.Some(style.FlexWrapOn),
			Width:         style.Some(style.Percent(100)),
			Padding:       style.Some(style.Edges(1)),
			Gap:           style.Some(style.Gap(1, 2)),
		}).WithID("container"),
	).Style(style.Style{
		Width:  style.Some(style.Percent(80)), // 80% of 64 = 51 cells
		Margin: style.Some(style.Edges(1, 2)),
	})

	env.Mount(root)
	env.Flush()

	// In a 51 cell container (after margins/padding), with 13 cell items + 2 cell gaps:
	// Line 1: Item 1 (13) + Gap (2) + Item 2 (13) = 28.
	// Item 3 (13) + Gap (2) = 15. 28 + 15 = 43. Still fits.
	// Item 4 (13) + Gap (2) = 15. 43 + 15 = 58. Too big.
	// So 3 items per line.
	// Remaining space = 51 - (1+1 padding) - (13*3) - (2*2 gaps) = 51 - 2 - 39 - 4 = 6 cells.
	// Grow factor is 1 for each, so each gets +2 cells.
	// Final width: 13 + 2 = 15.

	item1 := env.GetNodeByID("item-1")
	ro1 := env.RenderObject(item1)
	if ro1.Fragment().Size.Width != 15 {
		t.Errorf("expected width 15, got %d", ro1.Fragment().Size.Width)
	}

	// Verify that the last item also grew correctly (checking for rounding leaks)
	item3 := env.GetNodeByID("item-3")
	ro3 := env.RenderObject(item3)
	if ro3.Fragment().Size.Width != 15 {
		t.Errorf("expected item-3 width 15, got %d", ro3.Fragment().Size.Width)
	}
}
