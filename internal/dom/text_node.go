package dom

import (
	"github.com/masterkeysrd/kite/dom"
)

// TextNode is the concrete, exported implementation of TextNode.
type TextNode struct {
	BaseNode
	data string
}

// Compile-time assertion.
var _ dom.TextNode = (*TextNode)(nil)

// NewTextNode allocates a TextNode with the given data and owner document.
// If self is nil, the node's identity is itself.
func NewTextNode(doc dom.Document, data string, self dom.Node) *TextNode {
	return newTextNode(data, doc, self)
}

// newTextNode allocates a TextNode with the given data and owner document.
func newTextNode(data string, doc dom.Document, self dom.Node) *TextNode {
	t := &TextNode{data: data}
	t.ownerDocument = doc
	if self == nil {
		t.self = t
	} else {
		t.self = self
	}
	t.kind = dom.KindText
	t.name = "#text"
	return t
}

// Data returns the current text content.
func (t *TextNode) Data() string { return t.data }

// SetData replaces the text content and notifies the parent's render object.
func (t *TextNode) SetData(data string) {
	if t.data == data {
		return
	}
	t.data = data
	if d, ok := t.ownerDocument.(*Document); ok && d != nil {
		d.InvalidateTextNodeCache()
	}
	t.MarkNeedsSync()
}

// asBase returns the underlying *BaseNode.
func (t *TextNode) asBase() *BaseNode { return &t.BaseNode }
