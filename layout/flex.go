package layout

import (
	"iter"
	"sort"

	"github.com/masterkeysrd/kite/style"
)

// FlexAlgorithm implements the Flexbox formatting context layout.
type FlexAlgorithm struct {
	Node  Node
	Space ConstraintSpace
}

// flexGeometry provides logical axis helpers.
type flexGeometry struct {
	direction style.FlexDirection
}

func (g flexGeometry) MainSize(s Size) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return s.Height
	}
	return s.Width
}

func (g flexGeometry) CrossSize(s Size) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return s.Width
	}
	return s.Height
}

func (g flexGeometry) MakeSize(main, cross int) Size {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return Size{Width: cross, Height: main}
	}
	return Size{Width: main, Height: cross}
}

func (g flexGeometry) MainAxis(p Point) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return p.Y
	}
	return p.X
}

func (g flexGeometry) CrossAxis(p Point) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return p.X
	}
	return p.Y
}

func (g flexGeometry) MakePoint(main, cross int) Point {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return Point{X: cross, Y: main}
	}
	return Point{X: main, Y: cross}
}

// FlexItem represents a transient layout state for a flex child.
type FlexItem struct {
	Node Node

	// Flex base size (initial main size before growing/shrinking).
	BaseSize int

	// Hypothetical main size (base size clamped by min/max constraints).
	HypotheticalMainSize int

	// Final resolved sizes.
	MainSize  int
	CrossSize int

	// Frozen indicates the item's main size has been fixed.
	Frozen bool

	// Style shortcuts.
	Grow   int
	Shrink int
	Order  int

	// Cached measurement result.
	Fragment *Fragment
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
	// Reset width/height to auto to allow the anonymous box to shrink-wrap its inlines
	s.Width = style.Auto
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

func (a *AnonymousBlock) LogicalNode() any    { return nil }
func (a *AnonymousBlock) IsDirtyLayout() bool { return true }
func (a *AnonymousBlock) ClearDirtyLayout()   {}
func (a *AnonymousBlock) Fragment() *Fragment { return nil }

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
	border := comp.Border.Widths()
	padding := comp.Padding
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right
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

	result = result.Add(parentDecorX)
	a.Node.SetCachedMinMaxSizes(result)
	return result
}

