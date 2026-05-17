package dom

// textNode is the concrete, unexported implementation of TextNode.
type textNode struct {
	baseNode
	data string
}

// Compile-time assertion.
var _ TextNode = (*textNode)(nil)

// newTextNode allocates a TextNode with the given data and owner document.
func newTextNode(data string, doc Document) *textNode {
	t := &textNode{data: data}
	t.baseNode.ownerDocument = doc
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
