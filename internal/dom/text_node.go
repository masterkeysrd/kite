package dom

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/internal/render"
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
	t.MarkNeedsSync()

	// Mark the text node's own render object as dirty if it exists.
	if ro := t.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
	}

	// Notify the render tree. We need to find the nearest ancestor that has
	// a render object. For nodes in a UA subtree, we use the host element
	// (outer pointer) as the starting point for the walk-up.
	var start = t.parent
	if t.inUASubtree && t.outer != nil {
		start = t.outer
	}

	for p := start; p != nil; p = p.Parent() {
		if ro := p.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyLayout)
			break
		}
	}
}

// asBase returns the underlying *BaseNode.
func (t *TextNode) asBase() *BaseNode { return &t.BaseNode }

// CreateRenderObject implements render.CustomObjectProvider.
func (t *TextNode) CreateRenderObject() render.Object {
	// Use the actual text node as the logical node so the layout engine can
	// call Data() on it. Use t.EventTarget() so that UA-subtree text nodes
	// dispatch events to the host element (ADR-0036, ADR-009).
	return render.NewText(t, t.EventTarget())
}
