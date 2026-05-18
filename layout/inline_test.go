package layout

import (
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

type mockTextNode struct {
	mockNode
	data string
}

func (m *mockTextNode) Data() string     { return m.data }
func (m *mockTextNode) LogicalNode() any { return m }

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

	// Leading spaces should be collapsed at the start of the line
	if string(textFrag.Text[0].Bytes) != "L" {
		t.Errorf("expected leading spaces to be collapsed, first cluster is %q", string(textFrag.Text[0].Bytes))
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
