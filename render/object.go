package render

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// HostNode is the interface that logical nodes must implement to host a render object.
// This is typically implemented by dom.Node.
type HostNode interface {
	RenderObject() Object
	SetRenderObject(Object)
}

// RenderObjectHook is an optional interface that logical nodes can implement
// to be notified when their render object is created or replaced.
type RenderObjectHook interface {
	OnRenderObjectCreated(ro Object)
}

// CustomObjectProvider is an interface that logical nodes can implement
// to provide a custom render object instead of the default render.Box.
type CustomObjectProvider interface {
	CreateRenderObject() Object
}

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
	DefaultStyle() style.Style
	IntrinsicStyle() style.Style
	IsDirtyStyle() bool
	HasDirtyStyleChild() bool
	ClearDirtyStyle()
	ClearChildNeedsStyle()
	StyleParent() style.StyleNode
	StyleFirstChild() style.StyleNode
	StyleNextSibling() style.StyleNode

	// layout.Node implementation (Task 05)
	Style() *style.Computed
	FirstLayoutChild() layout.Node
	NextLayoutSibling(layout.Node) layout.Node
	IsDirtyLayout() bool
	IsDirtyPaint() bool
	HasChildNeedsPaint() bool
	ClearDirtyLayout()
	Fragment() *layout.Fragment
	CachedLayout(layout.ConstraintSpace) *layout.Fragment
	SetCachedLayout(layout.ConstraintSpace, *layout.Fragment)
	CachedMinMaxSizes() (layout.MinMaxSizes, bool)
	SetCachedMinMaxSizes(layout.MinMaxSizes)
	LogicalNode() any

	// Offset returns the physical offset of this object relative to its parent.
	// For most objects, this is managed by the parent's layout algorithm.
	// For overlays, this is calculated by the overlay's own layout algorithm.
	Offset() layout.Point
	// SetOffset updates the physical offset of this object.
	SetOffset(layout.Point)

	// IsAnonymous reports whether this object is a virtual layout-only node.
	IsAnonymous() bool

	// MaxScroll returns the maximum horizontal and vertical scroll offsets
	// based on the current layout fragment and content extent.
	MaxScroll() (x, y int)
}
