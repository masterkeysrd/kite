package kitex

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// BenchmarkContextRead measures the cost of UseContext lookup on subsequent renders (the O(1) path).
func BenchmarkContextRead(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc).(dom.Element)

	ctx := CreateContext("value")

	Consumer := FC("Consumer", func(props struct{}) Node {
		_ = UseContext(ctx)
		return Text("Consumer")
	})

	var triggerUpdate func()
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		_, setDummy := UseState(0)
		triggerUpdate = func() { setDummy(1) }
		return ctx.Provider("value", Consumer(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)

	b.ResetTimer()
	for b.Loop() {
		triggerUpdate()
	}
}

// BenchmarkContextPropagation measures propagation cost when the provider value changes,
// triggering subscriber dirty propagation and component re-render.
func BenchmarkContextPropagation(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc).(dom.Element)

	ctx := CreateContext("value")

	Consumer := FC("Consumer", func(props struct{}) Node {
		_ = UseContext(ctx)
		return Text("Consumer")
	})

	var setProviderValue func(string)
	var state int
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("value1")
		setProviderValue = setVal
		return ctx.Provider(val(), Consumer(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state = i % 2
		if state == 0 {
			setProviderValue("value1")
		} else {
			setProviderValue("value2")
		}
	}
}

// BenchmarkContextFlatReconcile measures the reconciliation cost of 100 elements
// wrapped inside a Context.Provider (requiring reconciler-level flattening and context push/pop).
func BenchmarkContextFlatReconcile(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc).(dom.Element)

	ctx := CreateContext("value")

	listA := make([]Node, 100)
	for i := range 100 {
		listA[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, ctx.Provider("value", listA...))

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		listB[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, ctx.Provider("value", listB...))

	b.ResetTimer()
	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}
