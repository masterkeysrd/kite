package layout

import (
	"iter"
	"slices"

	"github.com/masterkeysrd/kite/style"
)

// FlexAlgorithm implements the Flexbox formatting context layout.
type FlexAlgorithm struct {
	Node  Node
	Space ConstraintSpace
}

// AnonymousBlock represents an anonymous box created to wrap contiguous runs of inline content.
type AnonymousBlock struct {
	parent      Node
	children    []Node
	cachedSpace ConstraintSpace
}

var _ Node = (*AnonymousBlock)(nil)

func (a *AnonymousBlock) Style() *style.Computed {
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
	return &s
}

func (a *AnonymousBlock) LayoutChildren() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, child := range a.children {
			if !yield(child) {
				return
			}
		}
	}
}

func (a *AnonymousBlock) LogicalNode() any         { return nil }
func (a *AnonymousBlock) IsDirtyLayout() bool      { return true }
func (a *AnonymousBlock) IsDirtyPaint() bool       { return true }
func (a *AnonymousBlock) HasChildNeedsPaint() bool { return true }
func (a *AnonymousBlock) ClearDirtyLayout()        {}
func (a *AnonymousBlock) Fragment() *Fragment      { return nil }

func (a *AnonymousBlock) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *AnonymousBlock) Layout() *Fragment {
	// AnonymousBlock uses BlockAlgorithm to layout its IFC.
	// We need to resolve the algorithm manually because NewAlgorithm would return us (recursive).
	algo := &BlockAlgorithm{
		Node:  a,
		Space: a.cachedSpace,
	}
	return algo.Layout()
}

func (a *AnonymousBlock) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *AnonymousBlock) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *AnonymousBlock) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *AnonymousBlock) SetOffset(p Point) {}

func (a *AnonymousBlock) IsAnonymous() bool {
	return true
}

func (a *AnonymousBlock) ComputeMinMaxSizes() MinMaxSizes {
	// BlockAlgorithm's ComputeMinMaxSizes logic for IFC:
	inlineBuilder := NewInlineItemsBuilder(defaultShaper, a)
	for _, child := range a.children {
		inlineBuilder.collect(child)
	}
	return ComputeInlineMinMaxSizes(inlineBuilder.items)
}

// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
func (a *FlexAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}

	comp := a.Node.Style()
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)

	gap := comp.Gap

	if comp.Width.Kind() == style.KindCells {
		val := comp.Width.CellsValue()
		result := MinMaxSizes{Min: val, Max: val}
		a.Node.SetCachedMinMaxSizes(result)
		return result
	}

	var result MinMaxSizes
	isRow := comp.FlexDirection == style.FlexRow || comp.FlexDirection == style.FlexRowReverse
	geom := flexGeometry{direction: comp.FlexDirection}

	items := a.collectItems(geom)

	if isRow {
		// Row: Min = sum(child.min), Max = sum(child.max) + gaps
		totalMin := 0
		totalMax := 0
		count := 0
		for _, item := range items {
			childMinMax := IntrinsicMinMaxSizes(item.Node)
			childMargin := item.Node.Style().Margin
			totalMin += childMinMax.Min + childMargin.Left + childMargin.Right
			totalMax += childMinMax.Max + childMargin.Left + childMargin.Right
			count++
		}
		if count > 1 {
			totalMin += gap.Column * (count - 1)
			totalMax += gap.Column * (count - 1)
		}
		result = MinMaxSizes{Min: totalMin, Max: totalMax}
	} else {
		// Column: Min = max(child.min), Max = max(child.max)
		maxMin := 0
		maxMax := 0
		for _, item := range items {
			childMinMax := IntrinsicMinMaxSizes(item.Node)
			childMargin := item.Node.Style().Margin
			maxMin = max(maxMin, childMinMax.Min+childMargin.Left+childMargin.Right)
			maxMax = max(maxMax, childMinMax.Max+childMargin.Left+childMargin.Right)
		}
		result = MinMaxSizes{Min: maxMin, Max: maxMax}
	}

	result = result.Add(decor.Insets.Left + decor.Insets.Right)
	a.Node.SetCachedMinMaxSizes(result)
	return result
}

