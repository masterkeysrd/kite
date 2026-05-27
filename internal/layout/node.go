package layout

import (
	"github.com/masterkeysrd/kite/geom"
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// Node is the layout engine's view of a render object. It provides the
// style, tree-walk, and bounds-mutation interface required by
// formatting-context algorithms without importing the render package.
//
// Any render object that participates in layout must implement this interface.
type Node interface {
	// Style returns the node's fully-resolved computed style.
	// Must not return nil for nodes that participate in layout.
	Style() *style.Computed

	// FirstLayoutChild returns the first layout-visible child of this node.
	FirstLayoutChild() Node

	// NextLayoutSibling returns the next layout-visible sibling of the given child.
	NextLayoutSibling(child Node) Node

	// LogicalNode returns the logical DOM node that owns this render object,
	// typed as any to avoid an import cycle. May be nil.
	LogicalNode() any

	// IsDirtyLayout reports whether this node's layout is stale. The engine
	// uses this signal to decide whether cached intrinsic sizes are still
	// valid.
	IsDirtyLayout() bool

	// IsDirtyPaint reports whether this node's paint is stale.
	IsDirtyPaint() bool

	// HasChildNeedsPaint reports whether any descendant needs paint.
	HasChildNeedsPaint() bool

	// ClearDirtyLayout clears the DirtyLayout flag on this node. Called by
	// the layout engine after successfully measuring the node so that
	// subsequent cache lookups are valid.
	ClearDirtyLayout()

	// Fragment returns the most recent LayoutNG fragment generated for this node.
	// It is used by paint, hit testing, and spatial navigation to retrieve the
	// physical representation of the element.
	Fragment() *Fragment

	// CachedLayout returns a previously computed fragment if the node is clean
	// and the constraints match. Returns nil if a re-layout is required.
	CachedLayout(space ConstraintSpace) *Fragment

	// SetCachedLayout stores the computed fragment and the constraints that generated it.
	// Implementing this should implicitly clear the DirtyLayout flag.
	SetCachedLayout(space ConstraintSpace, frag *Fragment)

	// CachedMinMaxSizes returns the intrinsic minimum and maximum sizes if they are still valid.
	CachedMinMaxSizes() (MinMaxSizes, bool)

	// SetCachedMinMaxSizes stores the computed intrinsic minimum and maximum sizes.
	SetCachedMinMaxSizes(sizes MinMaxSizes)

	// SetOffset updates the physical offset of this node.
	SetOffset(geometry.Point)

	// IsAnonymous reports whether this node is a virtual layout-only node
	// (like flex's AnonymousBlock) that should trigger shrink-wrap for auto width.
	IsAnonymous() bool
}

type TableCellLever interface {
	ColSpan() int
	RowSpan() int
}

func getColSpan(node Node) int {
	if lever, ok := node.LogicalNode().(TableCellLever); ok {
		span := lever.ColSpan()
		if span > 0 {
			return span
		}
	}
	return 1
}

func getRowSpan(node Node) int {
	if lever, ok := node.LogicalNode().(TableCellLever); ok {
		span := lever.RowSpan()
		if span > 0 {
			return span
		}
	}
	return 1
}

type InlineLever interface {
	IsInlineLevel() bool
}

func IsInlineLevel(node Node) bool {
	if lever, ok := node.(InlineLever); ok {
		return lever.IsInlineLevel()
	}

	comp := node.Style()
	return comp.Display == style.DisplayInline || comp.Display == style.DisplayInlineBlock || comp.Display == style.DisplayInlineFlex
}

type OverlayLever interface {
	Anchor() any // Returns dom.Element, but typed as any to avoid cycle
	Placement() geom.Placement
	Flip() bool
}
