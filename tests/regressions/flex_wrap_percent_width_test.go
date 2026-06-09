package regressions

// TestFlexWrapOn_PercentWidthWithTextItems verifies that a flex container with
// FlexWrapOn and Width(Percent(100)) correctly wraps inline text children when
// the combined text width exceeds the parent's fixed width constraint.
//
// Regression: when multiple text nodes are flex children, the IFC assembles them
// into a single AnonymousBlock. Each text node is shaped independently, so the
// cross-item word boundary (space at end of one text, non-space at start of the
// next) loses its BreakSoft classification. The line breaker then has no break
// opportunity and places all text on one overflowing line.
//
// Layout tree (viewport 80, outer box 40 wide with 1-cell border):
//
//	outer  – Width(Cells(40)), Border(SingleBorder)  → content-box = 38
//	  flex  – DisplayFlex, FlexRow, FlexWrapOn, Width(Percent(100))
//	    text1 – "A_Very_Long_Tool_Name_1 "  (24 cells, trailing space)
//	    text2 – "A_Very_Long_Tool_Name_2 "  (24 cells)
//	    text3 – "A_Very_Long_Tool_Name_3 "  (24 cells)
//
// Total text = 72 cells. Available = 38. Line-breaking MUST split text across lines.
//
// Expected: outer right border at x=39 contains "┐" (box stays ≤ 40 wide).
// Buggy:    all 72 cells go on one line; right border pushed past x=39.

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestFlexWrapOn_PercentWidthWithTextItems(t *testing.T) {
	// Viewport wider than the outer box so overflow would be visible.
	env := testenv.Default(80, 10)
	defer env.Close()

	// Three text nodes that exceed the 38-cell content area when combined.
	// Each ends with a space, creating a word-boundary break opportunity
	// across nodes — but only if cross-item BreakSoft is propagated correctly.
	flex := element.Box(
		element.Text("A_Very_Long_Tool_Name_1 "),
		element.Text("A_Very_Long_Tool_Name_2 "),
		element.Text("A_Very_Long_Tool_Name_3 "),
	).Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Width(style.Percent(100)),
	).WithID("flex")

	outer := element.Box(flex).Style(style.S().
		Width(style.Cells(40)).
		Border(style.SingleBorder()),
	).WithID("outer")

	env.Mount(element.Box(outer))
	env.Flush()

	outerNode := env.GetNodeByID("outer")
	outerRO := env.RenderObject(outerNode)

	// The outer box must remain 40 cells wide.
	// Buggy behaviour: outer grew to accommodate the 72-cell text run.
	if outerRO.Fragment().Size.Width != 40 {
		t.Errorf("outer box width = %d, want 40; outer grew due to text overflow", outerRO.Fragment().Size.Width)
	}

	// The outer box right-border character (top-right corner) must be at x=39.
	// If the outer grew, this assertion is irrelevant, but the width check above
	// already catches that.
	testenv.ExpectScreen(t, env).CellAt(39, 0).ToHaveContent("┐")

	// The flex container must NOT overflow the outer content area.
	// If text overflows, the outer box would visually extend past x=39.
	// Verify that the outer left border is still at x=0.
	testenv.ExpectScreen(t, env).CellAt(0, 0).ToHaveContent("┌")

	// More importantly: verify that text DID wrap. The flex container height
	// must be > 1 row (text cannot all fit on one line of 38 cells).
	flexNode := env.GetNodeByID("flex")
	flexRO := env.RenderObject(flexNode)
	if flexRO.Fragment().Size.Height <= 1 {
		t.Errorf("flex container height = %d; expected > 1 (text must wrap across lines)", flexRO.Fragment().Size.Height)
	}
}

// TestFlexWrapOn_PercentWidthWithBlockItems verifies the same behaviour for
// block-level flex items (element.Box children), confirming the existing fix
// still works and providing a baseline for the text-item regression above.
func TestFlexWrapOn_PercentWidthWithBlockItems(t *testing.T) {
	env := testenv.Default(80, 10)
	defer env.Close()

	// Each item is 14 cells wide. Three items total = 42 > 38-cell content area.
	itemStyle := style.S().Width(style.Cells(14)).Height(style.Cells(1))

	flex := element.Box(
		element.Box().Style(itemStyle).WithID("item-1"),
		element.Box().Style(itemStyle).WithID("item-2"),
		element.Box().Style(itemStyle).WithID("item-3"),
	).Style(style.S().
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow).
		FlexWrap(style.FlexWrapOn).
		Width(style.Percent(100)),
	).WithID("flex-block")

	outer := element.Box(flex).Style(style.S().
		Width(style.Cells(40)).
		Border(style.SingleBorder()),
	).WithID("outer-block")

	env.Mount(element.Box(outer))
	env.Flush()

	outerNode := env.GetNodeByID("outer-block")
	outerRO := env.RenderObject(outerNode)

	if outerRO.Fragment().Size.Width != 40 {
		t.Errorf("outer box width = %d, want 40", outerRO.Fragment().Size.Width)
	}

	flexNode := env.GetNodeByID("flex-block")
	flexRO := env.RenderObject(flexNode)

	// item-3 must wrap: the flex container should have height > 1.
	if flexRO.Fragment().Size.Height <= 1 {
		t.Errorf("flex container height = %d; expected > 1 (item-3 must wrap)", flexRO.Fragment().Size.Height)
	}
}
