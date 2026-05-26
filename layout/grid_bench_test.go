package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func BenchmarkGridLayout_10x10(b *testing.B) {
	// Create a 10x10 grid with 100 items
	var items []Node
	for range 100 {
		items = append(items, &mockNode{
			style: &style.Computed{
				Display: style.DisplayBlock,
				Width:   style.Percent(100),
				Height:  style.Percent(100),
			},
		})
	}

	// Link siblings
	for i := 0; i < len(items)-1; i++ {
		items[i].(*mockNode).nextSibling = items[i+1]
	}

	parent := &mockNode{
		style: &style.Computed{
			Display:             style.DisplayGrid,
			GridTemplateColumns: style.Repeat(10, style.Fr(1)),
			Width:               style.Cells(100),
			Height:              style.Auto,
		},
		firstChild: items[0],
	}

	space := NewConstraintSpaceBuilder(Size{Width: 100, Height: 100}).ToConstraintSpace()
	ctx := &Context{}

	b.ResetTimer()
	for b.Loop() {
		// Clear cache manually to benchmark pure layout time
		for _, item := range items {
			mn := item.(*mockNode)
			mn.cachedFragment = nil
		}
		parent.cachedFragment = nil

		algo := &GridAlgorithm{Node: parent, Space: space}
		_ = algo.Layout(ctx)
	}
}

func BenchmarkGridLayout_Complex(b *testing.B) {
	// Create a complex grid with spanning items
	var items []Node
	for i := range 20 {
		item := &mockNode{
			style: &style.Computed{
				Display: style.DisplayBlock,
				Width:   style.Percent(100),
				Height:  style.Percent(100),
			},
		}
		if i%5 == 0 {
			item.style.GridColumn = style.GridPlacement{Span: 2}
		}
		items = append(items, item)
	}

	// Link siblings
	for i := 0; i < len(items)-1; i++ {
		items[i].(*mockNode).nextSibling = items[i+1]
	}

	parent := &mockNode{
		style: &style.Computed{
			Display:             style.DisplayGrid,
			GridTemplateColumns: style.Repeat(4, style.Fr(1)),
			GridColumnGap:       1,
			GridRowGap:          1,
			Width:               style.Cells(100),
			Height:              style.Auto,
		},
		firstChild: items[0],
	}

	space := NewConstraintSpaceBuilder(Size{Width: 100, Height: 100}).ToConstraintSpace()
	ctx := &Context{}

	b.ResetTimer()
	for b.Loop() {
		for _, item := range items {
			mn := item.(*mockNode)
			mn.cachedFragment = nil
		}
		parent.cachedFragment = nil

		algo := &GridAlgorithm{Node: parent, Space: space}
		_ = algo.Layout(ctx)
	}
}
