package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// Box represents a generic container element (like a HTML <div>).
type Box struct {
	elementBase[Box]
}

var _ Element = (*Box)(nil)
var _ dom.Lifecycle = (*Box)(nil)

// NewBox creates a new generic box container.
func NewBox(doc dom.Document) *Box {
	b := &Box{}
	b.initBase(doc.CreateElement("box", b), b)

	// We must register the render object creation on mount,
	// but the actual linking requires OnConnected lifecycle hook.
	return b
}

// OnConnected handles attaching the element's render object into the render tree.
func (b *Box) OnConnected() {
	if b.RenderObject() == nil {
		display := b.GetStyle().Display.UnwrapOr(style.DisplayBlock)
		var ro render.Object
		if display == style.DisplayFlex || display == style.DisplayInlineFlex {
			ro = render.NewFlex(b, b)
		} else {
			ro = render.NewBlock(b, b)
		}
		ro.SetRawStyle(b.GetStyle())
		ro.SetDisabled(b.IsDisabled())
		ro.SetFocusable(b.TagName() == "button" || b.TagName() == "input") // Generic fallback
		b.SetRenderObject(ro)
	}

	ro := b.RenderObject()
	parentEl := b.Parent()
	if parentEl == nil {
		return
	}

	parentRO := parentEl.RenderObject()
	if parentRO == nil {
		return
	}

	// Insert into the render tree preserving logical order
	var before render.Object
	if next := b.NextSibling(); next != nil {
		before = next.RenderObject()
	}
	parentRO.InsertChild(ro, before)
}

// OnDisconnected handles removing the render object from the render tree.
func (b *Box) OnDisconnected() {
	ro := b.RenderObject()
	if ro == nil {
		return
	}
	if parent := ro.Parent(); parent != nil {
		parent.RemoveChild(ro)
	}
}
