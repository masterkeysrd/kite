package paint

import (
	"testing"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

// buildBenchTree constructs a flat fragment tree with nChildren text fragments,
// each 1 cell wide and 1 cell tall, laid out in a single row. The root fragment
// carries the given computed style (used to control overflow).
func buildBenchTree(n int, rootStyle *style.Computed) *layout.Fragment {
	childStyle := &style.Computed{}
	children := make([]layout.FragmentLink, n)
	for i := range children {
		children[i] = layout.FragmentLink{
			Offset: layout.Point{X: i, Y: 0},
			Fragment: &layout.Fragment{
				Size: layout.Size{Width: 1, Height: 1},
				Node: &mockNode{s: childStyle},
				Text: []text.Cluster{{Bytes: []byte("x"), CellWidth: 1}},
			},
		}
	}
	return &layout.Fragment{
		Size:     layout.Size{Width: n, Height: 1},
		Node:     &mockNode{s: rootStyle},
		Children: children,
	}
}

// buildDeepBenchTree builds a linear chain of depth wrapper fragments, each
// carrying nLeaves text-cell children at the deepest level. Every intermediate
// node uses wrapStyle (typically OverflowHidden) and contains a single child.
// Total nodes = depth + nLeaves, so the tree is O(depth + nLeaves).
func buildDeepBenchTree(depth, nLeaves int, wrapStyle, leafStyle *style.Computed) *layout.Fragment {
	// Build the leaf level: nLeaves side-by-side text cells.
	leaves := make([]layout.FragmentLink, nLeaves)
	for i := range leaves {
		leaves[i] = layout.FragmentLink{
			Offset: layout.Point{X: i, Y: 0},
			Fragment: &layout.Fragment{
				Size: layout.Size{Width: 1, Height: 1},
				Node: &mockNode{s: leafStyle},
				Text: []text.Cluster{{Bytes: []byte("x"), CellWidth: 1}},
			},
		}
	}
	current := &layout.Fragment{
		Size:     layout.Size{Width: nLeaves, Height: 1},
		Node:     &mockNode{s: leafStyle},
		Children: leaves,
	}

	// Wrap current in depth wrapper fragments, each with OverflowHidden.
	for range depth {
		current = &layout.Fragment{
			Size: layout.Size{Width: nLeaves, Height: 1},
			Node: &mockNode{s: wrapStyle},
			Children: []layout.FragmentLink{
				{Offset: layout.Point{}, Fragment: current},
			},
		}
	}
	return current
}

// BenchmarkPaint_NoOverflow measures the paint overhead when every node has
// OverflowVisible (the default fast path). The new overflowClips branch must
// cost < 3 % relative to a baseline.
func BenchmarkPaint_NoOverflow(b *testing.B) {
	const nChildren = 100
	visibleStyle := &style.Computed{
		OverflowX: style.OverflowVisible,
		OverflowY: style.OverflowVisible,
	}
	frag := buildBenchTree(nChildren, visibleStyle)
	fb := NewFrameBuffer(0, 0, nChildren, 1)
	pe := &PaintEngine{}

	b.ResetTimer()
	for range b.N {
		fb.BumpVersion()
		pe.paintFragment(frag, layout.Point{}, fb)
	}
}

// BenchmarkPaint_DeepNestedClips measures the paint overhead for a 5-deep
// linear chain of Hidden-overflow wrapper nodes, each enclosing 50 leaf text
// cells. This exercises the clippedSurface allocation path under realistic
// nesting depth without exponential node growth.
func BenchmarkPaint_DeepNestedClips(b *testing.B) {
	const depth = 5
	const nLeaves = 50
	hiddenStyle := &style.Computed{
		OverflowX: style.OverflowHidden,
		OverflowY: style.OverflowHidden,
	}
	leafStyle := &style.Computed{}

	frag := buildDeepBenchTree(depth, nLeaves, hiddenStyle, leafStyle)
	fb := NewFrameBuffer(0, 0, nLeaves+10, depth+10)
	pe := &PaintEngine{}

	b.ResetTimer()
	for range b.N {
		fb.BumpVersion()
		pe.paintFragment(frag, layout.Point{}, fb)
	}
}
