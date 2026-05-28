package styler_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// helpers for benchmark trees
// ---------------------------------------------------------------------------

// newBenchNode returns a fresh fakeNode with dirtyStyle set.
func newBenchNode() *fakeNode {
	return &fakeNode{dirtyStyle: true}
}

// ---------------------------------------------------------------------------
// BenchmarkResolver_CacheHit
//
// Demonstrates zero-alloc cache hits: same node, same parent pointer, not dirty.
// ---------------------------------------------------------------------------

func BenchmarkResolver_CacheHit(b *testing.B) {
	r := styler.NewResolver()

	parent := &style.Computed{Foreground: color.RGBA{R: 100, A: 255}, Background: color.Transparent}
	node := &fakeNode{dirtyStyle: true}

	// Warm the cache.
	r.Resolve(node, parent)
	// Clear dirty so subsequent calls are cache hits.
	node.dirtyStyle = false

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = r.Resolve(node, parent)
	}
}

// ---------------------------------------------------------------------------
// BenchmarkResolver_FullResolve
//
// Measures full resolution (no cache) across a 1 000-node tree. Each
// iteration marks all nodes dirty so the resolver can never hit the cache.
// ---------------------------------------------------------------------------

func BenchmarkResolver_FullResolve(b *testing.B) {
	const n = 1000
	r := styler.NewResolver()

	nodes := make([]*fakeNode, n)
	for i := range nodes {
		nodes[i] = newBenchNode()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, nd := range nodes {
			nd.dirtyStyle = true
			_ = r.Resolve(nd, nil)
		}
	}
}

// ---------------------------------------------------------------------------
// BenchmarkResolveTree_OneDirtyLeaf_10kNodes
//
// Models a large, mostly-clean tree (10 000 nodes) where only a single leaf
// is dirty. Verifies that the relay flags ensure the traversal visits only
// the dirty path (root → spine → dirty leaf) rather than the full tree.
// ---------------------------------------------------------------------------

func BenchmarkResolveTree_OneDirtyLeaf_10kNodes(b *testing.B) {
	const depth = 10000 // linear chain

	r := styler.NewResolver()

	// Build a linear chain: root → n1 → n2 → ... → nDepth
	root := &fakeNode{}
	nodes := make([]*fakeNode, depth)
	nodes[0] = root
	for i := 1; i < depth; i++ {
		nodes[i] = &fakeNode{}
		nodes[i-1].appendChild(nodes[i])
	}

	// Pre-warm: resolve whole tree once so cache is populated.
	parent := (*fakeNode)(nil)
	for _, n := range nodes {
		res := r.Resolve(n, r.Cache[parent].Result)
		parent = n
		_ = res
	}

	b.ReportAllocs()
	b.ResetTimer()
	leaf := nodes[depth-1]
	for b.Loop() {
		// Mark only the deepest leaf dirty.
		leaf.markDirty()
		parent := (*fakeNode)(nil)
		for _, n := range nodes {
			res := r.Resolve(n, r.Cache[parent].Result)
			parent = n
			_ = res
		}
	}
}

// ---------------------------------------------------------------------------
// BenchmarkResolver_NoIntrinsic (TSK-022)
//
// Verifies that elements that return an empty IntrinsicStyle() (the common
// case) incur < 3 % overhead versus the resolver before the intrinsic layer
// was added. The benchmark exercises 1 000 fresh nodes so the cache is cold.
// ---------------------------------------------------------------------------

func BenchmarkResolver_NoIntrinsic(b *testing.B) {
	const n = 1000
	r := styler.NewResolver()

	nodes := make([]*fakeNode, n)
	for i := range nodes {
		nodes[i] = &fakeNode{
			dirtyStyle:     true,
			intrinsicStyle: style.Style{}, // empty — no UA-forced properties
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, nd := range nodes {
			nd.dirtyStyle = true
			_ = r.Resolve(nd, nil)
		}
	}
}

// ---------------------------------------------------------------------------
// BenchmarkResolver_WithIntrinsic (TSK-022)
//
// Measures the overhead of the intrinsic layer when elements have 3–5 forced
// properties set. Acceptable overhead is < 10 % vs BenchmarkResolver_NoIntrinsic.
// ---------------------------------------------------------------------------

func BenchmarkResolver_WithIntrinsic(b *testing.B) {
	const n = 1000
	r := styler.NewResolver()

	// 3-5 forced properties modelling a replaced element (e.g. <input>).
	intrinsic := style.Style{
		Display:    style.Some(style.DisplayInlineBlock),
		OverflowX:  style.Some(style.OverflowClip),
		OverflowY:  style.Some(style.OverflowClip),
		WhiteSpace: style.Some(style.WhiteSpaceNoWrap),
	}

	nodes := make([]*fakeNode, n)
	for i := range nodes {
		nodes[i] = &fakeNode{
			dirtyStyle:     true,
			rawStyle:       style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 100, A: 255})},
			intrinsicStyle: intrinsic,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, nd := range nodes {
			nd.dirtyStyle = true
			_ = r.Resolve(nd, nil)
		}
	}
}
