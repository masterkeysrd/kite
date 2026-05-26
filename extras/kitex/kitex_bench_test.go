package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// BenchmarkVDOMConstruction measures the performance of constructing a standard VDOM tree.
func BenchmarkVDOMConstruction(b *testing.B) {
	EnableDevMode = false

	for b.Loop() {
		_ = Box(BoxProps{
			ID:    "container",
			Class: "wrapper",
		},
			Button(ButtonProps{
				ID:       "btn-ok",
				Disabled: true,
			}, Text("OK")),
			Button(ButtonProps{
				ID: "btn-cancel",
			}, Text("Cancel")),
		)
	}
}

// BenchmarkComponentInstantiation measures functional component rendering and instantiation overhead.
func BenchmarkComponentInstantiation(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()

	type MyButtonProps struct {
		Label string
	}

	myButton := FC("MyButton", func(props MyButtonProps) Node {
		return Button(ButtonProps{
			Class: "fancy-btn",
		}, Text(props.Label))
	})

	for b.Loop() {
		compNode := myButton(MyButtonProps{Label: "click me"})
		_ = compNode.Instantiate(doc)
	}
}

// BenchmarkComponentHooks measures the performance of rendering a component that uses UseState hooks.
func BenchmarkComponentHooks(b *testing.B) {
	doc := dom.NewDocument()

	myCounter := FC("Counter", func(props struct{}) Node {
		getVal1, setVal1 := UseState(0)
		getVal2, setVal2 := UseState("hello")

		if getVal1() == 0 {
			setVal1(1)
		}
		if getVal2() == "hello" {
			setVal2("world")
		}

		return Box(BoxProps{})
	})

	for b.Loop() {
		compNode := myCounter(struct{}{})
		_ = compNode.Instantiate(doc)
	}
}

// BenchmarkComponentUpdate measures the performance of calling Update on a component with hooks.
func BenchmarkComponentUpdate(b *testing.B) {
	doc := dom.NewDocument()

	myCounter := FC("Counter", func(props struct{}) Node {
		getVal, setVal := UseState(0)
		if getVal() < 100 {
			setVal(getVal() + 1)
		}
		return Box(BoxProps{})
	})

	// Initial render
	node1 := myCounter(struct{}{})
	realNode := node1.Instantiate(doc)

	for b.Loop() {
		node2 := myCounter(struct{}{})
		node2.Update(realNode, node1)
		node1 = node2
	}
}

// --- Memoization benchmarks ---------------------------------------------------

// buildRichTree creates a component that renders a deeply nested tree with score > 5.
// This simulates a realistic "complex" component that would benefit from memoization.
func buildRichTree(label string) Node {
	return Box(BoxProps{},
		Span(SpanProps{}, Text(label)),
		Box(BoxProps{},
			Text("row1-col1"), Text("row1-col2"),
		),
		Box(BoxProps{},
			Text("row2-col1"), Text("row2-col2"),
		),
	)
}

// BenchmarkMemoizedUpdateIdenticalProps measures the cost of Update() when
// memoization fires — i.e., shouldMemo=true and props are deeply equal.
// In this path the RenderFn is skipped entirely; only a deepEqualProps reflection
// walk is performed.
//
// Performance delta vs BenchmarkMemoizedUpdateChangedProps (below) documents the
// savings that automatic memoization delivers on complex subtrees.
func BenchmarkMemoizedUpdateIdenticalProps(b *testing.B) {
	doc := dom.NewDocument()

	type RichProps struct{ Label string }

	myComp := FC("RichComp", func(props RichProps) Node {
		return buildRichTree(props.Label)
	})

	node1 := myComp(RichProps{Label: "hello"})
	realNode := node1.Instantiate(doc)

	// Confirm memoization is active.
	comp := node1.(*ComponentNode[RichProps])
	if !comp.shouldMemo {
		b.Fatalf("shouldMemo must be true for this benchmark (score=%d)", comp.complexityScore)
	}

	for b.Loop() {
		node2 := myComp(RichProps{Label: "hello"}) // identical props
		node2.Update(realNode, node1)
		node1 = node2
	}
}

// BenchmarkMemoizedUpdateChangedProps measures the cost of Update() when props
// differ and the RenderFn must execute. This is the baseline (no memo saving).
func BenchmarkMemoizedUpdateChangedProps(b *testing.B) {
	doc := dom.NewDocument()

	type RichProps struct{ Label string }

	myComp := FC("RichComp", func(props RichProps) Node {
		return buildRichTree(props.Label)
	})

	node1 := myComp(RichProps{Label: "hello"})
	realNode := node1.Instantiate(doc)

	for i := 0; b.Loop(); i++ {
		// Alternate labels so props always differ, defeating the memo.
		label := "hello"
		if i%2 == 0 {
			label = "world"
		}
		node2 := myComp(RichProps{Label: label})
		node2.Update(realNode, node1)
		ReleaseTree(node1)
		node1 = node2
	}
}

// BenchmarkUseMemoHitRate measures how fast UseMemo returns a cached value
// when deps do not change between renders (the common hot path).
func BenchmarkUseMemoHitRate(b *testing.B) {
	doc := dom.NewDocument()

	myComp := FC("UseMemoComp", func(props struct{}) Node {
		_ = UseMemo(func() []int {
			// Simulated "expensive" computation.
			out := make([]int, 100)
			for i := range out {
				out[i] = i * i
			}
			return out
		}, []any{"stable-key"})
		return Box(BoxProps{})
	})

	node1 := myComp(struct{}{})
	realNode := node1.Instantiate(doc)

	for b.Loop() {
		node2 := myComp(struct{}{})
		node2.Update(realNode, node1)
		node1 = node2
	}
}
