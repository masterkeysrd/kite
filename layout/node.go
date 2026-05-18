package layout

import (
	"iter"

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

	// LayoutChildren returns an iterator over the node's direct children in
	// tree order. Named LayoutChildren (not Children) so that render objects
	// can implement both this interface and a separate Children() method that
	// returns typed render objects without a method-set conflict.
	LayoutChildren() iter.Seq[Node]

	// LogicalNode returns the logical DOM node that owns this render object,
	// typed as any to avoid an import cycle. May be nil.
	LogicalNode() any

	// IsDirtyLayout reports whether this node's layout is stale. The engine
	// uses this signal to decide whether cached intrinsic sizes are still
	// valid.
	IsDirtyLayout() bool

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
}
