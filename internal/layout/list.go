package layout

import (
	"fmt"
	"strconv"

	"github.com/masterkeysrd/kite/dom"
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

var decimalMarkerCache [101]string

func init() {
	for i := 1; i <= 100; i++ {
		decimalMarkerCache[i] = fmt.Sprintf("%d. ", i)
	}
}

// ListAlgorithm implements the layout for list items with markers.
// It formats the list item as a two-column row: Column 1 is the marker,
// and Column 2 contains the children laid out using Block layout rules.
type ListAlgorithm struct{}

func (a *ListAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	// 1. Cache Check
	if cached := node.CachedLayout(space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(List)")()

	comp := node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding

	// 2. Resolve inline size
	var minMax MinMaxSizes
	if !space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes(ctx, node)
	}

	var resolvedInlineSize int
	if space.IsFixedInlineSize {
		resolvedInlineSize = space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			resolvedInlineSize = space.AvailableSize.Width
		case style.KindContent:
			resolvedInlineSize = min(minMax.Max, space.AvailableSize.Width)
		default:
			resolvedInlineSize = space.AvailableSize.Width
		}
	}

	builder := AcquireBoxFragmentBuilder(node, space)
	builder.SetInlineSize(resolvedInlineSize)

	parentDecorX := border.Left + border.Right + padding.Left + padding.Right
	parentDecorY := border.Top + border.Bottom + padding.Top + padding.Bottom
	contentWidth := max(0, resolvedInlineSize-parentDecorX)

	// 3. Synthesize Marker
	markerText := a.getMarkerText(node)
	var markerFrag *Fragment
	markerWidth := 0
	if markerText != "" {
		// Use shaper to create marker fragment
		shaped := defaultShaper.Shape(markerText)
		markerFrag = &Fragment{
			Size: geometry.Size{
				Width:  defaultShaper.MeasureRun(markerText),
				Height: 1,
			},
			Text:       shaped,
			ParentNode: node,
		}
		markerWidth = markerFrag.Size.Width
	}

	// 4. Layout Children (Block Layout in Column 2)
	insetX := border.Left + padding.Left
	column2X := insetX + markerWidth
	column2Width := max(0, contentWidth-markerWidth)

	if markerFrag != nil {
		builder.AddChild(markerFrag, geometry.Point{X: insetX, Y: border.Top + padding.Top})
	}

	// Mimic BlockAlgorithm's child iteration but constrained to Column 2
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
		breaker := AcquireLineBreaker(items, column2Width, comp.TextAlign, comp.AlignItems)
		for {
			line, ok := breaker.NextLine(ctx)
			if !ok {
				break
			}
			lineFrag := line.ToFragment()
			offset := geometry.Point{
				X: column2X,
				Y: builder.CurrentBlockOffset(),
			}
			builder.AddChild(lineFrag, offset)
			builder.AdvanceBlockOffset(lineFrag.Size.Height)
		}
		ReleaseLineBreaker(breaker)
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
		containingSpace := geometry.Size{Width: resolvedInlineSize, Height: space.AvailableSize.Height}
		containerSpace := geometry.Size{Width: column2Width, Height: max(0, space.AvailableSize.Height-parentDecorY)}
		adjustedContainer := geometry.Size{
			Width:  containerSpace.Width,
			Height: max(0, containerSpace.Height-builder.CurrentBlockOffset()),
		}
		childSpace := BuildChildSpace(child, adjustedContainer, containingSpace, space)
		childAlgo := GetAlgorithm(child)
		childFrag := childAlgo.Layout(ctx, child, childSpace)

		offset := geometry.Point{
			X: column2X + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)
	}

	processInlines()

	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

	if space.IsFixedBlockSize {
		builder.SetBlockSize(space.AvailableSize.Height)
	} else {
		resolvedHeight := builder.CurrentBlockOffset()
		if markerFrag != nil {
			resolvedHeight = max(resolvedHeight, border.Top+padding.Top+markerFrag.Size.Height+padding.Bottom+border.Bottom)
		}
		if comp.Height.Kind() == style.KindCells {
			resolvedHeight = max(resolvedHeight, comp.Height.CellsValue())
		}
		builder.SetBlockSize(resolvedHeight)
	}

	frag := builder.ToFragment()
	node.SetCachedLayout(space, frag)
	return frag
}

func (a *ListAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	if sizes, ok := node.CachedMinMaxSizes(); ok {
		return sizes
	}
	defer ctx.Begin("Layout(List):ComputeMinMaxSizes")()

	comp := node.Style()
	border := comp.Border.Widths()
	parentDecorX := border.Left + border.Right + comp.Padding.Left + comp.Padding.Right

	markerText := a.getMarkerText(node)
	markerWidth := 0
	if markerText != "" {
		markerWidth = defaultShaper.MeasureRun(markerText)
	}

	sizes := MinMaxSizes{Min: markerWidth, Max: markerWidth}

	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		childSizes := IntrinsicMinMaxSizes(ctx, child)
		childMargin := child.Style().Margin
		childDecorX := childMargin.Left + childMargin.Right

		sizes.Min = max(sizes.Min, markerWidth+childSizes.Min+childDecorX)
		sizes.Max = max(sizes.Max, markerWidth+childSizes.Max+childDecorX)
	}

	sizes.Min += parentDecorX
	sizes.Max += parentDecorX

	node.SetCachedMinMaxSizes(sizes)
	return sizes
}

func (a *ListAlgorithm) getMarkerText(node Node) string {
	comp := node.Style()
	switch comp.ListStyleType {
	case style.ListStyleNone:
		return ""
	case style.ListStyleDisc:
		return "• "
	case style.ListStyleCircle:
		return "○ "
	case style.ListStyleSquare:
		return "■ "
	case style.ListStyleDecimal:
		ordinal := a.computeOrdinal(node)
		if ordinal >= 1 && ordinal <= 100 {
			return decimalMarkerCache[ordinal]
		}
		return strconv.Itoa(ordinal) + ". "
	default:
		return "• "
	}
}

func (a *ListAlgorithm) computeOrdinal(node Node) int {
	ordinal := 1
	var logical dom.Node = node.LogicalNode()
	if logical == nil {
		return 1
	}

	doc := logical.OwnerDocument()
	if doc == nil {
		return 1
	}
	view := doc.DefaultView()

	curr := logical
	for {
		prev := curr.PreviousSibling()
		if prev == nil {
			break
		}
		curr = prev

		if view != nil {
			cs := view.GetComputedStyle(curr)
			if cs != nil && cs.Display != style.DisplayListItem {
				// Non-list-item sibling resets the count.
				break
			}
		}
		ordinal++
	}
	return ordinal
}
