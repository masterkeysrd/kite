package layout

import (
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// BlockAlgorithm implements the Block formatting context layout.
// It stacks children vertically (in the block-flow direction) and calculates
// its own intrinsic size based on the accumulated height of its children.
type BlockAlgorithm struct{}

// Layout executes the block layout algorithm and returns an immutable Fragment.
func (a *BlockAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	// 1. Cache Check: Return cached fragment if constraints match and the node is clean.
	if cached := node.CachedLayout(space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(Block)")()

	comp := node.Style()

	// 1. Initial Scrollbar Decision
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)

	frag, contentHeight := a.layoutInternal(ctx, node, space, hasScrollbarX, hasScrollbarY)

	// 2. Check for Auto Scrollbars
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)
	viewport := decor.ViewportSize(frag.Size)

	// Calculate content width from children for horizontal auto-scrollbar detection.
	contentWidth := 0
	for _, child := range frag.Children {
		contentWidth = max(contentWidth, child.Offset.X+child.Fragment.Size.Width-decor.Insets.Left)
	}

	needY := !hasScrollbarY && comp.Scrollbar.Y.UnwrapOr(false) && comp.OverflowY == style.OverflowAuto && contentHeight > viewport.Height
	needX := !hasScrollbarX && comp.Scrollbar.X.UnwrapOr(false) && comp.OverflowX == style.OverflowAuto && contentWidth > viewport.Width

	if needY || needX {
		frag, _ = a.layoutInternal(ctx, node, space, hasScrollbarX || needX, hasScrollbarY || needY)
	}

	node.SetCachedLayout(space, frag)
	return frag
}