// Layout executes the flex layout algorithm and returns an immutable Fragment.
func (a *FlexAlgorithm) Layout() *Fragment {
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()
	geom := flexGeometry{direction: comp.FlexDirection}

	// 1. Resolve Container Main/Cross Sizes
	border := comp.Border.Widths()
	padding := comp.Padding
	decorX := border.Left + border.Right + padding.Left + padding.Right
	decorY := border.Top + border.Bottom + padding.Top + padding.Bottom

	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes()
	}

	resolvedWidth := a.Space.AvailableSize.Width
	if !a.Space.IsFixedInlineSize {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedWidth = int(float32(a.Space.PercentageResolutionSize.Width) * comp.Width.PercentValue() / 100.0)
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

	resolvedHeight := a.Space.AvailableSize.Height
	if !a.Space.IsFixedBlockSize {
		switch comp.Height.Kind() {
		case style.KindPercent:
			resolvedHeight = int(float32(a.Space.PercentageResolutionSize.Height) * comp.Height.PercentValue() / 100.0)
		case style.KindCells:
			resolvedHeight = comp.Height.CellsValue()
		case style.KindAuto:
			// Resolve this later from lines if it's auto.
		case style.KindContent:
			// TODO: Resolve from content
		}
	}

	containerSize := Size{Width: resolvedWidth, Height: resolvedHeight}

	decorMain := decorX
	decorCross := decorY
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		decorMain = decorY
		decorCross = decorX
	}
	contentMainSize := geom.MainSize(containerSize) - decorMain
	contentCrossSizeForItems := geom.CrossSize(containerSize) - decorCross

	// 2. Prepare Items
	items := a.collectItems(geom)

	// 3. Line Breaking
	lines := a.breakLines(items, geom, contentMainSize, contentCrossSizeForItems)

	// 4. Resolve Main Sizes (Grow/Shrink)
	for _, line := range lines {
		a.resolveFlexibleLengths(line, geom, contentMainSize)
	}

	// 5. Resolve Final Dimensions & Alignment
	// Pre-calculate line sizes
	totalMaxLineMain := 0
	totalSumLineCross := 0

	mainGap := comp.Gap.Column
	crossGap := comp.Gap.Row
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		mainGap = comp.Gap.Row
		crossGap = comp.Gap.Column
	}

	for i, line := range lines {
		lineCrossSize := 0
		lineMainSize := 0
		for j, item := range line.Items {
			// Measure child with its resolved main size (subtracting margins).
			childStyle := item.Node.Style()
			childMargin := childStyle.Margin
			childMainSize := item.MainSize
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				childMainSize -= childMargin.Left + childMargin.Right
			} else {
				childMainSize -= childMargin.Top + childMargin.Bottom
			}

			// Available cross size for child
			measureCrossSize := contentCrossSizeForItems

			childSpaceBuilder := NewConstraintSpaceBuilder(geom.MakeSize(childMainSize, measureCrossSize))
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.SetIsFixedBlockSize(false)

			if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
				childSpaceBuilder.SetIsFixedInlineSize(false)
				childSpaceBuilder.SetIsFixedBlockSize(true)

				// Flex items in a column should shrink-wrap their width if they are 'auto'
				// and the container doesn't stretch them.
				if comp.AlignItems != style.AlignStretch && childStyle.Width.Kind() == style.KindAuto {
					measureCrossSize = min(IntrinsicMinMaxSizes(item.Node).Max, contentCrossSizeForItems)
					childSpaceBuilder = NewConstraintSpaceBuilder(geom.MakeSize(childMainSize, measureCrossSize))
					childSpaceBuilder.SetIsFixedInlineSize(true)
					childSpaceBuilder.SetIsFixedBlockSize(true)
				}
			}

			childAlgo := NewAlgorithm(item.Node, childSpaceBuilder.ToConstraintSpace())
			item.Fragment = childAlgo.Layout()

			// Cross size of the item includes its cross margins.
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
		// DisplayFlex (block) stretches by default, DisplayInlineFlex shrink-wraps.
		if comp.Width.Kind() == style.KindAuto {
			if comp.Display == style.DisplayInlineFlex {
				var logicalWidth int
				if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
					logicalWidth = totalSumLineCross
				} else {
					logicalWidth = totalMaxLineMain
				}
				resolvedWidth = min(logicalWidth+decorX, a.Space.AvailableSize.Width)
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
			resolvedWidth = min(logicalWidth+decorX, a.Space.AvailableSize.Width)
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
			resolvedHeight = logicalHeight + decorY
		}
	}

	// Update container size with resolved values
	containerSize = Size{Width: resolvedWidth, Height: resolvedHeight}
	contentCrossSizeForItems = geom.CrossSize(containerSize) - decorCross

	// align-content: stretch (default)
	// If there's extra cross space, distribute it among the lines.
	extraCross := contentCrossSizeForItems - totalSumLineCross

	// Support block fragmentation
	var breakToken *BreakToken

	if extraCross > 0 && len(lines) > 0 {
		// For simplicity, we currently just give all extra space to the first line
		// or distribute if we want to be more accurate.
		// CSS says stretch distributes extra space equally to all lines.
		if comp.AlignContent == style.AlignStretch {
			perLineExtra := extraCross / len(lines)
			for _, line := range lines {
				line.CrossSize += perLineExtra
			}
		}
	} else if extraCross < 0 && a.Space.IsFixedBlockSize {
		// Content overflows fixed block size, find break point.
		currentTotalCross := 0
		breakLineIndex := -1

		// Total items to skip in next fragmentation
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

		if breakLineIndex != -1 {
			// Create break token to resume from the global child index
			breakToken = &BreakToken{
				Node:       a.Node,
				ChildIndex: itemsToSkip,
			}
			// Truncate lines for this fragment
			lines = lines[:breakLineIndex]
		}
	}

	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	builder.SetInlineSize(resolvedWidth)
	builder.SetBlockSize(resolvedHeight)
	builder.SetBreakToken(breakToken)

	a.layoutLines(builder, lines, geom, containerSize)

	frag := builder.ToFragment()
	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

type flexLine struct {
	Items     []*FlexItem
	MainSize  int
	CrossSize int
}

