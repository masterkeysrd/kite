package kitex

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// BenchmarkReconcilerMount measures the cost of mounting a medium-sized VDOM tree.
func BenchmarkReconcilerMount(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()

	for b.Loop() {
		container := Box(BoxProps{}).Instantiate(doc).(dom.Element)
		root := Box(BoxProps{ID: "app"},
			Span(SpanProps{ID: "s1"}, Text("text 1")),
			Span(SpanProps{ID: "s2"}, Text("text 2")),
			Span(SpanProps{ID: "s3"}, Text("text 3")),
			Span(SpanProps{ID: "s4"}, Text("text 4")),
			Span(SpanProps{ID: "s5"}, Text("text 5")),
		)
		Render(root, container)
		Render(nil, container)
	}
}

// BenchmarkReconcilerNonKeyedUpdate measures list update performance when NO keys are used.
func BenchmarkReconcilerNonKeyedUpdate(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc).(dom.Element)

	// Pre-create some lists
	listA := make([]Node, 100)
	for i := range 100 {
		listA[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, listA...)

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		listB[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, listB...)

	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}

// BenchmarkReconcilerKeyedUpdate measures list update and reordering performance when keys ARE used.
func BenchmarkReconcilerKeyedUpdate(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc).(dom.Element)

	listA := make([]Node, 100)
	for i := range 100 {
		key := fmt.Sprintf("key-%d", i)
		listA[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, listA...)

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		key := fmt.Sprintf("key-%d", idx)
		listB[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, listB...)

	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}