// Layout executes the flex layout algorithm and returns an immutable Fragment.
func (a *FlexAlgorithm) Layout() *Fragment {
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()

	// 1. Initial Scrollbar Decision
	hasScrollbarX, hasScrollbarY := ShouldReserveScrollbar(comp)

	frag, contentSize := a.layoutInternal(hasScrollbarX, hasScrollbarY)

	// 2. Check for Auto Scrollbars
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)
	viewport := decor.ViewportSize(frag.Size)

	needY := !hasScrollbarY && comp.Scrollbar.Y.UnwrapOr(false) && comp.OverflowY == style.OverflowAuto && contentSize.Height > viewport.Height
	needX := !hasScrollbarX && comp.Scrollbar.X.UnwrapOr(false) && comp.OverflowX == style.OverflowAuto && contentSize.Width > viewport.Width

	if needY || needX {
		frag, _ = a.layoutInternal(hasScrollbarX || needX, hasScrollbarY || needY)
	}

	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

func (a *FlexAlgorithm) layoutInternal(hasScrollbarX, hasScrollbarY bool) (*Fragment, Size) {
	comp := a.Node.Style()
	geom := flexGeometry{direction: comp.FlexDirection}
	decor := ResolveDecorations(a.Node, hasScrollbarX, hasScrollbarY)

	// 1. Resolve Container Main/Cross Sizes
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes()
	}

	resolvedWidth := a.Space.AvailableSize.Width
	if !a.Space.IsFixedInlineSize {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedWidth = int(float32(a.Space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedWidth = comp.Width.CellsValue()
		case style.KindContent:
			resolvedWidth = min(minMax.Max, a.Space.AvailableSize.Width)
		case style.KindAuto:
			if comp.Display == style.DisplayInlineFlex {
				resolvedWidth = min(minMax.Max, a.Space.AvailableSize.Width)
			} else {
				resolvedWidth = a.Space.AvailableSize.Width
			}
		}
	}
	resolvedWidth = max(resolvedWidth, decor.Insets.Left+decor.Insets.Right)

	resolvedHeight := a.Space.AvailableSize.Height
	if !a.Space.IsFixedBlockSize {
		switch comp.Height.Kind() {
		case style.KindPercent:
			resolvedHeight = int(float32(a.Space.ContainerSpace.Height) * comp.Height.PercentValue() / 100.0)
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindAuto:
			// Resolve this later from lines if it's auto.
		case style.KindContent:
			// TODO: Resolve from content
		}
	}

	containerSize := Size{Width: resolvedWidth, Height: resolvedHeight}
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

	builder := NewFlexLineBuilder(geom, mainGap, crossGap)
	items := a.collectItems(geom)

	for _, item := range items {
		childStyle := item.Node.Style()
		childMargin := childStyle.Margin

		var baseSize, minSize, maxSize int

		// 1. Resolve Flex Basis (Main Size)
		basis := childStyle.Flex.Basis
		if basis.Kind() == style.KindCells {
			baseSize = basis.CellsValue()
		} else if basis.Kind() == style.KindContent {
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				baseSize = IntrinsicMinMaxSizes(item.Node).Max
			} else {
				baseSize = IntrinsicBlockSize(item.Node, contentCrossSizeForItems)
			}
		} else {
			// Auto: Use width/height property
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				if childStyle.Width.Kind() == style.KindCells {
					baseSize = childStyle.Width.CellsValue()
				} else {
					baseSize = IntrinsicMinMaxSizes(item.Node).Max
				}
			} else {
				if childStyle.Height.Kind() == style.KindCells {
					baseSize = childStyle.Height.CellsValue()
				} else {
					probeWidth := contentCrossSizeForItems
					if comp.AlignItems != style.AlignStretch && childStyle.Width.Kind() == style.KindAuto {
						probeWidth = min(IntrinsicMinMaxSizes(item.Node).Max, contentCrossSizeForItems)
					}
					baseSize = IntrinsicBlockSize(item.Node, probeWidth)
				}
			}
		}

		// 2. Resolve Min/Max Main Sizes
		if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
			baseSize += childMargin.Left + childMargin.Right

			// Default min-size is min-content (CSS flexbox spec: min-width: auto)
			if childStyle.MinWidth.Kind() == style.KindAuto {
				minSize = IntrinsicMinMaxSizes(item.Node).Min + childMargin.Left + childMargin.Right
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
				minSize = IntrinsicBlockSize(item.Node, contentCrossSizeForItems) + childMargin.Top + childMargin.Bottom
			} else if childStyle.MinHeight.Kind() == style.KindCells {
				minSize = childStyle.MinHeight.CellsValue() + childMargin.Top + childMargin.Bottom
			}

			if childStyle.MaxHeight.Kind() == style.KindCells {
				maxSize = childStyle.MaxHeight.CellsValue() + childMargin.Top + childMargin.Bottom
			}
		}

		builder.AddItem(item.Node, baseSize, minSize, maxSize, item.Grow, item.Shrink, item.Order)
	}

	// 3. Line Breaking
	wrap := comp.FlexWrap == style.FlexWrapOn
	builder.ComputeLines(contentMainSize, wrap)

	// 4. Resolve Main Sizes (Grow/Shrink)
	for i := range builder.Lines() {
		builder.ResolveFlexibleLengths(i, contentMainSize)
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
			flexContainingSpace := Size{Width: resolvedWidth, Height: resolvedHeight}
			flexContainerSpace := Size{Width: contentWidth, Height: contentHeight}

			childSpaceBuilder := NewConstraintSpaceBuilder(geom.MakeSize(childMainSize, measureCrossSize))
			childSpaceBuilder.SetContainingSpace(flexContainingSpace)
			childSpaceBuilder.SetContainerSpace(flexContainerSpace)
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.SetIsFixedBlockSize(false)

			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				childSpaceBuilder.SetIsFixedInlineSize(false)
				childSpaceBuilder.SetIsFixedBlockSize(false)

				if comp.AlignItems != style.AlignStretch && childStyle.Width.Kind() == style.KindAuto {
					measureCrossSize = min(IntrinsicMinMaxSizes(item.Node).Max, contentCrossSizeForItems)
					childSpaceBuilder = NewConstraintSpaceBuilder(geom.MakeSize(childMainSize, measureCrossSize))
					childSpaceBuilder.SetContainingSpace(flexContainingSpace)
					childSpaceBuilder.SetContainerSpace(flexContainerSpace)
					childSpaceBuilder.SetIsFixedInlineSize(true)
					childSpaceBuilder.SetIsFixedBlockSize(false)
				}
			}

			childAlgo := NewAlgorithm(item.Node, childSpaceBuilder.ToConstraintSpace())
			item.Fragment = childAlgo.Layout()

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
	if !a.Space.IsFixedInlineSize {
		if comp.Width.Kind() == style.KindAuto {
			if comp.Display == style.DisplayInlineFlex {
				var logicalWidth int
				if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
					logicalWidth = totalSumLineCross
				} else {
					logicalWidth = totalMaxLineMain
				}
				resolvedWidth = min(logicalWidth+decor.Insets.Left+decor.Insets.Right, a.Space.AvailableSize.Width)
			} else {
				resolvedWidth = a.Space.AvailableSize.Width
			}
		} else if comp.Width.Kind() == style.KindContent {
			var logicalWidth int
			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				logicalWidth = totalSumLineCross
			} else {
				logicalWidth = totalMaxLineMain
			}
			resolvedWidth = min(logicalWidth+decor.Insets.Left+decor.Insets.Right, a.Space.AvailableSize.Width)
		}
	}

	if !a.Space.IsFixedBlockSize {
		if comp.Height.Kind() == style.KindAuto || comp.Height.Kind() == style.KindContent {
			var logicalHeight int
			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				logicalHeight = totalMaxLineMain
			} else {
				logicalHeight = totalSumLineCross
			}
			resolvedHeight = logicalHeight + decor.Insets.Top + decor.Insets.Bottom
		}
	}

	containerSize = Size{Width: resolvedWidth, Height: resolvedHeight}
	contentCrossSizeForItems = geom.CrossSize(containerSize) - (decor.Insets.Top + decor.Insets.Bottom)
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		contentCrossSizeForItems = geom.CrossSize(containerSize) - (decor.Insets.Left + decor.Insets.Right)
	}

	// 6. Alignment & Positioning
	builder.AlignCrossAxis(contentCrossSizeForItems, comp.AlignContent, comp.AlignItems)
	isReverse := geom.direction == style.FlexRowReverse || geom.direction == style.FlexColumnReverse
	for i := range builder.Lines() {
		builder.AlignLine(i, contentMainSize, comp.JustifyContent, isReverse)
	}

	// Support block fragmentation
	var breakToken *BreakToken
	extraCross := contentCrossSizeForItems - totalSumLineCross

	if extraCross < 0 && a.Space.IsFixedBlockSize {
		currentTotalCross := 0
		breakLineIndex := -1
		itemsToSkip := 0
		if a.Space.BreakToken != nil {
			itemsToSkip = a.Space.BreakToken.ChildIndex
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
				Node:       a.Node,
				ChildIndex: itemsToSkip,
			}
			lines = lines[:breakLineIndex]
		}
	}

	fragBuilder := NewBoxFragmentBuilder(a.Node, a.Space)
	fragBuilder.SetInlineSize(resolvedWidth)
	fragBuilder.SetBlockSize(resolvedHeight)
	fragBuilder.SetBreakToken(breakToken)
	fragBuilder.SetHasScrollbarX(hasScrollbarX)
	fragBuilder.SetHasScrollbarY(hasScrollbarY)

	// Add children to fragment builder using resolved offsets
	for _, line := range lines {
		for _, item := range line.Items {
			offset := Point{
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
	return IsInlineLevel(node)
}

func (a *FlexAlgorithm) collectItems(geom flexGeometry) []*FlexItem {
	var allItems []*FlexItem

	children := a.Node.LayoutChildren()
	nextChild, stop := iter.Pull(children)
	defer stop()

	child, ok := nextChild()
	for ok {
		if a.isInlineLevel(child) {
			anon := &AnonymousBlock{
				parent: a.Node,
			}
			anon.children = append(anon.children, child)

			for {
				peek, peekOk := nextChild()
				if !peekOk {
					child = nil
					ok = false
					break
				}
				if a.isInlineLevel(peek) {
					anon.children = append(anon.children, peek)
				} else {
					child = peek
					ok = true
					break
				}
			}

			childStyle := anon.Style()
			item := &FlexItem{
				Node:   anon,
				Grow:   childStyle.Flex.Grow,
				Shrink: childStyle.Flex.Shrink,
			}
			allItems = append(allItems, item)

			if !ok {
				break
			}
			continue
		}

		childStyle := child.Style()
		item := &FlexItem{
			Node:   child,
			Grow:   childStyle.Flex.Grow,
			Shrink: childStyle.Flex.Shrink,
			Order:  childStyle.Order,
		}
		allItems = append(allItems, item)

		child, ok = nextChild()
	}

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
	if a.Space.BreakToken != nil {
		startIndex = a.Space.BreakToken.ChildIndex
	}

	if startIndex >= len(allItems) {
		return nil
	}

	return allItems[startIndex:]
}
