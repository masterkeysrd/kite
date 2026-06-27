package layout

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

func BenchmarkListAlgorithm_Ordinal(b *testing.B) {
	count := 100
	var nodes []Node

	for i := 0; i < count; i++ {
		li := &mockNode{
			style: &style.Computed{
				Display:       style.DisplayListItem,
				ListStyleType: style.ListStyleDecimal,
			},
		}
		nodes = append(nodes, li)
	}

	// Link siblings to allow ordinal computation via previous siblings
	var prev *mockNode
	for _, n := range nodes {
		curr := n.(*mockNode)
		if prev != nil {
			prev.nextSibling = curr
		}
		prev = curr
	}

	space := NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 1000}).ToConstraintSpace()
	ctx := &Context{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lastNode := nodes[count-1].(*mockNode)
		lastNode.cachedFragment = nil
		algo := GetAlgorithm(lastNode)
		_ = algo.Layout(ctx, lastNode, space)
	}
}

func BenchmarkListAlgorithm_FullList(b *testing.B) {
	count := 100
	var nodes []Node

	for i := 0; i < count; i++ {
		li := &mockNode{
			style: &style.Computed{
				Display:       style.DisplayListItem,
				ListStyleType: style.ListStyleDecimal,
			},
		}
		nodes = append(nodes, li)
	}

	space := NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 1000}).ToConstraintSpace()
	ctx := &Context{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, node := range nodes {
			mn := node.(*mockNode)
			mn.cachedFragment = nil
			algo := GetAlgorithm(node)
			_ = algo.Layout(ctx, node, space)
		}
	}
}

func BenchmarkListAlgorithm_InlineChildren(b *testing.B) {
	const count = 100
	inlineStyle := &style.Computed{
		Display: style.DisplayInline,
		Width:   style.Cells(10),
		Height:  style.Cells(1),
	}

	var nodes []Node
	for range count {
		// Each list item has one inline child
		child := &mockInlineNode{mockNode: mockNode{style: inlineStyle}}
		li := &mockNode{
			style: &style.Computed{
				Display:       style.DisplayListItem,
				ListStyleType: style.ListStyleDecimal,
			},
			firstChild: child,
		}
		nodes = append(nodes, li)
	}

	space := NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 1000}).ToConstraintSpace()
	ctx := &Context{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, node := range nodes {
			mn := node.(*mockNode)
			mn.cachedFragment = nil
			algo := GetAlgorithm(node)
			_ = algo.Layout(ctx, node, space)
		}
	}
}
