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
func (a *BlockAlgorithm) Layout(ctx *Context) *Fragment {
	// 1. Cache Check: Return cached fragment if constraints match and the node is clean.
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(Block)")()

	comp := a.Node.Style()

	// 1. Initial Scrollbar Decision
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)

	frag, contentHeight := a.layoutInternal(ctx, hasScrollbarX, hasScrollbarY)

	// 2. Check for Auto Scrollbars
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)
	viewport := decor.ViewportSize(frag.Size)

	// Calculate content width from children for horizontal auto-scrollbar detection.
	contentWidth := 0
	for _, child := range frag.Children {
		contentWidth = max(contentWidth, child.Offset.X+child.Fragment.Size.Width-decor.Insets.Left)
	}

	needY := !hasScrollbarY && comp.Scrollbar.Y.UnwrapOr(false) && comp.OverflowY == style.OverflowAuto && contentHeight > viewport.Height
	needX := !hasScrollbarX && comp.Scrollbar.X.UnwrapOr(false) && comp.OverflowX == style.OverflowAuto && contentWidth > viewport.Width

	if needY || needX {
		frag, _ = a.layoutInternal(ctx, hasScrollbarX || needX, hasScrollbarY || needY)
	}

	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

func (a *BlockAlgorithm) layoutInternal(ctx *Context, hasScrollbarX, hasScrollbarY bool) (*Fragment, int) {
	comp := a.Node.Style()
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)

	// 2. Intrinsic Sizing: If inline size is not fixed, use ComputeMinMaxSizes.
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize || comp.Width.Kind() == style.KindMaxContent {
		minMax = a.ComputeMinMaxSizes(ctx)
	}

	// Resolve the resolved inline size (width).
	var resolvedInlineSize int
	if a.Space.IsFixedInlineSize && comp.Width.Kind() != style.KindMaxContent {
		resolvedInlineSize = a.Space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(a.Space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			// Block elements stretch to available width by default if not a shrink-wrap case.
			if comp.Display == style.DisplayBlock || comp.Display == style.DisplayFlex {
				resolvedInlineSize = a.Space.AvailableSize.Width
			} else {
				resolvedInlineSize = min(minMax.Max, a.Space.AvailableSize.Width)
			}
		case style.KindContent:
			// Content (shrink-wrap, capped at available width).
			resolvedInlineSize = min(minMax.Max, a.Space.AvailableSize.Width)
		case style.KindMaxContent:
			// Unconstrained max-content width: the element grows as wide as its
			// content requires, regardless of the available space. Used by UA inner
			// elements that must overflow a clip container for programmatic scroll.
			resolvedInlineSize = minMax.Max
		default:
			// Block elements stretch to available width by default.
			resolvedInlineSize = a.Space.AvailableSize.Width
		}
	}
	resolvedInlineSize = max(resolvedInlineSize, decor.Insets.Left+decor.Insets.Right)

	// 3. Setup Builder: Initialize BoxFragmentBuilder and resolve FragmentGeometry.
	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	builder.SetInlineSize(resolvedInlineSize)
	builder.SetHasScrollbarX(hasScrollbarX)
	builder.SetHasScrollbarY(hasScrollbarY)

	// Internal content boundaries for positioning children.
	contentWidth := max(0, resolvedInlineSize-decor.Insets.Left-decor.Insets.Right)

	builder.SetBlockOffset(decor.Insets.Top)

	// 4. Child Iteration: Loop through in-flow layout children.
	children := a.Node.LayoutChildren()
	nextChild, stop := iter.Pull(children)
	defer stop()

	child, ok := nextChild()
	for ok {
		if child.Style().Display == style.DisplayNone {
			child, ok = nextChild()
			continue
		}

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
				line, ok := breaker.NextLine(ctx)
				if !ok {
					break
				}
				lineFrag := line.ToFragment()
				offset := Point{
					X: decor.Insets.Left,
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
					X: decor.Insets.Left,
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
		childMargin := child.Style().Margin

		// 5. Constraint Generation: delegate to BuildChildSpace (ADR-018).
		// Adjust container height for the space already consumed by previous children.
		containingSpace := Size{Width: resolvedInlineSize, Height: a.Space.AvailableSize.Height}
		containerSpace := Size{Width: contentWidth, Height: max(0, a.Space.AvailableSize.Height-decor.Insets.Top-decor.Insets.Bottom)}
		adjustedContainer := Size{
			Width:  containerSpace.Width,
			Height: max(0, containerSpace.Height-(builder.CurrentBlockOffset()-decor.Insets.Top)),
		}
		childSpace := BuildChildSpace(child, adjustedContainer, containingSpace, a.Space)
		childAlgo := NewAlgorithm(child, childSpace)
		childFrag := childAlgo.Layout(ctx)

		offset := Point{
			X: decor.Insets.Left + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)

		child, ok = nextChild()
	}

	contentHeight := builder.CurrentBlockOffset() - decor.Insets.Top

	// Final block size includes bottom decorations.
	builder.AdvanceBlockOffset(decor.Insets.Bottom)

	// If height is fixed, use that instead.
	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		var resolvedHeight int
		switch comp.Height.Kind() {
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindPercent:
			resolvedHeight = int(float32(a.Space.ContainerSpace.Height) * comp.Height.PercentValue() / 100.0)
			resolvedHeight = max(resolvedHeight, builder.CurrentBlockOffset())
		default:
			resolvedHeight = builder.CurrentBlockOffset()
		}
		builder.SetBlockSize(resolvedHeight)
	}

	// 7. Finalization: Invoke ToFragment() to seal the immutable fragment.
	return builder.ToFragment(), contentHeight
}

// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
func (a *BlockAlgorithm) ComputeMinMaxSizes(ctx *Context) MinMaxSizes {
	// 1. Cache Check.
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}
	defer ctx.Begin("Layout(Block):ComputeMinMaxSizes")()

	comp := a.Node.Style()
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)

	var result MinMaxSizes

	// If width is explicitly set, min and max are that width.
	if comp.Width.Kind() == style.KindCells {
		val := comp.Width.CellsValue()
		result = MinMaxSizes{Min: val, Max: val}
		a.Node.SetCachedMinMaxSizes(result)
		return result
	}

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

			inlineMinMax := ComputeInlineMinMaxSizes(ctx, inlineBuilder.items)
			childrenMinMax.Encompass(inlineMinMax)

			if !ok {
				break
			}
			continue
		}

		childMargin := child.Style().Margin
		childMinMax := IntrinsicMinMaxSizes(ctx, child)

		childMinMax = childMinMax.Add(childMargin.Left + childMargin.Right)
		childrenMinMax.Encompass(childMinMax)

		child, ok = nextChild()
	}

	result = childrenMinMax.Add(decor.Insets.Left + decor.Insets.Right)

	// 2. Cache and return.
	a.Node.SetCachedMinMaxSizes(result)
	return result
}
