package layout

import (
	"slices"

	"github.com/masterkeysrd/kite/dom"
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// FlexAlgorithm implements the CSS Flexbox formatting context layout.
type FlexAlgorithm struct{}

// AnonymousBlock represents an anonymous box created to wrap contiguous runs of inline content.
type AnonymousBlock struct {
	parent        Node
	children      []Node
	cachedSpace   ConstraintSpace
	cachedStyle   style.Computed
	hasStyle      bool
}

var _ Node = (*AnonymousBlock)(nil)

func (a *AnonymousBlock) Style() *style.Computed {
	if !a.hasStyle {
		// Anonymous boxes inherit styles from their parent and have DisplayBlock.
		s := *a.parent.Style()
		s.Display = style.DisplayBlock
		// Reset margins/padding/border for anonymous box to ensure they don't double up
		s.Margin = style.EdgeValues[int]{}
		s.Padding = style.EdgeValues[int]{}
		s.Border = style.Border{}
		// Use Width: Content to ensure the anonymous box shrink-wraps its inlines,
		// allowing the flex container to correctly align/center it.
		s.Width = style.Content
		s.Height = style.Auto
		a.cachedStyle = s
		a.hasStyle = true
	}
	return &a.cachedStyle
}

func (a *AnonymousBlock) FirstLayoutChild() Node {
	if len(a.children) == 0 {
		return nil
	}
	return a.children[0]
}

func (a *AnonymousBlock) NextLayoutSibling(child Node) Node {
	for i, c := range a.children {
		if c == child {
			if i+1 < len(a.children) {
				return a.children[i+1]
			}
			break
		}
	}
	return nil
}

func (a *AnonymousBlock) LogicalNode() dom.Node    { return nil }
func (a *AnonymousBlock) IsDirtyLayout() bool      { return true }
func (a *AnonymousBlock) IsDirtyPaint() bool       { return true }
func (a *AnonymousBlock) HasChildNeedsPaint() bool { return true }
func (a *AnonymousBlock) ClearDirtyLayout()        {}
func (a *AnonymousBlock) Fragment() *Fragment      { return nil }

func (a *AnonymousBlock) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *AnonymousBlock) Layout(ctx *Context) *Fragment {
	// AnonymousBlock uses BlockAlgorithm to layout its IFC.
	return blockAlgo.Layout(ctx, a, a.cachedSpace)
}

func (a *AnonymousBlock) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *AnonymousBlock) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *AnonymousBlock) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *AnonymousBlock) SetOffset(p geometry.Point) {}

func (a *AnonymousBlock) IsAnonymous() bool {
	return true
}

func (a *AnonymousBlock) ComputeMinMaxSizes(ctx *Context) MinMaxSizes {
	// BlockAlgorithm's ComputeMinMaxSizes logic for IFC:
	inlineBuilder := AcquireInlineItemsBuilder(defaultShaper, a)
	defer ReleaseInlineItemsBuilder(inlineBuilder)
	for child := a.FirstLayoutChild(); child != nil; child = a.NextLayoutSibling(child) {
		inlineBuilder.collect(child)
	}
	return ComputeInlineMinMaxSizes(ctx, inlineBuilder.items)
}

// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
func (a *FlexAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	if sizes, ok := node.CachedMinMaxSizes(); ok {
		return sizes
	}
	defer ctx.Begin("Layout(Flex):ComputeMinMaxSizes")()

	comp := node.Style()
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)

	gap := comp.Gap

	if comp.Width.Kind() == style.KindCells {
		val := comp.Width.CellsValue()
		result := MinMaxSizes{Min: val, Max: val}
		node.SetCachedMinMaxSizes(result)
		return result
	}

	var result MinMaxSizes
	isRow := comp.FlexDirection == style.FlexRow || comp.FlexDirection == style.FlexRowReverse
	geom := flexGeometry{direction: comp.FlexDirection}

	items := a.collectItems(ctx, node, ConstraintSpace{}, geom)

	if isRow {
		// Row: Max = sum(child.max) + gaps.
		// Min depends on wrapping: if wrapping is on, Min is max(child.min).
		// If wrapping is off, Min is sum(child.min) + gaps.
		totalMin := 0
		maxChildMin := 0
		totalMax := 0
		count := 0
		for _, item := range items {
			childMinMax := IntrinsicMinMaxSizes(ctx, item.Node)
			childMargin := item.Node.Style().Margin
			childMin := childMinMax.Min + childMargin.Left + childMargin.Right
			childMax := childMinMax.Max + childMargin.Left + childMargin.Right

			totalMin += childMin
			maxChildMin = max(maxChildMin, childMin)
			totalMax += childMax
			count++
		}
		if count > 1 {
			totalMin += gap.Column * (count - 1)
			totalMax += gap.Column * (count - 1)
		}

		if comp.FlexWrap == style.FlexWrapOn {
			result = MinMaxSizes{Min: maxChildMin, Max: totalMax}
		} else {
			result = MinMaxSizes{Min: totalMin, Max: totalMax}
		}
	} else {
		// Column: Min = max(child.min), Max = max(child.max)
		maxMin := 0
		maxMax := 0
		for _, item := range items {
			childMinMax := IntrinsicMinMaxSizes(ctx, item.Node)
			childMargin := item.Node.Style().Margin
			maxMin = max(maxMin, childMinMax.Min+childMargin.Left+childMargin.Right)
			maxMax = max(maxMax, childMinMax.Max+childMargin.Left+childMargin.Right)
		}
		result = MinMaxSizes{Min: maxMin, Max: maxMax}
	}

	result = result.Add(decor.Insets.Left + decor.Insets.Right)
	node.SetCachedMinMaxSizes(result)
	return result
}

// Layout executes the flex layout algorithm and returns an immutable Fragment.
func (a *FlexAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	if cached := node.CachedLayout(space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(Flex)")()

	comp := node.Style()

	// 1. Initial Scrollbar Decision
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)

	frag, contentSize := a.layoutInternal(ctx, node, space, hasScrollbarX, hasScrollbarY)

	// 2. Check for Auto Scrollbars
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)
	viewport := decor.ViewportSize(frag.Size)

	needY := !hasScrollbarY && comp.Scrollbar.Y.UnwrapOr(false) && comp.OverflowY == style.OverflowAuto && contentSize.Height > viewport.Height
	needX := !hasScrollbarX && comp.Scrollbar.X.UnwrapOr(false) && comp.OverflowX == style.OverflowAuto && contentSize.Width > viewport.Width

	if needY || needX {
		frag, _ = a.layoutInternal(ctx, node, space, hasScrollbarX || needX, hasScrollbarY || needY)
	}

	node.SetCachedLayout(space, frag)
	return frag
}