func (a *FlexAlgorithm) isInlineLevel(node Node) bool {
	comp := node.Style()
	return comp.Display == style.DisplayInline || comp.Display == style.DisplayInlineBlock || comp.Display == style.DisplayInlineFlex
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
	sort.SliceStable(allItems, func(i, j int) bool {
		return allItems[i].Order < allItems[j].Order
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

func (a *FlexAlgorithm) breakLines(items []*FlexItem, geom flexGeometry, availableMain, availableCross int) []*flexLine {
	comp := a.Node.Style()
	wrap := comp.FlexWrap == style.FlexWrapOn
	gap := comp.Gap.Column // Main axis gap if Row
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		gap = comp.Gap.Row
	}

	var lines []*flexLine
	if len(items) == 0 {
		return lines
	}

	currentLine := &flexLine{}
	lines = append(lines, currentLine)

	for _, item := range items {
		// Calculate base size.
		childStyle := item.Node.Style()
		childMargin := childStyle.Margin

		if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
			// Main axis is Width
			if childStyle.Width.Kind() == style.KindCells {
				item.BaseSize = childStyle.Width.CellsValue()
			} else {
				item.BaseSize = IntrinsicMinMaxSizes(item.Node).Max
			}
			item.BaseSize += childMargin.Left + childMargin.Right
		} else {
			// Main axis is Height
			if childStyle.Height.Kind() == style.KindCells {
				item.BaseSize = childStyle.Height.CellsValue()
			} else {
				// We need to measure the height based on the available width (cross axis)
				item.BaseSize = IntrinsicBlockSize(item.Node, availableCross)
			}
			item.BaseSize += childMargin.Top + childMargin.Bottom
		}

		item.HypotheticalMainSize = item.BaseSize

		if wrap && len(currentLine.Items) > 0 && currentLine.MainSize+gap+item.HypotheticalMainSize > availableMain {
			currentLine = &flexLine{}
			lines = append(lines, currentLine)
		}

		if len(currentLine.Items) > 0 {
			currentLine.MainSize += gap
		}
		currentLine.Items = append(currentLine.Items, item)
		currentLine.MainSize += item.HypotheticalMainSize
	}

	return lines
}

func (a *FlexAlgorithm) resolveFlexibleLengths(line *flexLine, geom flexGeometry, availableMain int) {
	// 1. Determine used flex factor.
	totalHypotheticalMainSize := 0
	for _, item := range line.Items {
		totalHypotheticalMainSize += item.HypotheticalMainSize
	}
	// Add gaps to the total hypothetical main size.
	comp := a.Node.Style()
	gap := comp.Gap.Column
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		gap = comp.Gap.Row
	}
	if len(line.Items) > 1 {
		totalHypotheticalMainSize += gap * (len(line.Items) - 1)
	}

	freeSpace := availableMain - totalHypotheticalMainSize
	useGrow := freeSpace > 0

	// 2. Size inflexible items.
	for _, item := range line.Items {
		item.Frozen = false
		if (useGrow && item.Grow == 0) || (!useGrow && item.Shrink == 0) {
			item.MainSize = item.HypotheticalMainSize
			item.Frozen = true
		}
	}

	// 3. Loop until all items are frozen.
	for {
		totalFlexFactor := 0
		totalBaseSizeFactor := 0 // for shrinking
		remainingFreeSpace := availableMain
		if len(line.Items) > 1 {
			remainingFreeSpace -= gap * (len(line.Items) - 1)
		}

		for _, item := range line.Items {
			if item.Frozen {
				remainingFreeSpace -= item.MainSize
			} else {
				remainingFreeSpace -= item.HypotheticalMainSize
				if useGrow {
					totalFlexFactor += item.Grow
				} else {
					totalFlexFactor += item.Shrink
					totalBaseSizeFactor += item.Shrink * item.BaseSize
				}
			}
		}

		if totalFlexFactor == 0 {
			for _, item := range line.Items {
				if !item.Frozen {
					item.MainSize = item.HypotheticalMainSize
					item.Frozen = true
				}
			}
			break
		}

		// Distribute free space.
		var violationCount int

		for _, item := range line.Items {
			if item.Frozen {
				continue
			}

			if useGrow {
				item.MainSize = item.HypotheticalMainSize + (remainingFreeSpace * item.Grow / totalFlexFactor)
			} else if totalBaseSizeFactor > 0 {
				shrinkAmount := ((-remainingFreeSpace) * item.Shrink * item.BaseSize) / totalBaseSizeFactor
				item.MainSize = item.HypotheticalMainSize - shrinkAmount
			} else {
				item.MainSize = item.HypotheticalMainSize
			}

			if item.MainSize < 0 {
				item.MainSize = 0
				item.Frozen = true
				violationCount++
			}
		}

		if violationCount == 0 {
			break
		}
		// If violations occurred, the frozen items' sizes are fixed and we loop again.
	}
}

