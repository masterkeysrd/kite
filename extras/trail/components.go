package trail

import (
	"strings"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/geom"
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
		emitted, setEmitted := kitex.UseState(0)
		boxRef := kitex.UseRef[dom.Element](nil)

		start := emitted()
		total := len(props.items)

		kitex.UseLayoutEffect(func() {
			if app != nil && start < total {
				if box := boxRef.Current; box != nil {
					for child := box.FirstChild(); child != nil; child = child.NextSibling() {
						app.Engine().InsertAbove(child)
					}
				}
				setEmitted(total)
			}
		}, []any{start, total})

		if start >= total {
			return nil
		}

		children := make([]kitex.Node, 0, total-start)
		for i := start; i < total; i++ {
			if n := props.renderItem(props.items[i], i); n != nil {
				children = append(children, kitex.Box(kitex.BoxProps{}, n))
			}
		}
		if len(children) == 0 {
			return nil
		}

		return kitex.Box(kitex.BoxProps{
			Ref: boxRef,
		}, children...)
	})(staticProps[T]{
		items:      items,
		renderItem: renderItem,
	})
}

// MeasureElement measures the physical dimensions of an element ref in the current layout.
// If the ref is nil or not yet mounted/measured, it returns geom.Size{}.
func MeasureElement(ref kitex.Ref[dom.Element]) geom.Size {
	if ref == nil || ref.Current == nil {
		return geom.Size{}
	}
	doc := ref.Current.OwnerDocument()
	if doc != nil && doc.DefaultView() != nil {
		if rect, ok := doc.DefaultView().GetBoundingClientRect(ref.Current); ok {
			return rect.Size
		}
	}
	return geom.Size{}
}

// Transform splits text into lines, applies transform to each line (if non-nil),
// and returns a Text node containing the joined transformed lines.
func Transform(transform func(line string, index int) string, text string) kitex.Node {
	lines := strings.Split(text, "\n")
	if transform != nil {
		for i, line := range lines {
			lines[i] = transform(line, i)
		}
	}
	return Text(strings.Join(lines, "\n"))
}
