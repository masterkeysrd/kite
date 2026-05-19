package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// BoxElement represents a generic container element (like a HTML <div>).
type BoxElement struct {
	elementBase[BoxElement]
}

var _ Element = (*BoxElement)(nil)

// NewBox creates a new generic box container.
func NewBox(doc dom.Document) *BoxElement {
	b := &BoxElement{}
	b.initBase(doc.CreateElement("box", b), b, style.Style{})
	return b
}

// Box creates a new generic box container with the given children.
func Box(children ...any) *BoxElement {
	b := NewBox(orphanDocument)
	processChildren(b, children)
	return b
}
