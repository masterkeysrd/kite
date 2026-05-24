package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// BenchmarkVDOMConstruction measures the performance of constructing a standard VDOM tree.
func BenchmarkVDOMConstruction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	doc := dom.NewDocument()

	type MyButtonProps struct {
		Label string
	}

	myButton := FC("MyButton", func(props MyButtonProps) Node {
		return Button(ButtonProps{
			Class: "fancy-btn",
		}, Text(props.Label))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node2 := myCounter(struct{}{})
		node2.Update(realNode, node1)
		node1 = node2
	}
}
