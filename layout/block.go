package layout

import (
	"github.com/masterkeysrd/kite/style"
)

// BlockAlgorithm implements the Block formatting context layout.
// It stacks children vertically (in the block-flow direction) and calculates
// its own intrinsic size based on the accumulated height of its children.
type BlockAlgorithm struct {
	Node  Node
	Space ConstraintSpace
}

// Layout executes the block layout algorithm and returns an immutable Fragment.
func (a *BlockAlgorithm) Layout() *Fragment {
	// 0. Return cached fragment immediately if constraints match and the node is clean.
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	var children []FragmentLink

	// Get computed style to extract border and padding.
	// In a border-box model, margin is NOT part of the node's own bounds.
	// Margins are spacing applied by the *parent* when positioning this node.
	comp := a.Node.Style()
	border := comp.Border.Width
	padding := comp.Padding

	// Internal content boundaries for positioning children.
	insetX := border.Left + padding.Left
	currentY := border.Top + padding.Top

	// Calculate the parent's horizontal decoration size (padding + border).
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right

	// We track the maximum width footprint (width + horizontal margins) of any child.
	maxChildWidth := 0

	// 1. Iterate over all direct layout children.
	for child := range a.Node.LayoutChildren() {
		childMargin := child.Style().Margin

		// 2. Build the ConstraintSpace for the child.
		// A block child can consume all available width (minus parent padding/borders and its own margins).
		// Its max height is bounded by what remains in the parent's max constraints.
		availWidth := max(0, a.Space.Constraints.Max.Width-parentDecorX-childMargin.Left-childMargin.Right)
		availHeight := max(0, a.Space.Constraints.Max.Height-currentY-childMargin.Top-childMargin.Bottom-(border.Bottom+padding.Bottom))

		childSpace := ConstraintSpace{
			Constraints: Constraints{
				Min: Size{Width: 0, Height: 0},
				Max: Size{Width: availWidth, Height: availHeight},
			},
		}

		// 3. Dispatch to the appropriate layout algorithm for the child.
		childAlgo := &BlockAlgorithm{
			Node:  child,
			Space: childSpace,
		}

		// 4. Perform the layout pass for the child.
		childFrag := childAlgo.Layout()

		// 5. Record the child fragment and its physical offset.
		// A child in a block flow starts at the parent's left inset plus its own left margin.
		// Vertically, it's pushed down by its top margin.
		children = append(children, FragmentLink{
			Offset: Point{
				X: insetX + childMargin.Left,
				Y: currentY + childMargin.Top,
			},
			Fragment: childFrag,
		})

		// 6. Accumulate dimensions.
		// Advance the Y cursor by the child's top margin, the fragment height, and the child's bottom margin.
		currentY += childMargin.Top + childFrag.Size.Height + childMargin.Bottom

		// The footprint of the child includes its horizontal margins.
		childFootprint := childMargin.Left + childFrag.Size.Width + childMargin.Right
		if childFootprint > maxChildWidth {
			maxChildWidth = childFootprint
		}
	}

	// 7. Add bottom decoration to the accumulated height.
	currentY += border.Bottom + padding.Bottom

	// The intrinsic width is the max child footprint plus our inner decorations.
	intrinsicWidth := maxChildWidth + parentDecorX

	// Resolve explicit dimensions from the computed style.
	// This ensures an element forces its size if Width or Height are set,
	// instead of always shrink-wrapping its intrinsic size.
	var explicitWidth, explicitHeight int
	var hasExplicitWidth, hasExplicitHeight bool

	if comp.Width.Kind() == style.KindCells {
		explicitWidth = comp.Width.CellsValue()
		hasExplicitWidth = true
	} else if comp.Width.Kind() == style.KindPercent {
		explicitWidth = int(float32(a.Space.Constraints.Max.Width) * (comp.Width.PercentValue() / 100.0))
		hasExplicitWidth = true
	}

	if comp.Height.Kind() == style.KindCells {
		explicitHeight = comp.Height.CellsValue()
		hasExplicitHeight = true
	} else if comp.Height.Kind() == style.KindPercent {
		explicitHeight = int(float32(a.Space.Constraints.Max.Height) * (comp.Height.PercentValue() / 100.0))
		hasExplicitHeight = true
	}

	if hasExplicitWidth {
		intrinsicWidth = explicitWidth
	}
	if hasExplicitHeight {
		currentY = explicitHeight
	}

	// 8. Compute the final physical size of this block fragment
	// clamped by the min/max constraints of the space.
	finalWidth := max(a.Space.Constraints.Min.Width, min(intrinsicWidth, a.Space.Constraints.Max.Width))
	finalHeight := max(a.Space.Constraints.Min.Height, min(currentY, a.Space.Constraints.Max.Height))

	// 9. Return the immutable layout fragment.
	frag := &Fragment{
		Size:     Size{Width: finalWidth, Height: finalHeight},
		Node:     a.Node,
		Children: children,
	}

	// 9. Store the result in the cache so subsequent identical passes can skip this work.
	a.Node.SetCachedLayout(a.Space, frag)

	return frag
}
