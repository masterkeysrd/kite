package dom

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
)

func init() {
	dom.RegisterImplementation(dom.Implementation{
		NewDocument: func() dom.Document {
			return NewDocument()
		},
		NewElement: func(doc dom.Document, tag string, self dom.Node) dom.Element {
			return NewElement(doc, tag, self)
		},
		NewTextNode: func(doc dom.Document, data string, self dom.Node) dom.TextNode {
			return NewTextNode(doc, data, self)
		},
		LayoutChildren: LayoutChildren,
		Outer:          Outer,
		IsUANode:       IsUANode,
		UARoot:         UARoot,
		DefaultScroller: func(host dom.Element) event.Scrollable {
			return DefaultScroller(host)
		},
	})
}
