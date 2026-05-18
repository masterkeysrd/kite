package layout

import (
	"github.com/masterkeysrd/kite/text"
)

// Fragment represents the immutable output of a layout algorithm.
// Once created, a Fragment's fields must never be modified. This immutability
// allows fragments to be cached and reused across layout passes.
type Fragment struct {
	// Size is the computed physical dimensions of this fragment in terminal cells.
	Size Size

	// Node is the layout node that generated this fragment.
	Node Node

	// Children contains the positioned child fragments relative to this fragment.
	Children []FragmentLink

	// Text contains the shaped clusters if this fragment represents a text run.
	Text []text.Cluster

	// ParentNode is the containing inline element for text fragments (for style inheritance).
	ParentNode Node
}

// FragmentLink connects a child Fragment to its parent at a specific physical offset.
// Positioning information is stored here rather than inside the Fragment itself,
// allowing the exact same Fragment to be reused in different positions.
type FragmentLink struct {
	// Offset is the physical position of the child relative to the parent fragment's origin.
	Offset Point

	// Fragment is the immutable child fragment.
	Fragment *Fragment
}

// ConstraintSpace defines the inputs for a layout operation. It encapsulates the
// physical size constraints (Min/Max) alongside any additional context required
// during the layout walk (e.g., percentage resolution sizes, exclusion spaces for floats).
type ConstraintSpace struct {
	// AvailableSize is the ideal size the node should consume, provided by the parent.
	AvailableSize Size

	// PercentageResolutionSize is the size used to resolve percentage-based dimensions.
	PercentageResolutionSize Size

	// IsFixedInlineSize indicates the inline size (width) is pre-determined.
	IsFixedInlineSize bool

	// IsFixedBlockSize indicates the block size (height) is pre-determined.
	IsFixedBlockSize bool
}

// MinMaxSizes represents the intrinsic minimum and maximum widths of a node.
type MinMaxSizes struct {
	Min, Max int
}

// Encompass expands the min/max bounds to fit another MinMaxSizes.
func (m *MinMaxSizes) Encompass(other MinMaxSizes) {
	m.Min = max(m.Min, other.Min)
	m.Max = max(m.Max, other.Max)
}

// EncompassSize expands the min/max bounds to fit an explicit value.
func (m *MinMaxSizes) EncompassSize(value int) {
	m.Min = max(m.Min, value)
	m.Max = max(m.Max, value)
}

// Constrain caps the boundaries (min/max) to a specific value.
func (m MinMaxSizes) Constrain(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: min(m.Min, value),
		Max: min(m.Max, value),
	}
}

// Add shifts both min and max sizes simultaneously.
func (m MinMaxSizes) Add(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: m.Min + value,
		Max: m.Max + value,
	}
}

// Subtract shifts both min and max sizes simultaneously.
func (m MinMaxSizes) Subtract(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: max(0, m.Min-value),
		Max: max(0, m.Max-value),
	}
}

// Algorithm is the interface that all LayoutNG-inspired layout formatters must implement.
type Algorithm interface {
	// Layout computes and returns an immutable Fragment based on the underlying node and constraints.
	Layout() *Fragment

	// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
	ComputeMinMaxSizes() MinMaxSizes
}

// AbsoluteBounds traverses the fragment tree starting at root and computes the absolute
// bounding rectangle of the target node. Returns the rect and true if found, or a zero
// rect and false if the node is not present in the tree.
func AbsoluteBounds(root *Fragment, target Node) (Rect, bool) {
	if root == nil {
		return Rect{}, false
	}
	if root.Node == target {
		return Rect{Origin: Point{0, 0}, Size: root.Size}, true
	}
	for _, childLink := range root.Children {
		if rect, found := AbsoluteBounds(childLink.Fragment, target); found {
			// Add this link's offset to the child's absolute origin.
			rect.Origin.X += childLink.Offset.X
			rect.Origin.Y += childLink.Offset.Y
			return rect, true
		}
	}
	return Rect{}, false
}
