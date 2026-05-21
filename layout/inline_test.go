package layout

import (
	"image/color"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestInlineLayout_BasicText(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInline,
			},
		},
		data: "Hello World",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have one child (the LineBox fragment)
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	lineBoxFrag := frag.Children[0].Fragment
	if lineBoxFrag.Size.Width != 11 { // "Hello World" length
		t.Errorf("expected LineBox width 11, got %d", lineBoxFrag.Size.Width)
	}

	// LineBox should have one child (the text fragment)
	if len(lineBoxFrag.Children) != 1 {
		t.Fatalf("expected 1 child in LineBox, got %d", len(lineBoxFrag.Children))
	}

	textFrag := lineBoxFrag.Children[0].Fragment
	if string(textFrag.Text[0].Bytes) != "H" {
		t.Errorf("expected first cluster to be 'H', got %s", string(textFrag.Text[0].Bytes))
	}
}

func TestInlineLayout_Wrapping(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInline,
			},
		},
		data: "Hello World",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(8), // Force wrap between "Hello" and "World"
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{8, 10}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have two LineBox fragments
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(frag.Children))
	}

	if frag.Children[0].Fragment.Size.Width != 6 { // "Hello "
		t.Errorf("expected first line width 6, got %d", frag.Children[0].Fragment.Size.Width)
	}
	if frag.Children[1].Fragment.Size.Width != 5 { // "World"
		t.Errorf("expected second line width 5, got %d", frag.Children[1].Fragment.Size.Width)
	}
}

func TestInlineLayout_AtomicInline(t *testing.T) {
	atomic := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(5),
			Height:  style.Cells(2),
		},
	}
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInline,
			},
			nextSibling: atomic,
		},
		data: "Hey",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have one LineBox fragment
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	lineBoxFrag := frag.Children[0].Fragment
	// "Hey" (3) + Atomic (5) = 8
	if lineBoxFrag.Size.Width != 8 {
		t.Errorf("expected LineBox width 8, got %d", lineBoxFrag.Size.Width)
	}
	if lineBoxFrag.Size.Height != 2 { // Height of atomic inline
		t.Errorf("expected LineBox height 2, got %d", lineBoxFrag.Size.Height)
	}
}

func TestInlineLayout_InlineFlexAtomic(t *testing.T) {
	// A flex container as an atomic inline
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(5),
			Height: style.Cells(1),
		},
	}
	flex := &mockNode{
		style: &style.Computed{
			Display:       style.DisplayInlineFlex,
			FlexDirection: style.FlexRow,
			Width:         style.Cells(10),
			Height:        style.Cells(2),
		},
		firstChild: child,
	}
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInline,
			},
			nextSibling: flex,
		},
		data: "Hey",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have one LineBox fragment
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	lineBoxFrag := frag.Children[0].Fragment
	// "Hey" (3) + Flex (10) = 13
	if lineBoxFrag.Size.Width != 13 {
		t.Errorf("expected LineBox width 13, got %d", lineBoxFrag.Size.Width)
	}
	if lineBoxFrag.Size.Height != 2 {
		t.Errorf("expected LineBox height 2, got %d", lineBoxFrag.Size.Height)
	}

	// Verify the flex container actually laid out its child.
	flexLink := lineBoxFrag.Children[1]
	if flexLink.Fragment.Size.Width != 10 {
		t.Errorf("expected flex container width 10, got %d", flexLink.Fragment.Size.Width)
	}
	if len(flexLink.Fragment.Children) != 1 {
		t.Fatalf("expected 1 child in flex container, got %d", len(flexLink.Fragment.Children))
	}
}

func TestInlineLayout_NoWrap(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display:    style.DisplayInline,
				WhiteSpace: style.WhiteSpaceNoWrap,
			},
		},
		data: "Hello World",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(8),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{8, 10}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have one LineBox fragment (even though it overflows)
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(frag.Children))
	}

	if frag.Children[0].Fragment.Size.Width != 11 {
		t.Errorf("expected LineBox width 11, got %d", frag.Children[0].Fragment.Size.Width)
	}
}