func (a *FlexAlgorithm) layoutInternal(ctx *Context, node Node, space ConstraintSpace, hasScrollbarX, hasScrollbarY bool) (*Fragment, geometry.Size) {
	comp := node.Style()
	geom := flexGeometry{direction: comp.FlexDirection}
	decor := ResolveDecorations(node, hasScrollbarX, hasScrollbarY)

	// 1. Resolve Container Main/Cross Sizes
	var minMax MinMaxSizes
	if !space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes(ctx, node)
	}

	resolvedWidth := space.AvailableSize.Width
	if !space.IsFixedInlineSize {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedWidth = int(float32(space.ContainerSpace.Width)*comp.Width.PercentValue()/100.0 + 0.5)
		case style.KindCells:
			resolvedWidth = comp.Width.CellsValue()
		case style.KindContent:
			resolvedWidth = min(minMax.Max, space.AvailableSize.Width)
		case style.KindAuto:
			if comp.Display == style.DisplayInlineFlex {
				resolvedWidth = min(minMax.Max, space.AvailableSize.Width)
			} else {
				resolvedWidth = space.AvailableSize.Width
			}
		}
		resolvedWidth = ClampWidth(node, resolvedWidth, space)
	}
	resolvedWidth = max(resolvedWidth, decor.Insets.Left+decor.Insets.Right)

	resolvedHeight := space.AvailableSize.Height
	if !space.IsFixedBlockSize {
		switch comp.Height.Kind() {
		case style.KindPercent:
			if space.ContainerSpace.Height < InfiniteBlockSize {
				resolvedHeight = int(float32(space.ContainerSpace.Height)*comp.Height.PercentValue()/100.0 + 0.5)
			}
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindAuto:
			// Resolve this later from lines if it's auto.
		case style.KindContent:
			// TODO: Resolve from content
		}
		resolvedHeight = ClampHeight(node, resolvedHeight, space)
	}

	contentWidth := max(0, resolvedWidth-decor.Insets.Left-decor.Insets.Right)
	contentHeight := max(0, resolvedHeight-decor.Insets.Top-decor.Insets.Bottom)

	contentMainSize := contentWidth
	contentCrossSizeForItems := contentHeight
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		contentMainSize = contentHeight
		contentCrossSizeForItems = contentWidth
	}

	// 2. Prepare Items & Instantiate Builder
	mainGap := comp.Gap.Column
	crossGap := comp.Gap.Row
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		mainGap = comp.Gap.Row
		crossGap = comp.Gap.Column
	}

	builder := AcquireFlexLineBuilder(geom, mainGap, crossGap)
	defer ReleaseFlexLineBuilder(builder)

	items := a.collectItems(ctx, node, space, geom)

	for _, item := range items {
		childStyle := item.Node.Style()
		childMargin := childStyle.Margin

		var baseSize, minSize, maxSize int

		// 1. Resolve Flex Basis (Main geometry.Size)
		basis := childStyle.Flex.Basis
		if basis.Kind() == style.KindCells {
			baseSize = basis.CellsValue()
		} else if basis.Kind() == style.KindPercent && contentMainSize < InfiniteBlockSize {
			baseSize = int(float32(contentMainSize)*basis.PercentValue()/100.0 + 0.5)
		} else if basis.Kind() == style.KindContent {
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				baseSize = IntrinsicMinMaxSizes(ctx, item.Node).Max
			} else {
				baseSize = IntrinsicBlockSize(ctx, item.Node, contentCrossSizeForItems)
			}
		} else {
			// Auto: Use width/height property
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				if childStyle.Width.Kind() == style.KindCells {
					baseSize = childStyle.Width.CellsValue()
				} else if childStyle.Width.Kind() == style.KindPercent && contentMainSize < InfiniteBlockSize {
					baseSize = int(float32(contentMainSize)*childStyle.Width.PercentValue()/100.0 + 0.5)
				} else {
					baseSize = IntrinsicMinMaxSizes(ctx, item.Node).Max
				}
			} else {
				if childStyle.Height.Kind() == style.KindCells {
					baseSize = childStyle.Height.CellsValue()
				} else if childStyle.Height.Kind() == style.KindPercent && contentMainSize < InfiniteBlockSize {
					baseSize = int(float32(contentMainSize)*childStyle.Height.PercentValue()/100.0 + 0.5)
				} else {
					probeWidth := contentCrossSizeForItems
					if comp.AlignItems != style.AlignStretch && childStyle.Width.Kind() == style.KindAuto {
						probeWidth = min(IntrinsicMinMaxSizes(ctx, item.Node).Max, contentCrossSizeForItems)
					}
					baseSize = IntrinsicBlockSize(ctx, item.Node, probeWidth)
				}
			}
		}

		// 2. Resolve Min/Max Main Sizes
		if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
			baseSize += childMargin.Left + childMargin.Right

			// Default min-size is min-content (CSS flexbox spec: min-width: auto)
			if childStyle.MinWidth.Kind() == style.KindAuto {
				if childStyle.OverflowX != style.OverflowVisible {
					minSize = childMargin.Left + childMargin.Right
				} else {
					minSize = IntrinsicMinMaxSizes(ctx, item.Node).Min + childMargin.Left + childMargin.Right
				}
			} else if childStyle.MinWidth.Kind() == style.KindCells {
				minSize = childStyle.MinWidth.CellsValue() + childMargin.Left + childMargin.Right
			}

			if childStyle.MaxWidth.Kind() == style.KindCells {
				maxSize = childStyle.MaxWidth.CellsValue() + childMargin.Left + childMargin.Right
			}
		} else {
			baseSize += childMargin.Top + childMargin.Bottom

			// Default min-size is min-content (CSS flexbox spec: min-height: auto)
			if childStyle.MinHeight.Kind() == style.KindAuto {
				if childStyle.OverflowY != style.OverflowVisible {
					minSize = childMargin.Top + childMargin.Bottom
				} else {
					minSize = IntrinsicBlockSize(ctx, item.Node, contentCrossSizeForItems) + childMargin.Top + childMargin.Bottom
				}
			} else if childStyle.MinHeight.Kind() == style.KindCells {
				minSize = childStyle.MinHeight.CellsValue() + childMargin.Top + childMargin.Bottom
			}

			if childStyle.MaxHeight.Kind() == style.KindCells {
				maxSize = childStyle.MaxHeight.CellsValue() + childMargin.Top + childMargin.Bottom
			}
		}

		childAlign := comp.AlignItems
		if childStyle.AlignSelf != style.AlignAuto {
			childAlign = childStyle.AlignSelf
		}
		builder.AddItem(item.Node, baseSize, minSize, maxSize, item.Grow, item.Shrink, item.Order)
		builder.items[len(builder.items)-1].AlignSelf = childAlign
	}

	// 3. Line Breaking
	wrap := comp.FlexWrap == style.FlexWrapOn
	builder.ComputeLines(contentMainSize, wrap)

	// 4. Resolve Main Sizes (Grow/Shrink)
	isMainDefinite := true
	if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
		if !space.IsFixedInlineSize {
			isContentOrInlineFlexAuto := comp.Width.Kind() == style.KindContent ||
				(comp.Width.Kind() == style.KindAuto && comp.Display == style.DisplayInlineFlex)
			if isContentOrInlineFlexAuto {
				isMainDefinite = false
			}
		}
	} else {
		if !space.IsFixedBlockSize {
			isAutoOrContent := comp.Height.Kind() == style.KindAuto || comp.Height.Kind() == style.KindContent
			isIndefinitePercent := comp.Height.Kind() == style.KindPercent && space.ContainerSpace.Height >= InfiniteBlockSize
			if isAutoOrContent || isIndefinitePercent {
				isMainDefinite = false
			}
		}
	}

	if isMainDefinite && contentMainSize < InfiniteBlockSize {
		for i := range builder.Lines() {
			builder.ResolveFlexibleLengths(i, contentMainSize)
		}
	} else {
		for _, line := range builder.Lines() {
			for _, item := range line.Items {
				item.MainSize = item.HypotheticalMainSize
			}
		}
	}

	// 5. Measure Final Dimensions
	totalMaxLineMain := 0
	totalSumLineCross := 0
	lines := builder.Lines()

	for i, line := range lines {
		lineCrossSize := 0
		lineMainSize := 0
		for j, item := range line.Items {
			childStyle := item.Node.Style()
			childMargin := childStyle.Margin
			childMainSize := item.MainSize
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				childMainSize -= childMargin.Left + childMargin.Right
			} else {
				childMainSize -= childMargin.Top + childMargin.Bottom
			}

			measureCrossSize := contentCrossSizeForItems
			var childSpace ConstraintSpace
			flexContainingSpace := geometry.Size{Width: resolvedWidth, Height: resolvedHeight}
			flexContainerSpace := geometry.Size{Width: contentWidth, Height: contentHeight}

			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				isFixedInline := true
				isFixedBlock := false

				if !wrap && item.AlignSelf == style.AlignStretch && childStyle.Height.Kind() == style.KindAuto && space.IsFixedBlockSize {
					isFixedBlock = true
				}

				childSpace = ConstraintSpace{
					AvailableSize:     geometry.Size{Width: childMainSize, Height: measureCrossSize},
					ContainingSpace:   flexContainingSpace,
					ContainerSpace:    flexContainerSpace,
					IsFixedInlineSize: isFixedInline,
					IsFixedBlockSize:  isFixedBlock,
				}
			} else {
				isFixedInline := false
				isFixedBlock := true

				if !wrap && item.AlignSelf == style.AlignStretch && childStyle.Width.Kind() == style.KindAuto && space.IsFixedInlineSize {
					isFixedInline = true
				} else if item.AlignSelf != style.AlignStretch && childStyle.Width.Kind() == style.KindAuto {
					measureCrossSize = min(IntrinsicMinMaxSizes(ctx, item.Node).Max, contentCrossSizeForItems)
				}

				childSpace = ConstraintSpace{
					AvailableSize:     geometry.Size{Width: measureCrossSize, Height: childMainSize},
					ContainingSpace:   flexContainingSpace,
					ContainerSpace:    flexContainerSpace,
					IsFixedInlineSize: isFixedInline,
					IsFixedBlockSize:  isFixedBlock,
				}
			}

			childAlgo := GetAlgorithm(item.Node)
			item.Fragment = childAlgo.Layout(ctx, item.Node, childSpace)

			item.MainSize = geom.MainSize(item.Fragment.Size)
			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				item.MainSize += childMargin.Top + childMargin.Bottom
			} else {
				item.MainSize += childMargin.Left + childMargin.Right
			}

			itemCrossSize := geom.CrossSize(item.Fragment.Size)
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				itemCrossSize += childMargin.Top + childMargin.Bottom
			} else {
				itemCrossSize += childMargin.Left + childMargin.Right
			}
			item.CrossSize = itemCrossSize
			lineCrossSize = max(lineCrossSize, item.CrossSize)

			lineMainSize += item.MainSize
			if j > 0 {
				lineMainSize += mainGap
			}
		}
		line.CrossSize = lineCrossSize
		line.MainSize = lineMainSize

		totalSumLineCross += line.CrossSize
		if i > 0 {
			totalSumLineCross += crossGap
		}
		totalMaxLineMain = max(totalMaxLineMain, line.MainSize)
	}

	// Resolve the final physical dimensions.
	if !space.IsFixedInlineSize {
		if comp.Width.Kind() == style.KindAuto {
			if comp.Display == style.DisplayInlineFlex {
				var logicalWidth int
				if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
					logicalWidth = totalSumLineCross
				} else {
					logicalWidth = totalMaxLineMain
				}
				resolvedWidth = min(logicalWidth+decor.Insets.Left+decor.Insets.Right, space.AvailableSize.Width)
			} else {
				resolvedWidth = space.AvailableSize.Width
			}
		} else if comp.Width.Kind() == style.KindContent {
			var logicalWidth int
			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				logicalWidth = totalSumLineCross
			} else {
				logicalWidth = totalMaxLineMain
			}
			resolvedWidth = min(logicalWidth+decor.Insets.Left+decor.Insets.Right, space.AvailableSize.Width)
		}
	}

	if !space.IsFixedBlockSize {
		isIndefinitePercent := comp.Height.Kind() == style.KindPercent && space.ContainerSpace.Height >= InfiniteBlockSize
		if comp.Height.Kind() == style.KindAuto || comp.Height.Kind() == style.KindContent || isIndefinitePercent {
			var logicalHeight int
			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				logicalHeight = totalMaxLineMain
			} else {
				logicalHeight = totalSumLineCross
			}
			resolvedHeight = logicalHeight + decor.Insets.Top + decor.Insets.Bottom
		}
	}

	if !space.IsFixedInlineSize {
		resolvedWidth = ClampWidth(node, resolvedWidth, space)
	}
	if !space.IsFixedBlockSize {
		resolvedHeight = ClampHeight(node, resolvedHeight, space)
	}

	containerSize := geometry.Size{Width: resolvedWidth, Height: resolvedHeight}
	contentCrossSizeForItems = geom.CrossSize(containerSize) - (decor.Insets.Top + decor.Insets.Bottom)
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		contentCrossSizeForItems = geom.CrossSize(containerSize) - (decor.Insets.Left + decor.Insets.Right)
	}

	// 6. Alignment & Positioning
	builder.AlignCrossAxis(contentCrossSizeForItems, comp.AlignContent, comp.AlignItems)

	// Second pass for stretching items in wrapping containers or when line cross size changed
	flexContainingSpace := geometry.Size{Width: resolvedWidth, Height: resolvedHeight}
	flexContainerSpace := geometry.Size{Width: contentWidth, Height: contentHeight}

	hasStretch := false
	for _, line := range builder.Lines() {
		for _, item := range line.Items {
			childStyle := item.Node.Style()
			if item.AlignSelf == style.AlignStretch {
				isRow := geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse
				isAuto := (isRow && childStyle.Height.Kind() == style.KindAuto) || (!isRow && childStyle.Width.Kind() == style.KindAuto)
				if isAuto {
					childMargin := childStyle.Margin
					targetCross := line.CrossSize
					if isRow {
						targetCross -= childMargin.Top + childMargin.Bottom
					} else {
						targetCross -= childMargin.Left + childMargin.Right
					}

					if geom.CrossSize(item.Fragment.Size) != targetCross {
						hasStretch = true
						measureMain := geom.MainSize(item.Fragment.Size)
						childSpace := ConstraintSpace{
							AvailableSize:     geom.MakeSize(measureMain, targetCross),
							ContainingSpace:   flexContainingSpace,
							ContainerSpace:    flexContainerSpace,
							IsFixedInlineSize: true,
							IsFixedBlockSize:  true,
						}
						item.Fragment = GetAlgorithm(item.Node).Layout(ctx, item.Node, childSpace)
					}
				}
			}
		}
	}

	if hasStretch {
		builder.AlignCrossAxis(contentCrossSizeForItems, comp.AlignContent, comp.AlignItems)
	}

	isReverse := geom.direction == style.FlexRowReverse || geom.direction == style.FlexColumnReverse
	for i := range builder.Lines() {
		builder.AlignLine(i, contentMainSize, comp.JustifyContent, isReverse)
	}

	// Support block fragmentation
	var breakToken *BreakToken
	extraCross := contentCrossSizeForItems - totalSumLineCross

	if extraCross < 0 && space.IsFixedBlockSize {
		currentTotalCross := 0
		breakLineIndex := -1
		itemsToSkip := 0
		if space.BreakToken != nil {
			itemsToSkip = space.BreakToken.ChildIndex
		}

		for i, line := range lines {
			lineHeightWithGap := line.CrossSize
			if i > 0 {
				lineHeightWithGap += crossGap
			}

			if currentTotalCross+lineHeightWithGap > contentCrossSizeForItems {
				breakLineIndex = i
				break
			}
			currentTotalCross += lineHeightWithGap
			itemsToSkip += len(line.Items)
		}

		if breakLineIndex != -1 && breakLineIndex > 0 {
			breakToken = &BreakToken{
				Node:       node,
				ChildIndex: itemsToSkip,
			}
			lines = lines[:breakLineIndex]
		}
	}

	fragBuilder := AcquireBoxFragmentBuilder(node, space)
	fragBuilder.SetInlineSize(resolvedWidth)
	fragBuilder.SetBlockSize(resolvedHeight)
	fragBuilder.SetBreakToken(breakToken)
	fragBuilder.SetHasScrollbarX(hasScrollbarX)
	fragBuilder.SetHasScrollbarY(hasScrollbarY)

	// Add children to fragment builder using resolved offsets
	for _, line := range lines {
		for _, item := range line.Items {
			offset := geometry.Point{
				X: decor.Insets.Left + item.Offset.X,
				Y: decor.Insets.Top + item.Offset.Y,
			}
			fragBuilder.AddChild(item.Fragment, offset)
		}
	}

	contentSize := geom.MakeSize(totalMaxLineMain, totalSumLineCross)
	return fragBuilder.ToFragment(), contentSize
}

