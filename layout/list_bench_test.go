package layout_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

func BenchmarkListAlgorithm_Ordinal(b *testing.B) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("ul", nil)

	count := 100
	var nodes []layout.Node

	for i := 0; i < count; i++ {
		el := doc.CreateElement("li", nil)
		ro := render.NewBox(el, el)
		ro.SetComputedStyle(&style.Computed{
			Display:       style.DisplayListItem,
			ListStyleType: style.ListStyleDecimal,
		})
		el.SetRenderObject(ro)
		parent.AppendChild(el)
		nodes = append(nodes, ro)
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 100, Height: 1000}).ToConstraintSpace()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Only benchmark the last node, which has the longest sibling walk (100 items)
		lastNode := nodes[count-1]
		algo := &layout.ListAlgorithm{Node: lastNode, Space: space}
		_ = algo.Layout(nil)

		// Also clear cache to force re-layout if necessary,
		// though we want to benchmark the algorithm itself.
		lastNode.ClearDirtyLayout()
	}
}

func BenchmarkListAlgorithm_FullList(b *testing.B) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("ul", nil)

	count := 100
	var nodes []layout.Node

	for i := 0; i < count; i++ {
		el := doc.CreateElement("li", nil)
		ro := render.NewBox(el, el)
		ro.SetComputedStyle(&style.Computed{
			Display:       style.DisplayListItem,
			ListStyleType: style.ListStyleDecimal,
		})
		el.SetRenderObject(ro)
		parent.AppendChild(el)
		nodes = append(nodes, ro)
	}

	space := layout.NewConstraintSpaceBuilder(layout.Size{Width: 100, Height: 1000}).ToConstraintSpace()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, node := range nodes {
			algo := &layout.ListAlgorithm{Node: node, Space: space}
			_ = algo.Layout(nil)
			node.ClearDirtyLayout()
		}
	}
}
