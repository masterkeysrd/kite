package trail

import (
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
)

// Re-exported types for convenience.
type (
	Node      = kitex.Node
	BoxProps  = kitex.BoxProps
	SpanProps = kitex.SpanProps
)

// Re-exported primitives and component constructors from kitex.
var (
	Box       = kitex.Box
	Text      = kitex.Text
	Span      = kitex.Span
	Fragment  = kitex.Fragment
	SimpleFC  = kitex.SimpleFC
	SimpleFCC = kitex.SimpleFCC
)

// FC creates a functional component wrapper that does not take children.
func FC[P any](name string, render func(P) Node) func(P) Node {
	return kitex.FC(name, render)
}

// FCC creates a functional component wrapper that can accept children.
func FCC[P any](name string, render func(P) Node) func(P, ...Node) Node {
	return kitex.FCC(name, render)
}

// Newline renders n (default 1) line breaks as a box of height n.
func Newline(count ...int) kitex.Node {
	n := 1
	if len(count) > 0 {
		n = count[0]
	}
	if n < 0 {
		n = 0
	}
	return kitex.Box(kitex.BoxProps{
		Style: style.S().Height(style.Cells(n)),
	})
}

// Spacer creates a flexible space component that expands to fill available flex space.
func Spacer() kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: style.S().Flex(1),
	})
}

type staticProps[T any] struct {
	items      []T
	renderItem func(item T, index int) kitex.Node
}

// Static renders completed items permanently above the active inline frame.
// Newly appended items are flushed once into terminal scrollback and do not
// occupy space in the live inline DOM.
func Static[T any](items []T, renderItem func(item T, index int) kitex.Node) kitex.Node {
	return kitex.FC("Static", func(props staticProps[T]) kitex.Node {
		app := UseApp()
		emitted := kitex.UseRef(0)

		kitex.UseLayoutEffect(func() {
			start := emitted.Current
			total := len(props.items)
			if app != nil && start < total {
				for i := start; i < total; i++ {
					if n := props.renderItem(props.items[i], i); n != nil {
						app.InsertAbove(n)
					}
				}
				emitted.Current = total
			}
		}, nil)

		return nil
	})(staticProps[T]{
		items:      items,
		renderItem: renderItem,
	})
}
