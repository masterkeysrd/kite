package styler_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// helpers for benchmark trees
// ---------------------------------------------------------------------------

func BenchmarkResolver_WarmCache_ZeroAlloc(b *testing.B) {
	r := styler.NewResolver()
	node := &fakeNode{kind: dom.KindElement}
	ro := render.NewBox(node, nil)
	ro.ClearDirty(render.DirtyStyle)

	parent := &style.Computed{}

	// Warm the cache
	_ = r.Resolve(ro, parent)

	b.ReportAllocs()
	for b.Loop() {
		// Demonstrates zero-alloc cache hits: same node, same parent pointer, not dirty
		_ = r.Resolve(ro, parent)
	}
}

func BenchmarkResolver_FullResolve(b *testing.B) {
	r := styler.NewResolver()

	// 3-5 forced properties modelling a replaced element (e.g. <input>).
	intrinsic := style.Style{
		Display:    style.Some(style.DisplayInlineBlock),
		OverflowX:  style.Some(style.OverflowClip),
		WhiteSpace: style.Some(style.WhiteSpacePre),
	}

	// build a 1000-node linear tree
	nodes := make([]*fakeNode, 1000)
	ros := make([]render.Object, 1000)
	for i := range 1000 {
		nodes[i] = &fakeNode{
			kind:           dom.KindElement,
			intrinsicStyle: intrinsic,
		}
		ros[i] = render.NewBox(nodes[i], nil)
		if i > 0 {
			ros[i-1].InsertChild(ros[i], nil)
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		// Force re-resolve by marking all nodes dirty
		for _, nd := range ros {
			nd.MarkDirty(render.DirtyStyle)
		}
		r.ResolveTree(ros[0], nil, false)
	}
}

func BenchmarkResolver_FullResolve_RealElements(b *testing.B) {
	r := styler.NewResolver()
	doc := dom.NewDocument()

	// build a 1000-node linear tree of real elements
	nodes := make([]dom.Element, 1000)
	ros := make([]render.Object, 1000)
	for i := range 1000 {
		nodes[i] = dom.NewElement(doc, "div", nil)
		ros[i] = render.NewBox(nodes[i], nil)
		if i > 0 {
			ros[i-1].InsertChild(ros[i], nil)
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		for _, nd := range ros {
			nd.MarkDirty(render.DirtyStyle)
		}
		r.ResolveTree(ros[0], nil, false)
	}
}

func BenchmarkResolver_LargeTree_PartialDirty(b *testing.B) {
	r := styler.NewResolver()

	// Build a 10-way tree with depth 3 (1+10+100+1000 = 1111 nodes)
	root := &fakeNode{kind: dom.KindElement}
	rootRO := render.NewBox(root, nil)
	buildTree(rootRO, 3, 10)

	// Warm resolve
	r.ResolveTree(rootRO, nil, false)

	b.ReportAllocs()
	for b.Loop() {
		// Dirty only one leaf
		leaf := findLeaf(rootRO)
		leaf.MarkDirty(render.DirtyStyle)

		r.ResolveTree(rootRO, nil, false)
	}
}

func buildTree(parent render.Object, depth, width int) {
	if depth == 0 {
		return
	}
	for range width {
		child := &fakeNode{kind: dom.KindElement}
		childRO := render.NewBox(child, nil)
		parent.InsertChild(childRO, nil)
		buildTree(childRO, depth-1, width)
	}
}

func findLeaf(root render.Object) render.Object {
	curr := root
	for curr.FirstChild() != nil {
		curr = curr.FirstChild()
	}
	return curr
}

func BenchmarkResolver_MapVsSwitch(b *testing.B) {
	// Baseline check for map lookup overhead in the cache
	r := styler.NewResolver()
	node := &fakeNode{kind: dom.KindElement}
	ro := render.NewBox(node, nil)
	ro.ClearDirty(render.DirtyStyle)
	parent := &style.Computed{}
	_ = r.Resolve(ro, parent)

	b.Run("MapLookup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = r.Resolve(ro, parent)
		}
	})
}
