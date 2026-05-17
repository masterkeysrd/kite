package focus_test

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/render"
)

// newBenchManager returns a Manager with a simple no-op dispatcher/resolver,
// suitable for benchmarks that don't care about event dispatch.
func newBenchManager(root *testObject) *focus.Manager {
	resolver := func(_ render.Object) *event.EventTarget { return nil }
	d := event.NewDispatcher(resolver)
	return focus.NewManager(root, d, resolver)
}

// BenchmarkManager_Next_LargeTree measures Next() over a flat tree with many
// focusable siblings, approximating a toolbar or menu bar scenario.
func BenchmarkManager_Next_LargeTree(b *testing.B) {
	const n = 256
	root := newNonFocusable()
	for range n {
		link(root, newFocusable())
	}

	m := newBenchManager(root)
	m.Next() // land on first element

	b.ResetTimer()
	for range b.N {
		m.Next()
	}
}

// BenchmarkScope_PushPop_Cost measures the combined cost of PushScope +
// PopScope, which includes focus capture and restore.
func BenchmarkScope_PushPop_Cost(b *testing.B) {
	root := newNonFocusable()
	prev := newFocusable()
	link(root, prev)

	modal := newNonFocusable()
	inside := newFocusable()
	link(root, modal)
	link(modal, inside)

	m := newBenchManager(root)
	m.Focus(prev, focus.ReasonProgrammatic)

	b.ResetTimer()
	for range b.N {
		s := &focus.Scope{Root: modal, Autofocus: inside}
		m.PushScope(s)
		m.PopScope()
	}
}