func (a *BlockAlgorithm) layoutInternal(ctx *Context, node Node, space ConstraintSpace, hasScrollbarX, hasScrollbarY bool) (*Fragment, int) {
	comp := node.Style()
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)

	// 2. Intrinsic Sizing: If inline size is not fixed, use ComputeMinMaxSizes.
	var minMax MinMaxSizes
	if !space.IsFixedInlineSize || comp.Width.Kind() == style.KindMaxContent {
		minMax = a.ComputeMinMaxSizes(ctx, node)
	}

	// Resolve the resolved inline size (width).
	var resolvedInlineSize int
	if space.IsFixedInlineSize && comp.Width.Kind() != style.KindMaxContent {
		resolvedInlineSize = space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			// Block elements stretch to available width by default if not a shrink-wrap case.
			if comp.Display == style.DisplayBlock || comp.Display == style.DisplayFlex {
				resolvedInlineSize = space.AvailableSize.Width
			} else {
				resolvedInlineSize = min(minMax.Max, space.AvailableSize.Width)
			}
		case style.KindContent:
			// Content (shrink-wrap, capped at available width).
			resolvedInlineSize = min(minMax.Max, space.AvailableSize.Width)
		case style.KindMaxContent:
			// Unconstrained max-content width: the element grows as wide as its
			// content requires, regardless of the available space. Used by UA inner
			// elements that must overflow a clip container for programmatic scroll.
			resolvedInlineSize = minMax.Max
		default:
			// Block elements stretch to available width by default.
			resolvedInlineSize = space.AvailableSize.Width
		}
	}
	resolvedInlineSize = max(resolvedInlineSize, decor.Insets.Left+decor.Insets.Right)
	if !space.IsFixedInlineSize {
		resolvedInlineSize = ClampWidth(node, resolvedInlineSize, space)
	}

	// 3. Setup Builder: Initialize BoxFragmentBuilder and resolve FragmentGeometry.
	builder := AcquireBoxFragmentBuilder(node, space)
	builder.SetInlineSize(resolvedInlineSize)
	builder.SetHasScrollbarX(hasScrollbarX)
	builder.SetHasScrollbarY(hasScrollbarY)

	// Internal content boundaries for positioning children.
	contentWidth := max(0, resolvedInlineSize-decor.Insets.Left-decor.Insets.Right)

	builder.SetBlockOffset(decor.Insets.Top)

	// 4. Child Iteration: Loop through in-flow layout children.
	var inlineBuilder *InlineItemsBuilder
	defer func() {
		if inlineBuilder != nil {
			ReleaseInlineItemsBuilder(inlineBuilder)
		}
	}()
	var bufferedInlines []Node
	if ctx != nil {
		bufferedInlines = ctx.InlineBuffer[:0]
	}

	processInlines := func() {
		if len(bufferedInlines) == 0 {
			return
		}

		if inlineBuilder == nil {
			inlineBuilder = AcquireInlineItemsBuilder(defaultShaper, node)
		}

		inlineBuilder.Reset()
		for _, child := range bufferedInlines {
			inlineBuilder.collect(child)
		}
		bufferedInlines = bufferedInlines[:0]
		if ctx != nil {
			ctx.InlineBuffer = bufferedInlines
		}

		items := inlineBuilder.items
		blockStyle := node.Style()
		breaker := AcquireLineBreaker(items, contentWidth, blockStyle.TextAlign, blockStyle.AlignItems)
		linesEmitted := 0
		for {
			line, ok := breaker.NextLine(ctx)
			if !ok {
				break
			}
			lineFrag := line.ToFragment()
			offset := geometry.Point{
				X: decor.Insets.Left,
				Y: builder.CurrentBlockOffset(),
			}
			builder.AddChild(lineFrag, offset)
			builder.AdvanceBlockOffset(lineFrag.Size.Height)
			linesEmitted++
		}
		ReleaseLineBreaker(breaker)
		// An IFC with no visible text (e.g. an empty input) still occupies
		// one content row — just like a browser renders an empty line box.
		if linesEmitted == 0 {
			line := lineBoxPool.Get().(*LineBox)
			line.Size = geometry.Size{Width: 0, Height: 1}
			line.Children = line.Children[:0]
			lineFrag := line.ToFragment()
			offset := geometry.Point{
				X: decor.Insets.Left,
				Y: builder.CurrentBlockOffset(),
			}
			builder.AddChild(lineFrag, offset)
			builder.AdvanceBlockOffset(1)
		}
	}

	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		if child.Style().Display == style.DisplayNone {
			continue
		}

		if IsInlineLevel(child) {
			bufferedInlines = append(bufferedInlines, child)
			if ctx != nil {
				ctx.InlineBuffer = bufferedInlines
			}
			continue
		}

		// Block child encountered: flush buffered inlines first.
		processInlines()

		// Standard Block Child Layout
		childMargin := child.Style().Margin

		// 5. Constraint Generation: delegate to BuildChildSpace (ADR-018).
		// Adjust container height for the space already consumed by previous children.
		containingSpace := geometry.Size{Width: resolvedInlineSize, Height: space.AvailableSize.Height}
		containerSpace := geometry.Size{Width: contentWidth, Height: max(0, space.AvailableSize.Height-decor.Insets.Top-decor.Insets.Bottom)}
		adjustedContainer := geometry.Size{
			Width:  containerSpace.Width,
			Height: max(0, containerSpace.Height-(builder.CurrentBlockOffset()-decor.Insets.Top)),
		}
		childSpace := BuildChildSpace(child, adjustedContainer, containingSpace, space)
		childAlgo := GetAlgorithm(child)
		childFrag := childAlgo.Layout(ctx, child, childSpace)

		offset := geometry.Point{
			X: decor.Insets.Left + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)
	}

	// Final flush
	processInlines()

	contentHeight := builder.CurrentBlockOffset() - decor.Insets.Top

	// Final block size includes bottom decorations.
	builder.AdvanceBlockOffset(decor.Insets.Bottom)

	// If height is fixed, use that instead.
	if space.IsFixedBlockSize {
		builder.SetBlockSize(space.AvailableSize.Height)
	} else {
		var resolvedHeight int
		switch comp.Height.Kind() {
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindPercent:
			if space.ContainerSpace.Height < InfiniteBlockSize {
				resolvedHeight = int(float32(space.ContainerSpace.Height) * comp.Height.PercentValue() / 100.0)
			} else {
				resolvedHeight = builder.CurrentBlockOffset()
			}
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
func (a *BlockAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	// 1. Cache Check.
	if sizes, ok := node.CachedMinMaxSizes(); ok {
		return sizes
	}
	defer ctx.Begin("Layout(Block):ComputeMinMaxSizes")()

	comp := node.Style()
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)

	var result MinMaxSizes

	// If width is explicitly set, min and max are that width.
	if comp.Width.Kind() == style.KindCells {
		val := comp.Width.CellsValue()
		result = MinMaxSizes{Min: val, Max: val}
		node.SetCachedMinMaxSizes(result)
		return result
	}

	// Otherwise, iterate through children.
	var childrenMinMax MinMaxSizes

	var inlineBuilder *InlineItemsBuilder
	defer func() {
		if inlineBuilder != nil {
			ReleaseInlineItemsBuilder(inlineBuilder)
		}
	}()
	var bufferedInlines []Node
	if ctx != nil {
		bufferedInlines = ctx.InlineBuffer[:0]
	}

	processInlines := func() {
		if len(bufferedInlines) == 0 {
			return
		}
		if inlineBuilder == nil {
			inlineBuilder = AcquireInlineItemsBuilder(defaultShaper, node)
		}
		inlineBuilder.Reset()
		for _, child := range bufferedInlines {
			inlineBuilder.collect(child)
		}
		bufferedInlines = bufferedInlines[:0]
		if ctx != nil {
			ctx.InlineBuffer = bufferedInlines
		}

		inlineMinMax := ComputeInlineMinMaxSizes(ctx, inlineBuilder.items)
		childrenMinMax.Encompass(inlineMinMax)
	}

	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		if child.Style().Display == style.DisplayNone {
			continue
		}

		if IsInlineLevel(child) {
			bufferedInlines = append(bufferedInlines, child)
			if ctx != nil {
				ctx.InlineBuffer = bufferedInlines
			}
			continue
		}

		processInlines()

		childMargin := child.Style().Margin
		childMinMax := IntrinsicMinMaxSizes(ctx, child)

		childMinMax = childMinMax.Add(childMargin.Left + childMargin.Right)
		childrenMinMax.Encompass(childMinMax)
	}

	processInlines()

	result = childrenMinMax.Add(decor.Insets.Left + decor.Insets.Right)

	// 2. Cache and return.
	node.SetCachedMinMaxSizes(result)
	return result
}
