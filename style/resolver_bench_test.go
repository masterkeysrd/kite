package style_test

import (
	"image/color"
	"testing"

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
	r := style.NewResolver()

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
	r := style.NewResolver()

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

	r := style.NewResolver()

	// Build a linear chain: root → n1 → n2 → ... → nDepth
	root := &fakeNode{}
	nodes := make([]*fakeNode, depth)
	nodes[0] = root
	for i := 1; i < depth; i++ {
		nodes[i] = &fakeNode{}
		nodes[i-1].appendChild(nodes[i])
	}

	// Pre-warm: resolve whole tree once so cache is populated.
	leaf := nodes[depth-1]
	leaf.markDirty()
	style.ResolveTree(r, root)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Mark only the deepest leaf dirty.
		leaf.markDirty()
		style.ResolveTree(r, root)
	}
}
