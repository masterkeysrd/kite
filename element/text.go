package element

import (
	"github.com/masterkeysrd/kite/dom"
)

// Text represents a text node in the logical tree.
type Text struct {
	dom.TextNode
}

var _ dom.TextNode = (*Text)(nil)

// NewText creates a new text node.
func NewText(doc dom.Document, data string) *Text {
	t := &Text{}
	t.TextNode = doc.CreateTextNode(data, t)
	return t
}

func (t *Text) Data() string        { return t.TextNode.Data() }
func (t *Text) SetData(data string) { t.TextNode.SetData(data) }
func (t *Text) Unwrap() dom.Node    { return t.TextNode }
