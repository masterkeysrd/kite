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
	if t.data == data {
		return
	}
	t.data = data
	t.MarkNeedsSync()

	// Notify the render tree. We need to find the nearest ancestor that has
	// a render object. For nodes in a UA subtree, we use the host element
	// (outer pointer) as the starting point for the walk-up.
	var start Node = t.parent
	if t.inUASubtree && t.outer != nil {
		start = t.outer
	}

	for p := start; p != nil; p = p.Parent() {
		if ro := p.RenderObject(); ro != nil {
			ro.MarkChildrenDirty()
			break
		}
	}
}

// asBase returns the underlying *baseNode.
func (t *textNode) asBase() *baseNode { return &t.baseNode }

// CreateRenderObject implements render.CustomObjectProvider.
func (t *textNode) CreateRenderObject() render.Object {
	// Use the actual text node as the logical node so the layout engine can
	// call Data() on it. Use t.EventTarget() so that UA-subtree text nodes
	// dispatch events to the host element (ADR-0036, ADR-009).
	return render.NewText(t, t.EventTarget())
}