func (a *FlexAlgorithm) layoutLines(builder *BoxFragmentBuilder, lines []*flexLine, geom flexGeometry, containerSize Size) {
	comp := a.Node.Style()
	padding := comp.Padding
	border := comp.Border.Widths()

	mainGap := comp.Gap.Column
	crossGap := comp.Gap.Row
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		mainGap = comp.Gap.Row
		crossGap = comp.Gap.Column
	}

	containerMainSize := geom.MainSize(containerSize) - (border.Left + border.Right + padding.Left + padding.Right)
	if geom.direction == style.FlexColumn || geom.direction == style.FlexColumnReverse {
		containerMainSize = geom.MainSize(containerSize) - (border.Top + border.Bottom + padding.Top + padding.Bottom)
	}

	isReverse := geom.direction == style.FlexRowReverse || geom.direction == style.FlexColumnReverse

	currentCrossOffset := geom.CrossAxis(Point{X: border.Left + padding.Left, Y: border.Top + padding.Top})

	for _, line := range lines {
		// Calculate Main-Axis alignment (justify-content)
		lineMainSize := 0
		for i, item := range line.Items {
			lineMainSize += item.MainSize
			if i > 0 {
				lineMainSize += mainGap
			}
		}

		remainingMain := containerMainSize - lineMainSize
		var startMainOffset int
		var itemSpacing = mainGap

		switch comp.JustifyContent {
		case style.JustifyStart:
			if isReverse {
				startMainOffset = remainingMain
			} else {
				startMainOffset = 0
			}
		case style.JustifyEnd:
			if isReverse {
				startMainOffset = 0
			} else {
				startMainOffset = remainingMain
			}
		case style.JustifyCenter:
			startMainOffset = remainingMain / 2
		case style.JustifyBetween:
			if len(line.Items) > 1 {
				itemSpacing = mainGap + remainingMain/(len(line.Items)-1)
			}
		case style.JustifyAround:
			if len(line.Items) > 0 {
				itemSpacing = mainGap + remainingMain/len(line.Items)
				startMainOffset = (itemSpacing - mainGap) / 2
			}
		case style.JustifyEvenly:
			if len(line.Items) > 0 {
				itemSpacing = mainGap + remainingMain/(len(line.Items)+1)
				startMainOffset = itemSpacing - mainGap
			}
		}

		// Second pass to position items.
		currentMainOffset := geom.MainAxis(Point{X: border.Left + padding.Left, Y: border.Top + padding.Top}) + startMainOffset
		for i, item := range line.Items {
			childStyle := item.Node.Style()
			childMargin := childStyle.Margin

			// Calculate Cross-Axis alignment (align-items)
			crossOffset := 0
			alignSelf := comp.AlignItems // TODO: Support align-self on child

			// Available cross space for this item (line size minus its own margins)
			itemAvailableCross := line.CrossSize
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				itemAvailableCross -= childMargin.Top + childMargin.Bottom
			} else {
				itemAvailableCross -= childMargin.Left + childMargin.Right
			}

			itemActualCross := geom.CrossSize(item.Fragment.Size)

			switch alignSelf {
			case style.AlignStart:
				crossOffset = 0
			case style.AlignEnd:
				crossOffset = itemAvailableCross - itemActualCross
			case style.AlignCenter:
				crossOffset = (itemAvailableCross - itemActualCross) / 2
			case style.AlignStretch:
				crossOffset = 0
			}

			// Add the "before" margin on the cross axis.
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				crossOffset += childMargin.Top
			} else {
				crossOffset += childMargin.Left
			}

			// Add the "before" margin on the main axis.
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				currentMainOffset += childMargin.Left
			} else {
				currentMainOffset += childMargin.Top
			}

			offset := geom.MakePoint(currentMainOffset, currentCrossOffset+crossOffset)
			builder.AddChild(item.Fragment, offset)

			currentMainOffset += geom.MainSize(item.Fragment.Size)

			// Add the "after" margin on the main axis.
			if geom.direction == style.FlexRow || geom.direction == style.FlexRowReverse {
				currentMainOffset += childMargin.Right
			} else {
				currentMainOffset += childMargin.Bottom
			}

			if i < len(line.Items)-1 {
				currentMainOffset += itemSpacing
			}
		}
		currentCrossOffset += line.CrossSize + crossGap
	}
}
