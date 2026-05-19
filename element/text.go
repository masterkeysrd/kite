package element

import (
	"github.com/masterkeysrd/kite/dom"
)

// TextElement represents a text node in the logical tree.
type TextElement struct {
	dom.TextNode
}

var _ dom.TextNode = (*TextElement)(nil)

// NewText creates a new text node.
func NewText(doc dom.Document, data string) *TextElement {
	t := &TextElement{}
	t.TextNode = doc.CreateTextNode(data, t)
	return t
}

// Text creates a new text node with the given data.
func Text(data string) *TextElement {
	return NewText(orphanDocument, data)
}

func (t *TextElement) Data() string        { return t.TextNode.Data() }
func (t *TextElement) SetData(data string) { t.TextNode.SetData(data) }
func (t *TextElement) Unwrap() dom.Node    { return t.TextNode }
