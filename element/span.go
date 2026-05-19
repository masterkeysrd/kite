package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// Span represents an inline container element (like a HTML <span>).
type Span struct {
	elementBase[Span]
}

var _ Element = (*Span)(nil)

// NewSpan creates a new inline span container.
func NewSpan(doc dom.Document) *Span {
	s := &Span{}
	s.initBase(doc.CreateElement("span", s), s, style.Style{
		Display: style.Some(style.DisplayInline),
	})
	return s
}
