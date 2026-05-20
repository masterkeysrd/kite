package dom

import (
	"github.com/masterkeysrd/kite/render"
)

// textNode is the concrete, unexported implementation of TextNode.
type textNode struct {
	baseNode
	data string
}

// Compile-time assertion.
var _ TextNode = (*textNode)(nil)

// NewTextNode allocates a TextNode with the given data and owner document.
// If self is nil, the node's identity is itself.
func NewTextNode(doc Document, data string, self Node) TextNode {
	return newTextNode(data, doc, self)
}

// newTextNode allocates a TextNode with the given data and owner document.
func newTextNode(data string, doc Document, self Node) *textNode {
	t := &textNode{data: data}
	t.ownerDocument = doc
	if self == nil {
		t.self = t
	} else {
		t.self = self
	}
	t.kind = KindText
	t.name = "#text"
	return t
}

// Data returns the current text content.
func (t *textNode) Data() string { return t.data }

// SetData replaces the text content and notifies the parent's render object.
func (t *textNode) SetData(data string) {
	t.data = data
	if p := t.parent; p != nil {
		if ro := p.RenderObject(); ro != nil {
			ro.MarkChildrenDirty()
		}
	}
}

// asBase returns the underlying *baseNode.
func (t *textNode) asBase() *baseNode { return &t.baseNode }

// CreateRenderObject implements render.CustomObjectProvider.
func (t *textNode) CreateRenderObject() render.Object {
	return render.NewText(t.self, t.self)
}