func TestInlineLayout_Alignment(t *testing.T) {
	// 1. Horizontal Alignment (Center)
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Hello",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display:   style.DisplayBlock,
			Width:     style.Cells(10),
			TextAlign: style.TextAlignCenter,
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{10, 1}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	lineBox := frag.Children[0].Fragment
	textLink := lineBox.Children[0]
	// "Hello" is 5 cells. Container is 10. Center offset = (10-5)/2 = 2.
	if textLink.Offset.X != 2 {
		t.Errorf("expected horizontal offset 2 for centered text, got %d", textLink.Offset.X)
	}

	// 2. Vertical Alignment (Bottom)
	atomic := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(2),
			Height:  style.Cells(3),
		},
	}
	textNode2 := &mockTextNode{
		mockNode: mockNode{
			style:       &style.Computed{Display: style.DisplayInline},
			nextSibling: atomic,
		},
		data: "Hi",
	}

	parent2 := &mockNode{
		style: &style.Computed{
			Display:    style.DisplayBlock,
			Width:      style.Cells(10),
			AlignItems: style.AlignEnd, // Bottom align
		},
		firstChild: textNode2,
	}

	algo2 := &BlockAlgorithm{Node: parent2, Space: space}
	frag2 := algo2.Layout()

	lineBox2 := frag2.Children[0].Fragment
	if lineBox2.Size.Height != 3 {
		t.Errorf("expected line height 3, got %d", lineBox2.Size.Height)
	}

	textLink2 := lineBox2.Children[0]
	// Text height is 1. Line height is 3. Bottom align offset = 3-1 = 2.
	if textLink2.Offset.Y != 2 {
		t.Errorf("expected vertical offset 2 for bottom-aligned text, got %d", textLink2.Offset.Y)
	}
}

func TestInlineLayout_SpaceCollapsing(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "   Leading spaces",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 1}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	lineBox := frag.Children[0].Fragment
	textFrag := lineBox.Children[0].Fragment

	// Leading spaces should be visually collapsed (CellWidth=0) but their
	// bytes must remain in the fragment for cursor byte-offset tracking.
	for i, c := range textFrag.Text {
		if len(c.Bytes) == 1 && c.Bytes[0] == ' ' {
			if c.CellWidth != 0 {
				t.Errorf("leading space cluster %d: CellWidth=%d, want 0 (collapsed)", i, c.CellWidth)
			}
		} else {
			// First non-space cluster must be 'L'.
			if string(c.Bytes) != "L" {
				t.Errorf("first non-space cluster: got %q, want \"L\"", string(c.Bytes))
			}
			break
		}
	}
}

