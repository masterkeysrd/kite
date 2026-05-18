package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
)

// Text represents a text node in the logical tree.
type Text struct {
	dom.TextNode
}

var _ dom.TextNode = (*Text)(nil)
var _ dom.Lifecycle = (*Text)(nil)

// NewText creates a new text node.
func NewText(doc dom.Document, data string) *Text {
	t := &Text{}
	t.TextNode = doc.CreateTextNode(data, t)
	return t
}

func (t *Text) Data() string        { return t.TextNode.Data() }
func (t *Text) SetData(data string) { t.TextNode.SetData(data) }
func (t *Text) Unwrap() dom.Node    { return t.TextNode }

// OnConnected handles attaching the text node's render object into the render tree.
func (t *Text) OnConnected() {
	if t.RenderObject() == nil {
		ro := render.NewText(t, t)
		// Text nodes inherit styles, so we don't necessarily set raw style here
		// unless there are specific text styles to apply.
		t.SetRenderObject(ro)
	}

	ro := t.RenderObject()
	parentEl := t.Parent()
	if parentEl == nil {
		return
	}

	parentRO := parentEl.RenderObject()
	if parentRO == nil {
		return
	}

	// Insert into the render tree preserving logical order
	var before render.Object
	if next := t.NextSibling(); next != nil {
		before = next.RenderObject()
	}
	parentRO.InsertChild(ro, before)
}

// OnDisconnected handles removing the render object from the render tree.
func (t *Text) OnDisconnected() {
	ro := t.RenderObject()
	if ro == nil {
		return
	}
	if parent := ro.Parent(); parent != nil {
		parent.RemoveChild(ro)
	}
}
