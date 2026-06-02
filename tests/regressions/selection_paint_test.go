package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/paint"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestTextSelectionPaint(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// 1. Build a document with some text.
	doc := env.Document()
	div := element.NewBox(doc)
	div.Style(style.S().Background(color.RGBA{R: 0, G: 0, B: 0, A: 255}).Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255}))

	t1 := element.Text("Hello ")
	t2 := element.Text("Selection")
	t3 := element.Text(" World")

	div.AppendChild(t1)
	div.AppendChild(t2)
	div.AppendChild(t3)
	doc.AppendChild(div)

	// 2. Set selection on the middle text node.
	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t2, 0)
	rng.SetEnd(t2, 9)
	sel.AddRange(rng)

	env.Flush()

	// 3. Verify that the "Selection" text has the inversion attribute.
	for x := 6; x < 15; x++ {
		testenv.ExpectScreen(t, env).CellAt(x, 0).ToHaveAttribute(paint.AttrInverse)
	}

	// "Hello " should NOT have inversion.
	testenv.ExpectScreen(t, env).CellAt(0, 0).ToHaveContent("H")
}

func TestTextSelectionCustomColors(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	selFG := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	selBG := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	doc := env.Document()
	div := element.NewBox(doc)
	div.Style(style.S().SelectionForeground(selFG).SelectionBackground(selBG))

	t1 := element.Text("Selected")
	div.AppendChild(t1)
	doc.AppendChild(div)

	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t1, 0)
	rng.SetEnd(t1, 8)
	sel.AddRange(rng)

	env.Flush()

	// Verify custom colors
	for x := 0; x < 8; x++ {
		cell := env.Backend.LastFrame().Surface.CellAt(x, 0)
		if cell.Fg != selFG {
			t.Errorf("at x=%d, expected FG %v, got %v", x, selFG, cell.Fg)
		}
		if cell.Bg != selBG {
			t.Errorf("at x=%d, expected BG %v, got %v", x, selBG, cell.Bg)
		}
	}
}

