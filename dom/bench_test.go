package dom_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// BenchmarkElement_AppendChild measures the cost of appending 1k children to
// a single parent element, catching accidental O(N) sibling rewires.
func BenchmarkElement_AppendChild(b *testing.B) {
	const n = 1000
	doc := dom.NewDocument()
	children := make([]dom.Node, n)
	for i := range children {
		children[i] = doc.CreateElement("span", nil)
	}

	for b.Loop() {
		parent := doc.CreateElement("div", nil)
		for _, child := range children {
			parent.AppendChild(child)
		}
	}
}

// BenchmarkElement_RemoveChild_Middle measures the cost of removing the
// middle child from a 1k-child parent (catching accidental O(N) sibling
// rewires). The tree is built once; each iteration removes the middle node
// and re-inserts it at the same position so the next iteration starts clean.
// Remove + Insert are both O(1), so timing them together still validates the
// asymptotic cost.
func BenchmarkElement_RemoveChild_Middle(b *testing.B) {
	const n = 1000
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	for range n {
		parent.AppendChild(doc.CreateElement("span", nil))
	}

	// Collect children to find middle and its next sibling.
	children := make([]dom.Node, 0, n)
	for c := range parent.ChildNodes() {
		children = append(children, c)
	}
	middle := children[n/2]
	after := children[n/2+1] // stays in place; used to restore middle

	for b.Loop() {
		parent.RemoveChild(middle)
		parent.InsertBefore(middle, after)
	}
}