func BenchmarkInlineLayout(b *testing.B) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(50),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{50, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_Wrapping(b *testing.B) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20), // Force wrapping
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_Atomic(b *testing.B) {
	atomic := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(5),
			Height:  style.Cells(2),
		},
	}
	textNode := &mockTextNode{
		mockNode: mockNode{
			style:       &style.Computed{Display: style.DisplayInline},
			nextSibling: atomic,
		},
		data: "Hey",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_SpaceCollapsing(b *testing.B) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "   Leading spaces",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 1}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_Alignment(b *testing.B) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Hello",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display:   style.DisplayBlock,
			Width:     style.Cells(10),
			TextAlign: style.TextAlignCenter,
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{10, 1}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayoutComplex(b *testing.B) {
	// Simulate a more complex scenario with multiple inline elements and wrapping
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	atomic1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(5),
			Height:  style.Cells(2),
		},
	}
	atomic2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(3),
			Height:  style.Cells(1),
		},
	}

	textNode.nextSibling = atomic1
	atomic1.nextSibling = atomic2

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20), // Force wrapping
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_Caching(b *testing.B) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).SetIsFixedInlineSize(true).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineBlockLayout(b *testing.B) {
	atomic := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(5),
			Height:  style.Cells(2),
		},
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: atomic,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkMixedInlineBlockLayout(b *testing.B) {
	atomic := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(5),
			Height:  style.Cells(2),
		},
	}
	textNode := &mockTextNode{
		mockNode: mockNode{
			style:       &style.Computed{Display: style.DisplayInline},
			nextSibling: atomic,
		},
		data: "Hey",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_DeeplyNested(b *testing.B) {
	// Create a deeply nested structure of inline elements
	depth := 10
	var current *mockNode
	for range depth {
		node := &mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		}
		if current != nil {
			current.firstChild = node
		}
		current = node
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: current, // Top-level inline node
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}

func BenchmarkInlineLayout_LargeText(b *testing.B) {
	// Simulate a large text node to test performance of inline layout with many clusters
	var largeText strings.Builder
	for range 1000 {
		largeText.WriteString("Lorem ipsum ")
	}

	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: largeText.String(),
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(50),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{50, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	for b.Loop() {
		algo.Layout()
	}
}
func TestInlineLayout_VerticalAlignment(t *testing.T) {
	// Container (Block) with AlignItems: Stretch (default)
	// Contains: Text (height 1) and InlineFlex (height 3)
	// Surrounding text should ideally be centered relative to the tall InlineFlex item if the line height is 3.

	itemStyle := style.DefaultStyle()
	itemStyle.Display = style.DisplayInlineFlex
	itemStyle.Width = style.Cells(10)
	itemStyle.Height = style.Cells(3)

	inlineFlex := &mockNode{style: &itemStyle}

	textStyle := style.DefaultStyle()
	textStyle.Display = style.DisplayInline
	textNode1 := &mockTextNode{
		mockNode: mockNode{style: &textStyle},
		data:     "Before",
	}
	textNode1.nextSibling = inlineFlex

	textNode2 := &mockTextNode{
		mockNode: mockNode{style: &textStyle},
		data:     "After",
	}
	inlineFlex.nextSibling = textNode2

	parentStyle := style.DefaultStyle()
	parentStyle.Display = style.DisplayBlock
	parentStyle.AlignItems = style.AlignStretch
	parentStyle.Width = style.Cells(100)

	parent := &mockNode{
		style:      &parentStyle,
		firstChild: textNode1,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Line height should be 3
	if len(frag.Children) == 0 {
		t.Fatalf("expected at least one line child")
	}
	lineFrag := frag.Children[0].Fragment
	if lineFrag.Size.Height != 3 {
		t.Errorf("expected line height 3, got %d", lineFrag.Size.Height)
	}

	// In lineFrag.Children:
	// 0: "Before" (text)
	// 1: inlineFlex (atomic)
	// 2: "After" (text)

	if len(lineFrag.Children) != 3 {
		t.Fatalf("expected 3 items on line, got %d", len(lineFrag.Children))
	}

	// Check "Before" text vertical offset.
	// If it's AlignStart (defaulting from Stretch), offset.Y will be 0.
	// If we want it centered, it should be (3 - 1) / 2 = 1.
	beforeTextOffset := lineFrag.Children[0].Offset.Y
	if beforeTextOffset != 1 {
		t.Errorf("expected 'Before' text Y offset 1 (centered), got %d", beforeTextOffset)
	}
}

func BenchmarkComplexInlineLayout_100k(b *testing.B) {
	const count = 100000

	styles := []*style.Computed{
		{Display: style.DisplayInline, Foreground: color.RGBA{R: 255, A: 255}},
		{Display: style.DisplayInline, Bold: true},
		{Display: style.DisplayInline, Italic: true},
		{Display: style.DisplayInlineBlock, Width: style.Cells(2), Height: style.Cells(1), Background: color.RGBA{G: 255, A: 255}},
	}

	var firstChild Node
	var prev *mockNode

	for i := range count {
		s := styles[i%len(styles)]
		var curr *mockNode
		if s.Display == style.DisplayInline {
			mtn := &mockTextNode{
				mockNode: mockNode{style: s},
				data:     "text ",
			}
			curr = &mtn.mockNode
		} else {
			curr = &mockNode{style: s}
		}

		if firstChild == nil {
			firstChild = curr
		} else {
			prev.nextSibling = curr
		}
		prev = curr
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(100),
		},
		firstChild: firstChild,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100000}).ToConstraintSpace()

	for b.Loop() {
		// Clear cache to force full layout pass
		parent.cachedFragment = nil
		algo := &BlockAlgorithm{Node: parent, Space: space}
		algo.Layout()
	}
}

func BenchmarkComplexInlineLayout_Nested_100k(b *testing.B) {
	const count = 100000

	// Each iteration adds 5 nodes (1 outer, 1 inner, 3 text)
	const nodesPerIter = 5
	const iterations = count / nodesPerIter

	styles := []*style.Computed{
		{Display: style.DisplayInline, Foreground: color.RGBA{R: 255, A: 255}},
		{Display: style.DisplayInline, Bold: true},
		{Display: style.DisplayInline, Italic: true},
	}

	var firstChild Node
	var prev *mockNode

	for range iterations {
		// Outer span
		outer := &mockNode{style: styles[0]}

		// Child 1: text
		t1 := &mockTextNode{
			mockNode: mockNode{style: styles[1]},
			data:     "outer ",
		}
		outer.firstChild = &t1.mockNode

		// Child 2: inner span
		inner := &mockNode{style: styles[2]}
		t1.nextSibling = inner

		// Child 3: inner text
		t2 := &mockTextNode{
			mockNode: mockNode{style: styles[0]},
			data:     "inner ",
		}
		inner.firstChild = &t2.mockNode

		// Child 4: outer trailing text
		t3 := &mockTextNode{
			mockNode: mockNode{style: styles[1]},
			data:     "trailing ",
		}
		inner.nextSibling = &t3.mockNode

		if firstChild == nil {
			firstChild = outer
		} else {
			prev.nextSibling = outer
		}
		prev = outer
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(100),
		},
		firstChild: firstChild,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100000}).ToConstraintSpace()

	for b.Loop() {
		parent.cachedFragment = nil
		algo := &BlockAlgorithm{Node: parent, Space: space}
		algo.Layout()
	}
}

