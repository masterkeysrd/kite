package render

import (
	"iter"

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
	Children() iter.Seq[Object]

	InsertChild(child, before Object)
	RemoveChild(child Object)

	Focusable() bool
	Disabled() bool
	SetDisabled(bool)
	SetFocusable(bool)
	ComputedStyle() *style.Computed
	SetComputedStyle(*style.Computed)
	Flags() DirtyFlag
	MarkDirty(DirtyFlag)
	ClearDirty(DirtyFlag)
	MarkChildrenDirty()
	ClearDirtyRecursive(DirtyFlag)

	IsDetached() bool

	// StyleNode implementation (Task 06)
	RawStyle() style.Style
	SetRawStyle(style.Style)
	ElementDefaultStyle() style.Style
	IsDirtyStyle() bool
	HasDirtyStyleChild() bool
	ClearDirtyStyle()
	ClearChildNeedsStyle()
	StyleParent() style.StyleNode
	StyleFirstChild() style.StyleNode
	StyleNextSibling() style.StyleNode

	// layout.Node implementation (Task 05)
	Style() *style.Computed
	LayoutChildren() iter.Seq[layout.Node]
	IsDirtyLayout() bool
	ClearDirtyLayout()
	Fragment() *layout.Fragment
	CachedLayout(layout.ConstraintSpace) *layout.Fragment
	SetCachedLayout(layout.ConstraintSpace, *layout.Fragment)
	LogicalNode() any
}
