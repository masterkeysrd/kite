package layout

import (
	"iter"

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

	// fmt.Printf("DEBUG: Layout node kind=%v\n", a.Node.Style().Display)

	comp := a.Node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding

	// 2. Intrinsic Sizing: If inline size is not fixed, use ComputeMinMaxSizes.
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize || comp.Width.Kind() == style.KindMaxContent {
		minMax = a.ComputeMinMaxSizes()
	}

	// Resolve the resolved inline size (width).
	var resolvedInlineSize int
	if a.Space.IsFixedInlineSize && comp.Width.Kind() != style.KindMaxContent {
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
			// Content (shrink-wrap, capped at available width).
			resolvedInlineSize = min(minMax.Max, a.Space.AvailableSize.Width)
		case style.KindMaxContent:
			// Unconstrained max-content width: the element grows as wide as its
			// content requires, regardless of the available space. Used by UA inner
			// elements that must overflow a clip container for programmatic scroll.
			resolvedInlineSize = minMax.Max
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

	contentWidth := max(0, resolvedInlineSize-parentDecorX)

	// 4. Child Iteration: Loop through in-flow layout children.
	children := a.Node.LayoutChildren()
	nextChild, stop := iter.Pull(children)
	defer stop()

	child, ok := nextChild()
	for ok {
		// fmt.Printf("DEBUG: Child isInline=%v logical=%T\n", a.isInlineLevel(child), child.LogicalNode())
		if IsInlineLevel(child) {
			// Sequence of inline children: group into Anonymous Block (IFC).
			inlineBuilder := NewInlineItemsBuilder(defaultShaper, a.Node)
			inlineBuilder.collect(child)

			for {
				peek, peekOk := nextChild()
				if !peekOk {
					child = nil
					ok = false
					break
				}
				if IsInlineLevel(peek) {
					inlineBuilder.collect(peek)
				} else {
					child = peek
					ok = true
					break
				}
			}

			items := inlineBuilder.items
			blockStyle := a.Node.Style()
			breaker := NewLineBreaker(items, contentWidth, blockStyle.TextAlign, blockStyle.AlignItems)
			linesEmitted := 0
			for {
				line, ok := breaker.NextLine()
				if !ok {
					break
				}
				lineFrag := line.ToFragment()
				offset := Point{
					X: insetX,
					Y: builder.CurrentBlockOffset(),
				}
				builder.AddChild(lineFrag, offset)
				builder.AdvanceBlockOffset(lineFrag.Size.Height)
				linesEmitted++
			}
			// An IFC with no visible text (e.g. an empty input) still occupies
			// one content row — just like a browser renders an empty line box.
			if linesEmitted == 0 {
				line := &LineBox{
					Size: Size{Width: 0, Height: 1},
				}
				lineFrag := line.ToFragment()
				offset := Point{
					X: insetX,
					Y: builder.CurrentBlockOffset(),
				}
				builder.AddChild(lineFrag, offset)
				builder.AdvanceBlockOffset(1)
			}

			if !ok {
				break
			}
			continue // Process current 'child' (first block after inlines)
		}

		// Standard Block Child Layout
		childStyle := child.Style()
		childMargin := childStyle.Margin

		// 5. Constraint Generation: Use ConstraintSpaceBuilder.
		childAvailWidth := max(0, resolvedInlineSize-parentDecorX-childMargin.Left-childMargin.Right)
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-childMargin.Top-childMargin.Bottom-(border.Bottom+padding.Bottom))

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{
			Width:  contentWidth,
			Height: max(0, a.Space.AvailableSize.Height-parentDecorY),
		})

		if childStyle.Width.Kind() == style.KindCells {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = childStyle.Width.CellsValue()
		} else if childStyle.Width.Kind() == style.KindPercent {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = int(float32(contentWidth) * childStyle.Width.PercentValue() / 100.0)
		} else if childStyle.Width.Kind() == style.KindAuto {
			if childStyle.Display != style.DisplayTable {
				childSpaceBuilder.SetIsFixedInlineSize(true)
				childSpaceBuilder.space.AvailableSize.Width = childAvailWidth
			}
		}
		// KindMaxContent: do NOT set IsFixedInlineSize so the child's own block
		// algorithm calls ComputeMinMaxSizes and uses the unconstrained max-content
		// width. Leave AvailableSize.Width as the parent available width for the
		// child's percent-resolution baseline only.

		if childStyle.Height.Kind() == style.KindCells {
			childSpaceBuilder.SetIsFixedBlockSize(true)
			childSpaceBuilder.space.AvailableSize.Height = childStyle.Height.CellsValue()
		} else if childStyle.Height.Kind() == style.KindPercent {
			if a.Space.IsFixedBlockSize {
				childSpaceBuilder.SetIsFixedBlockSize(true)
				childSpaceBuilder.space.AvailableSize.Height = int(float32(a.Space.AvailableSize.Height-parentDecorY) * childStyle.Height.PercentValue() / 100.0)
			}
		}

		childSpace := childSpaceBuilder.ToConstraintSpace()
		childAlgo := NewAlgorithm(child, childSpace)
		childFrag := childAlgo.Layout()

		offset := Point{
			X: insetX + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)

		child, ok = nextChild()
	}

	// Final block size includes bottom decorations.
	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

	// If height is fixed, use that instead.
	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		var resolvedHeight int
		switch comp.Height.Kind() {
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindPercent:
			resolvedHeight = int(float32(a.Space.PercentageResolutionSize.Height) * comp.Height.PercentValue() / 100.0)
		default:
			resolvedHeight = builder.CurrentBlockOffset()
		}
		builder.SetBlockSize(resolvedHeight)
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
	border := comp.Border.Widths()
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

	// If we know the percentage resolution size, we could resolve it here.
	// But intrinsic sizes usually shouldn't depend on the parent's resolved size.
	// However, in many engines, percentages are treated as 0 for min-content
	// and the resolved value for max-content if known.
	// For Kite, we'll just fall through to child measurement if it's not KindCells.
	// if comp.Width.Kind() == style.KindPercent {

	// Otherwise, iterate through children.
	var childrenMinMax MinMaxSizes

	children := a.Node.LayoutChildren()
	nextChild, stop := iter.Pull(children)
	defer stop()

	child, ok := nextChild()
	for ok {
		if IsInlineLevel(child) {
			inlineBuilder := NewInlineItemsBuilder(defaultShaper, a.Node)
			inlineBuilder.collect(child)

			for {
				peek, peekOk := nextChild()
				if !peekOk {
					child = nil
					ok = false
					break
				}
				if IsInlineLevel(peek) {
					inlineBuilder.collect(peek)
				} else {
					child = peek
					ok = true
					break
				}
			}

			inlineMinMax := ComputeInlineMinMaxSizes(inlineBuilder.items)
			childrenMinMax.Encompass(inlineMinMax)

			if !ok {
				break
			}
			continue
		}

		childMargin := child.Style().Margin
		childMinMax := IntrinsicMinMaxSizes(child)

		childMinMax = childMinMax.Add(childMargin.Left + childMargin.Right)
		childrenMinMax.Encompass(childMinMax)

		child, ok = nextChild()
	}

	result = childrenMinMax.Add(parentDecorX)

	// 2. Cache and return.
	a.Node.SetCachedMinMaxSizes(result)
	return result
}