func BenchmarkComplexInlineLayout_DistinctStyles_100k(b *testing.B) {
	const count = 100000

	var firstChild Node
	var prev *mockNode

	for i := range count {
		// Create a distinct style for every node
		s := &style.Computed{
			Display:    style.DisplayInline,
			Foreground: color.RGBA{R: uint8(i % 256), G: uint8((i / 256) % 256), B: uint8((i / 65536) % 256), A: 255},
			Bold:       i%2 == 0,
			Italic:     i%3 == 0,
		}

		mtn := &mockTextNode{
			mockNode: mockNode{style: s},
			data:     "text ",
		}
		curr := &mtn.mockNode

		if firstChild == nil {
			firstChild = curr
		} else {
			prev.nextSibling = curr
		}
		prev = curr
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(100),
		},
		firstChild: firstChild,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100000}).ToConstraintSpace()

	for b.Loop() {
		parent.cachedFragment = nil
		algo := &BlockAlgorithm{Node: parent, Space: space}
		algo.Layout()
	}
}

func TestInlineLayout_MandatoryBreak(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display:    style.DisplayInline,
				WhiteSpace: style.WhiteSpacePreWrap,
			},
		},
		data: "line1\nline2",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have exactly two LineBox fragments
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(frag.Children))
	}

	if frag.Children[0].Fragment.Size.Width != 5 { // "line1" (plus newline consumed but 0 width)
		t.Errorf("expected first line width 5, got %d", frag.Children[0].Fragment.Size.Width)
	}
	if frag.Children[1].Fragment.Size.Width != 5 { // "line2"
		t.Errorf("expected second line width 5, got %d", frag.Children[1].Fragment.Size.Width)
	}
}

func TestInlineLayout_TrailingNewline(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display:    style.DisplayInline,
				WhiteSpace: style.WhiteSpacePreWrap,
			},
		},
		data: "line1\n",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have exactly two LineBox fragments (one for "line1\n", one empty for after \n)
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(frag.Children))
	}

	if frag.Children[0].Fragment.Size.Width != 5 {
		t.Errorf("expected first line width 5, got %d", frag.Children[0].Fragment.Size.Width)
	}
	if frag.Children[1].Fragment.Size.Width != 0 {
		t.Errorf("expected second line width 0, got %d", frag.Children[1].Fragment.Size.Width)
	}
}

func TestInlineLayout_EmergencyBreak(t *testing.T) {
	textNode := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display:    style.DisplayInline,
				WhiteSpace: style.WhiteSpacePreWrap,
			},
		},
		data: "123456789012345",
	}

	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(10),
		},
		firstChild: textNode,
	}

	space := NewConstraintSpaceBuilder(Size{10, 10}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Should have two LineBox fragments
	// Line 0: "1234567890" (10 chars)
	// Line 1: "12345" (5 chars)
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(frag.Children))
	}

	if frag.Children[0].Fragment.Size.Width != 10 {
		t.Errorf("expected first line width 10, got %d", frag.Children[0].Fragment.Size.Width)
	}
	if frag.Children[1].Fragment.Size.Width != 5 {
		t.Errorf("expected second line width 5, got %d", frag.Children[1].Fragment.Size.Width)
	}
}
