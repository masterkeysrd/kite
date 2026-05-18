package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
)

// Span represents an inline container element (like a HTML <span>).
type Span struct {
	elementBase[Span]
}

var _ Element = (*Span)(nil)
var _ dom.Lifecycle = (*Span)(nil)

// NewSpan creates a new inline span container.
func NewSpan(doc dom.Document) *Span {
	s := &Span{}
	s.initBase(doc.CreateElement("span", s), s)
	return s
}

// OnConnected handles attaching the element's render object into the render tree.
func (s *Span) OnConnected() {
	if s.RenderObject() == nil {
		ro := render.NewInline(s, s)
		ro.SetRawStyle(s.GetStyle())
		ro.SetDisabled(s.IsDisabled())
		s.SetRenderObject(ro)
	}

	ro := s.RenderObject()
	parentEl := s.Parent()
	if parentEl == nil {
		return
	}

	parentRO := parentEl.RenderObject()
	if parentRO == nil {
		return
	}

	// Insert into the render tree preserving logical order
	var before render.Object
	if next := s.NextSibling(); next != nil {
		before = next.RenderObject()
	}
	parentRO.InsertChild(ro, before)
}

// OnDisconnected handles removing the render object from the render tree.
func (s *Span) OnDisconnected() {
	ro := s.RenderObject()
	if ro == nil {
		return
	}
	if parent := ro.Parent(); parent != nil {
		parent.RemoveChild(ro)
	}
}
