package layout

import "github.com/masterkeysrd/kite/style"

// BuildChildSpace constructs a ConstraintSpace for a block-level child element.
//
// Parameters:
//   - child: the child layout node (for computed style access).
//   - containerSpace: the parent's content-box dimensions, pre-adjusted for the current
//     block offset so that Height reflects the remaining vertical space.
//   - containingSpace: the parent's border-box dimensions, used as the base for
//     KindPercent resolution (ADR-018).
//   - parentSpace: the parent's own ConstraintSpace (for IsFixedBlockSize and BreakToken).
//
// This function centralises the child constraint generation that was previously
// duplicated across BlockAlgorithm and ListAlgorithm (ADR-018).
func BuildChildSpace(child Node, containerSpace Size, containingSpace Size, parentSpace ConstraintSpace) ConstraintSpace {
	childStyle := child.Style()
	childMargin := childStyle.Margin

	childAvailWidth := max(0, containerSpace.Width-childMargin.Left-childMargin.Right)
	childAvailHeight := max(0, containerSpace.Height-childMargin.Top-childMargin.Bottom)

	b := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
	b.SetContainingSpace(containingSpace)
	b.SetContainerSpace(containerSpace)

	// Resolve inline (width) size.
	switch childStyle.Width.Kind() {
	case style.KindCells:
		b.SetIsFixedInlineSize(true)
		b.space.AvailableSize.Width = childStyle.Width.CellsValue()
	case style.KindPercent:
		// Percentage resolves against the parent's border-box (containingSpace), not the
		// content-box. This is the correct behaviour per ADR-017 / ADR-018.
		b.SetIsFixedInlineSize(true)
		b.space.AvailableSize.Width = int(float32(containingSpace.Width) * childStyle.Width.PercentValue() / 100.0)
	case style.KindAuto:
		// Tables shrink-wrap; all other block-level elements stretch to fill.
		if childStyle.Display != style.DisplayTable {
			b.SetIsFixedInlineSize(true)
			b.space.AvailableSize.Width = childAvailWidth
		}
	case style.KindMaxContent:
		// Do NOT set IsFixedInlineSize — the child's own algorithm calls
		// ComputeMinMaxSizes and uses the unconstrained max-content width.
	}

	// Resolve block (height) size.
	switch childStyle.Height.Kind() {
	case style.KindCells:
		b.SetIsFixedBlockSize(true)
		b.space.AvailableSize.Height = childStyle.Height.CellsValue()
	case style.KindPercent:
		// Percentage height is only defined when the parent has a fixed block size.
		if parentSpace.IsFixedBlockSize {
			b.SetIsFixedBlockSize(true)
			b.space.AvailableSize.Height = int(float32(containingSpace.Height) * childStyle.Height.PercentValue() / 100.0)
		}
	}

	if parentSpace.BreakToken != nil {
		b.space.BreakToken = parentSpace.BreakToken
	}

	return b.ToConstraintSpace()
}
