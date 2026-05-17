package layout

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
	// Constraints defines the minimum and maximum dimensions allowed for this layout pass.
	// A fixed size is simply represented by Min == Max.
	Constraints Constraints
}

// Algorithm is the interface that all LayoutNG-inspired layout formatters must implement.
type Algorithm interface {
	// Layout computes and returns an immutable Fragment based on the underlying node and constraints.
	Layout() *Fragment
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