func (a *FlexAlgorithm) isInlineLevel(node Node) bool {
	_, ok := node.(InlineLever)
	return ok
}

func (a *FlexAlgorithm) collectItems(ctx *Context, node Node, space ConstraintSpace, geom flexGeometry) []*FlexItem {
	var allItems []*FlexItem

	// Refactored to avoid iter.Pull and closures, reusing context buffer
	var bufferedInlines []Node
	if ctx != nil {
		bufferedInlines = ctx.InlineBuffer[:0]
	}

	processInlines := func() {
		if len(bufferedInlines) == 0 {
			return
		}
		anon := &AnonymousBlock{
			parent:   node,
			children: slices.Clone(bufferedInlines),
		}
		bufferedInlines = bufferedInlines[:0]
		if ctx != nil {
			ctx.InlineBuffer = bufferedInlines
		}

		childStyle := anon.Style()
		item := &FlexItem{
			Node:   anon,
			Grow:   childStyle.Flex.Grow,
			Shrink: childStyle.Flex.Shrink,
		}
		allItems = append(allItems, item)
	}

	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		if child.Style().Display == style.DisplayNone {
			continue
		}

		if a.isInlineLevel(child) {
			bufferedInlines = append(bufferedInlines, child)
			if ctx != nil {
				ctx.InlineBuffer = bufferedInlines
			}
			continue
		}

		processInlines()

		childStyle := child.Style()
		item := &FlexItem{
			Node:   child,
			Grow:   childStyle.Flex.Grow,
			Shrink: childStyle.Flex.Shrink,
			Order:  childStyle.Order,
		}
		allItems = append(allItems, item)
	}

	processInlines()

	// Sort by order
	slices.SortFunc(allItems, func(a, b *FlexItem) int {
		return a.Order - b.Order
	})

	// Handle reverse directions
	if geom.direction == style.FlexRowReverse || geom.direction == style.FlexColumnReverse {
		for i, j := 0, len(allItems)-1; i < j; i, j = i+1, j-1 {
			allItems[i], allItems[j] = allItems[j], allItems[i]
		}
	}

	startIndex := 0
	if space.BreakToken != nil {
		startIndex = space.BreakToken.ChildIndex
	}

	if startIndex >= len(allItems) {
		return nil
	}

	return allItems[startIndex:]
}