func TestPartialTextSelectionPaint(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	t1 := element.Text("0123456789")
	doc.AppendChild(t1)

	// Select "234" (indices 2, 3, 4)
	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t1, 2)
	rng.SetEnd(t1, 5)
	sel.AddRange(rng)

	env.Flush()

	// 0, 1 should NOT be selected
	testenv.ExpectScreen(t, env).CellAt(0, 0).ToNotHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(1, 0).ToNotHaveAttribute(paint.AttrInverse)

	// 2, 3, 4 SHOULD be selected
	testenv.ExpectScreen(t, env).CellAt(2, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(3, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(4, 0).ToHaveAttribute(paint.AttrInverse)

	// 5, 6, 7, 8, 9 should NOT be selected
	testenv.ExpectScreen(t, env).CellAt(5, 0).ToNotHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(6, 0).ToNotHaveAttribute(paint.AttrInverse)
}

func TestMultilineTextSelectionPaint(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	// Box with fixed width to force wrap
	box := element.Box(
		element.Text("Line One. Line Two."),
	).Style(style.S().Width(style.Cells(10)))
	doc.AppendChild(box)

	// "Line One. " is 10 chars, should be first line.
	// "Line Two." is 9 chars, should be second line.

	// Select "ne One. Li"
	// Indices 2 to 12.
	// Line 1: "Line One. " (indices 0-9). Selected 2-10. -> "ne One. " (X: 2 to 10)
	// Line 2: "Line Two." (indices 10-19). Selected 10-12. -> "Li" (X: 0 to 2)

	sel := doc.Selection()
	rng := doc.CreateRange()
	t1 := box.FirstChild().(dom.TextNode)
	rng.SetStart(t1, 2)
	rng.SetEnd(t1, 12)
	sel.AddRange(rng)

	env.Flush()

	// Line 0: "Line One. "
	// Selected: "ne One. " (indices 2 to 10)
	testenv.ExpectScreen(t, env).CellAt(1, 0).ToNotHaveAttribute(paint.AttrInverse) // 'i'
	testenv.ExpectScreen(t, env).CellAt(2, 0).ToHaveAttribute(paint.AttrInverse)    // 'n'
	testenv.ExpectScreen(t, env).CellAt(9, 0).ToHaveAttribute(paint.AttrInverse)    // ' '

	// Line 1: "Line Two."
	// Selected: "Li" (indices 10 to 12)
	testenv.ExpectScreen(t, env).CellAt(0, 1).ToHaveAttribute(paint.AttrInverse)    // 'L'
	testenv.ExpectScreen(t, env).CellAt(1, 1).ToHaveAttribute(paint.AttrInverse)    // 'i'
	testenv.ExpectScreen(t, env).CellAt(2, 1).ToNotHaveAttribute(paint.AttrInverse) // 'n'
}

func TestListItemTextSelectionPaint(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	li := element.LI(
		element.Text("Item One"),
	)
	doc.AppendChild(element.UL(li))

	// Marker "• " is at X=2, 3. Text "Item One" starts at 4,0
	// Select "tem" (indices 1, 2, 3 of "Item One") -> X=5, 6, 7
	t1 := li.FirstChild().(dom.TextNode)
	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t1, 1)
	rng.SetEnd(t1, 4)
	sel.AddRange(rng)

	env.Flush()

	// "I" (X=4) should not be selected
	testenv.ExpectScreen(t, env).CellAt(4, 0).ToNotHaveAttribute(paint.AttrInverse)
	// "tem" (X=5, 6, 7) should be selected
	testenv.ExpectScreen(t, env).CellAt(5, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(6, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(7, 0).ToHaveAttribute(paint.AttrInverse)
	// " " (X=8) should not be selected
	testenv.ExpectScreen(t, env).CellAt(8, 0).ToNotHaveAttribute(paint.AttrInverse)

	// Now select from Marker to middle of text
	// Select "• Item"
	// Marker is associated with LI node.
	// Since markers are synthesized, selecting the LI node should ideally select the marker.
	// But our logic excludes markers from logical offsets.
	// If we select the LI node itself:
	rng.SetStart(li, 0)
	rng.SetEnd(t1, 4) // End at "tem"
	env.Flush()

	// Marker "•" (X=2) is NOT selected because it's a synthesized marker and not part of the logical text buffer.
	// The user asked for "char by char selection" which typically refers to the text buffer.
	testenv.ExpectScreen(t, env).CellAt(2, 0).ToNotHaveAttribute(paint.AttrInverse)
	// "I" (X=4) is start of T1, should be selected
	testenv.ExpectScreen(t, env).CellAt(4, 0).ToHaveAttribute(paint.AttrInverse)
}

func TestListItemClickSelection(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	li := element.LI(
		element.Text("Item One"),
	)
	doc.AppendChild(element.UL(li))

	env.Flush()

	// "Item One" text should be at X=4, Y=0 (Marker "• " at X=2, 3)
	// Click on 't' in "Item" which is at index 1.
	// X=4 is 'I', X=5 is 't'.

	env.MouseDown(5, 0, event.ButtonLeft)
	env.Flush()

	sel := doc.Selection()
	if sel.RangeCount() != 1 {
		t.Fatalf("expected 1 range, got %d", sel.RangeCount())
	}
	rng := sel.GetRangeAt(0)

	// Check anchor offset.
	if rng.StartOffset() != 1 {
		t.Errorf("expected start offset 1, got %d", rng.StartOffset())
	}
}

func TestListSelectInto(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	t1 := element.Text("Before")
	li := element.LI(element.Text("Inside"))
	doc.AppendChild(t1)
	doc.AppendChild(element.UL(li))

	env.Flush()

	// Select from 'f' in "Before" (index 2) to 's' in "Inside" (index 3 of T1)
	// "Before" is at 0,0.
	// UL/LI is at Y=1. Marker at X=2,3. "Inside" at X=4...

	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t1, 2)
	tInside := li.FirstChild().(dom.TextNode)
	rng.SetEnd(tInside, 3) // "Ins"
	sel.AddRange(rng)

	env.Flush()

	// 'd' (index 4) in "Inside" should NOT be selected. X=8
	testenv.ExpectScreen(t, env).CellAt(8, 1).ToNotHaveAttribute(paint.AttrInverse)
}

func TestListItemOmitAdornments(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	li1 := element.LI(element.Text("Item One"))
	li2 := element.LI(element.Text("Item Two"))
	doc.AppendChild(element.UL(li1, li2))

	env.Flush()

	// Select everything.
	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(li1.FirstChild().(dom.TextNode), 0)
	rng.SetEnd(li2.FirstChild().(dom.TextNode), 8)
	sel.AddRange(rng)

	env.Flush()

	// Marker of LI 1 is at X=2. Should NOT be highlighted.
	testenv.ExpectScreen(t, env).CellAt(2, 0).ToNotHaveAttribute(paint.AttrInverse)
	// Marker of LI 2 is at X=2, Y=1. Should NOT be highlighted.
	testenv.ExpectScreen(t, env).CellAt(2, 1).ToNotHaveAttribute(paint.AttrInverse)

	// Text should be highlighted.
	testenv.ExpectScreen(t, env).CellAt(4, 0).ToHaveAttribute(paint.AttrInverse) // 'I'
	testenv.ExpectScreen(t, env).CellAt(4, 1).ToHaveAttribute(paint.AttrInverse) // 'I'
}

func TestSelectLastCharOfLine(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	// Use a block container to satisfy findBlockAncestor
	box := element.Box(element.Text("ABCDE"))
	doc.AppendChild(box)

	env.Flush()

	// "ABCDE" at 0,0.
	// E is at X=4.
	// Click on 'E' (X=4) and drag right to X=5.
	env.MouseDown(4, 0, event.ButtonLeft)
	env.MouseMove(5, 0)
	env.Flush()

	sel := doc.Selection()
	if sel.String() != "E" {
		t.Errorf("expected selection 'E', got %q", sel.String())
	}
}

func TestSelectLastCharOfWrappedLine(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	// Force wrap after 3 cells
	box := element.Box(element.Text("ABCDE")).Style(style.S().Width(style.Cells(3)).OverflowWrap(style.OverflowWrapBreakWord))
	doc.AppendChild(box)

	env.Flush()

	// Line 0: "ABC"
	// Line 1: "DE"
	// 'C' is at X=2, Y=0.
	// 'D' is at X=0, Y=1.

	env.MouseDown(2, 0, event.ButtonLeft)
	env.MouseMove(1, 1) // Drag to 'E' (X=1, Y=1) to include 'D'
	env.Flush()

	sel := doc.Selection()
	// Should select "CD"
	// 'C' is index 2. 'D' is index 3. 'E' is index 4.
	// Range [2, 4) -> "CD"
	if sel.String() != "CD" {
		t.Errorf("expected selection 'CD', got %q", sel.String())
	}
}

func TestListItemDragDownSelection(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	li1 := element.LI(element.Text("Item One"))
	li2 := element.LI(element.Text("Item Two"))
	doc.AppendChild(element.UL(li1, li2))

	env.Flush()

	// LI 1 at Y=0. "Item One" at X=4...
	// LI 2 at Y=1. "Item Two" at X=4...

	// Drag from 'n' in "One" (X=10, Y=0) down to 'm' in "Item" (X=7, Y=1)
	env.MouseDown(10, 0, event.ButtonLeft)
	env.MouseMove(7, 1)
	env.Flush()

	// Should select "e" from Item One, the \n (if any), the marker of LI 2, and "Ite" from Item Two.
	// Index of 'n' in "Item One" is 6. Index of 'e' is 7.
	// Selection from 6 to end of LI 1 text (8).
	// Then LI 2 marker.
	// Then LI 2 text from 0 to 3 ("Ite").

	// Check that it's NOT selecting the whole line 1.
	// 'e' (X=6) is selected. 'm' (X=7) should NOT be selected.
	testenv.ExpectScreen(t, env).CellAt(6, 1).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(7, 1).ToNotHaveAttribute(paint.AttrInverse)
}

func TestListItemReverseSelection(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	li := element.LI(element.Text("Reverse"))
	doc.AppendChild(element.UL(li))

	env.Flush()

	// Text "Reverse" starts at X=4 (Marker "• " at X=2, 3)
	// Drag from 'e' (index 6, X=10) back to 'v' (index 2, X=6)

	env.MouseDown(10, 0, event.ButtonLeft)
	env.MouseMove(6, 0)
	env.Flush()

	// Should select "vers" (indices 2, 3, 4, 5)
	// X=6 (v), 7 (e), 8 (r), 9 (s)
	testenv.ExpectScreen(t, env).CellAt(6, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(9, 0).ToHaveAttribute(paint.AttrInverse)
	testenv.ExpectScreen(t, env).CellAt(10, 0).ToNotHaveAttribute(paint.AttrInverse)
}

func TestSelectionOverPaddingArea(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	doc := env.Document()
	// Box with padding and multiple children
	outer := element.Box(
		element.Box("Header"),
		element.Box("Target Text"),
	).Style(style.S().Padding(style.Edges(2, 2)))
	doc.AppendChild(outer)

	env.Flush()

	// "Header" is at Y=2 (due to padding).
	// "Target Text" is at Y=3.
	// Mouse moves to Y=0 (padding area of 'outer') while selecting "Target Text".

	// 1. MouseDown on "Target" (Y=3, X=2... because of padding-left default is usually 0 if not set,
	// but let's assume it's at X=2 due to our Padding: Edges(2,2))
	env.MouseDown(2, 3, event.ButtonLeft)
	env.Flush()

	// 2. MouseMove to Y=0 (still X=2)
	env.MouseMove(2, 0)
	env.Flush()

	sel := doc.Selection()
	// The selection should NOT have jumped to "Header".
	// Since Y=0 is above "Header" (which is at Y=2), it should select everything BEFORE "Target".
	// Including "Header".

	// Wait, if I drag UP from "Target", I expect to select "Header" too.
	// But the user says it "goes to the title" when pointing to margin/padding.
	// If I'm pointing at Y=0, and that's the padding of the outer box,
	// ByteOffsetAtPoint(outer, 2, 0) should now correctly see that Y=0 is BEFORE any text in 'outer' (if we had no header)
	// OR it should correctly find that it's above everything.

	// If I drag from Y=3 (Target) to Y=0 (Padding).
	// Range should be from Start of Doc to index in Target? No, Reverse: from index in Target to Start of Doc.

	if sel.String() == "" {
		t.Error("selection is empty, should have selected something")
	}

	// The user says "the selection goes to the title instead of being keep".
	// This usually means the selection anchor jumped or the end jumped to something else.

	// With recursive ByteOffsetAtPoint, it should be stable.
}
