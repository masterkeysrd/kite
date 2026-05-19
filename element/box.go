package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// Box represents a generic container element (like a HTML <div>).
type Box struct {
	elementBase[Box]
}

var _ Element = (*Box)(nil)

// NewBox creates a new generic box container.
func NewBox(doc dom.Document) *Box {
	b := &Box{}
	b.initBase(doc.CreateElement("box", b), b, style.Style{})
	return b
}
