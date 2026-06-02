package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// SpanElement represents an inline container element (like a HTML <span>).
type SpanElement struct {
	elementBase[SpanElement]
}

var _ Element = (*SpanElement)(nil)

// NewSpan creates a new inline span container.
func NewSpan(doc dom.Document) *SpanElement {
	s := &SpanElement{}
	s.initBase(doc.CreateElement("span", s), s, style.S().Display(style.DisplayInline))
	return s
}

// Span creates a new inline span container with the given children.
func Span(children ...any) *SpanElement {
	s := NewSpan(orphanDocument)
	processChildren(s, children)
	return s
}
