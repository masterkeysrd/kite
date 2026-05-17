package render

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// Object is the interface for all renderable objects that sit in the render tree.
type Object interface {
	// EventTarget returns the event target associated with this render object.
	// This is typically the logical DOM node.
	EventTarget() event.EventTarget

	Parent() Object
	FirstChild() Object
	LastChild() Object
	NextSibling() Object
	PreviousSibling() Object

	Bounds() layout.Rect
	SetBounds(layout.Rect)

	Focusable() bool
	Disabled() bool
	ComputedStyle() *style.Computed
	MarkDirty(DirtyFlag)
	MarkChildrenDirty()
}
