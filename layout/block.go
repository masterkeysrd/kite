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
	// 1. Cache Check: Return cached fragment if constraints match and the node is clean.
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()
	border := comp.Border.Width
	padding := comp.Padding

	// 2. Intrinsic Sizing: If inline size is not fixed, use ComputeMinMaxSizes.
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes()
	}

	// Resolve the resolved inline size (width).
	var resolvedInlineSize int
	if a.Space.IsFixedInlineSize {
		resolvedInlineSize = a.Space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(a.Space.PercentageResolutionSize.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			// Block elements stretch to available width by default.
			resolvedInlineSize = a.Space.AvailableSize.Width
		case style.KindContent:
			// Content (shrink-wrap)
			resolvedInlineSize = min(minMax.Max, a.Space.AvailableSize.Width)
		default:
			// Fallback to stretching for block elements as per specification
			resolvedInlineSize = a.Space.AvailableSize.Width
		}
	}

	// 3. Setup Builder: Initialize BoxFragmentBuilder and resolve FragmentGeometry.
	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	builder.SetInlineSize(resolvedInlineSize)

	// Internal content boundaries for positioning children.
	insetX := border.Left + padding.Left

	// Parent horizontal and vertical decoration size.
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right
	parentDecorY := border.Top + border.Bottom + padding.Top + padding.Bottom

	// 4. Child Iteration: Loop through in-flow layout children.
	for child := range a.Node.LayoutChildren() {
		childStyle := child.Style()
		childMargin := childStyle.Margin

		// 5. Constraint Generation: Use ConstraintSpaceBuilder.
		// Available width for child is our resolved width minus our decorations and child margins.
		childAvailWidth := max(0, resolvedInlineSize-parentDecorX-childMargin.Left-childMargin.Right)
		
		// For block height, we pass down what's left of our available height.
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-childMargin.Top-childMargin.Bottom-(border.Bottom+padding.Bottom))

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{
			Width:  max(0, resolvedInlineSize-parentDecorX),
			Height: max(0, a.Space.AvailableSize.Height-parentDecorY),
		})
		
		// If child has explicit width, we can set IsFixedInlineSize.
		if childStyle.Width.Kind() == style.KindCells {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = childStyle.Width.CellsValue()
		} else if childStyle.Width.Kind() == style.KindPercent {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = int(float32(resolvedInlineSize-parentDecorX) * childStyle.Width.PercentValue() / 100.0)
		} else if childStyle.Width.Kind() == style.KindAuto {
			// Auto width on a block child also results in a fixed inline size (stretched)
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = childAvailWidth
		}

		if childStyle.Height.Kind() == style.KindCells {
			childSpaceBuilder.SetIsFixedBlockSize(true)
			childSpaceBuilder.space.AvailableSize.Height = childStyle.Height.CellsValue()
		} else if childStyle.Height.Kind() == style.KindPercent {
			// Height percentage resolution is only possible if we have a fixed height or similar.
			// For now, resolve against our available height if it's fixed.
			if a.Space.IsFixedBlockSize {
				childSpaceBuilder.SetIsFixedBlockSize(true)
				childSpaceBuilder.space.AvailableSize.Height = int(float32(a.Space.AvailableSize.Height-(border.Top+border.Bottom+padding.Top+padding.Bottom)) * childStyle.Height.PercentValue() / 100.0)
			}
		}

		childSpace := childSpaceBuilder.ToConstraintSpace()

		// 6. Child Layout & Positioning.
		childAlgo := &BlockAlgorithm{
			Node:  child,
			Space: childSpace,
		}
		childFrag := childAlgo.Layout()

		// Position the child fragment.
		offset := Point{
			X: insetX + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)

		// Advance the block offset (Y cursor).
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)
	}

	// Final block size includes bottom decorations.
	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)
	
	// If height is fixed, use that instead.
	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		builder.SetBlockSize(builder.CurrentBlockOffset())
	}

	// 7. Finalization: Invoke ToFragment() to seal the immutable fragment.
	frag := builder.ToFragment()

	// Store the result in the cache.
	a.Node.SetCachedLayout(a.Space, frag)

	return frag
}

// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
func (a *BlockAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	// 1. Cache Check.
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}

	comp := a.Node.Style()
	border := comp.Border.Width
	padding := comp.Padding
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right

	var result MinMaxSizes

	// If width is explicitly set, min and max are that width.
	if comp.Width.Kind() == style.KindCells {
		val := comp.Width.CellsValue()
		result = MinMaxSizes{Min: val, Max: val}
		a.Node.SetCachedMinMaxSizes(result)
		return result
	}

	if comp.Width.Kind() == style.KindPercent {
		// If we know the percentage resolution size, we could resolve it here.
		// But intrinsic sizes usually shouldn't depend on the parent's resolved size.
		// However, in many engines, percentages are treated as 0 for min-content
		// and the resolved value for max-content if known.
		// For Kite, we'll just fall through to child measurement if it's not KindCells.
	}

	// Otherwise, iterate through children.
	var childrenMinMax MinMaxSizes

	for child := range a.Node.LayoutChildren() {
		childMargin := child.Style().Margin
		childAlgo := &BlockAlgorithm{
			Node: child,
		}
		childMinMax := childAlgo.ComputeMinMaxSizes()
		
		childMinMax = childMinMax.Add(childMargin.Left + childMargin.Right)
		childrenMinMax.Encompass(childMinMax)
	}

	result = childrenMinMax.Add(parentDecorX)

	// 2. Cache and return.
	a.Node.SetCachedMinMaxSizes(result)
	return result
}
