package layout

import (
	"fmt"
	"iter"
	"reflect"

	"github.com/masterkeysrd/kite/style"
)

// ListAlgorithm implements the layout for list items with markers.
// It formats the list item as a two-column row: Column 1 is the marker,
// and Column 2 contains the children laid out using Block layout rules.
type ListAlgorithm struct {
	Node  Node
	Space ConstraintSpace
}

func (a *ListAlgorithm) Layout() *Fragment {
	// 1. Cache Check
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding

	// 2. Resolve inline size
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize {
		minMax = a.ComputeMinMaxSizes()
	}

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
			resolvedInlineSize = a.Space.AvailableSize.Width
		case style.KindContent:
			resolvedInlineSize = min(minMax.Max, a.Space.AvailableSize.Width)
		default:
			resolvedInlineSize = a.Space.AvailableSize.Width
		}
	}

	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	builder.SetInlineSize(resolvedInlineSize)

	parentDecorX := border.Left + border.Right + padding.Left + padding.Right
	parentDecorY := border.Top + border.Bottom + padding.Top + padding.Bottom
	contentWidth := max(0, resolvedInlineSize-parentDecorX)

	// 3. Synthesize Marker
	markerText := a.getMarkerText()
	var markerFrag *Fragment
	markerWidth := 0
	if markerText != "" {
		// Use shaper to create marker fragment
		shaped := defaultShaper.Shape(markerText)
		markerFrag = &Fragment{
			Size: Size{
				Width:  defaultShaper.MeasureRun(markerText),
				Height: 1,
			},
			Text: shaped,
		}
		markerWidth = markerFrag.Size.Width
	}

	// 4. Layout Children (Block Layout in Column 2)
	insetX := border.Left + padding.Left
	column2X := insetX + markerWidth
	column2Width := max(0, contentWidth-markerWidth)

	if markerFrag != nil {
		builder.AddChild(markerFrag, Point{X: insetX, Y: border.Top + padding.Top})
	}

	// Mimic BlockAlgorithm's child iteration but constrained to Column 2
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

			items := inlineBuilder.items
			breaker := NewLineBreaker(items, column2Width, comp.TextAlign, comp.AlignItems)
			for {
				line, ok := breaker.NextLine()
				if !ok {
					break
				}
				lineFrag := line.ToFragment()
				offset := Point{
					X: column2X,
					Y: builder.CurrentBlockOffset(),
				}
				builder.AddChild(lineFrag, offset)
				builder.AdvanceBlockOffset(lineFrag.Size.Height)
			}

			if !ok {
				break
			}
			continue
		}

		// Block Child
		childStyle := child.Style()
		childMargin := childStyle.Margin

		childAvailWidth := max(0, column2Width-childMargin.Left-childMargin.Right)
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-childMargin.Top-childMargin.Bottom-(border.Bottom+padding.Bottom))

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{
			Width:  column2Width,
			Height: max(0, a.Space.AvailableSize.Height-parentDecorY),
		})

		if childStyle.Width.Kind() == style.KindCells {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = childStyle.Width.CellsValue()
		} else if childStyle.Width.Kind() == style.KindPercent {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = int(float32(column2Width) * childStyle.Width.PercentValue() / 100.0)
		} else if childStyle.Width.Kind() == style.KindAuto {
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpaceBuilder.space.AvailableSize.Width = childAvailWidth
		}

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
			X: column2X + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)

		child, ok = nextChild()
	}

	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		resolvedHeight := builder.CurrentBlockOffset()
		// Ensure height is at least 1 if we have a marker, even if no children
		if markerFrag != nil {
			resolvedHeight = max(resolvedHeight, border.Top+padding.Top+markerFrag.Size.Height+padding.Bottom+border.Bottom)
		}
		if comp.Height.Kind() == style.KindCells {
			resolvedHeight = comp.Height.CellsValue()
		}
		builder.SetBlockSize(resolvedHeight)
	}

	frag := builder.ToFragment()
	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

func (a *ListAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}

	comp := a.Node.Style()
	border := comp.Border.Widths()
	parentDecorX := border.Left + border.Right + comp.Padding.Left + comp.Padding.Right

	markerText := a.getMarkerText()
	markerWidth := 0
	if markerText != "" {
		markerWidth = defaultShaper.MeasureRun(markerText)
	}

	sizes := MinMaxSizes{Min: markerWidth, Max: markerWidth}

	for child := range a.Node.LayoutChildren() {
		childSizes := IntrinsicMinMaxSizes(child)
		childMargin := child.Style().Margin
		childDecorX := childMargin.Left + childMargin.Right

		sizes.Min = max(sizes.Min, markerWidth+childSizes.Min+childDecorX)
		sizes.Max = max(sizes.Max, markerWidth+childSizes.Max+childDecorX)
	}

	sizes.Min += parentDecorX
	sizes.Max += parentDecorX

	a.Node.SetCachedMinMaxSizes(sizes)
	return sizes
}

func (a *ListAlgorithm) getMarkerText() string {
	comp := a.Node.Style()
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
		ordinal := a.computeOrdinal()
		return fmt.Sprintf("%d. ", ordinal)
	default:
		return "• "
	}
}

func (a *ListAlgorithm) computeOrdinal() int {
	ordinal := 1
	logical := a.Node.LogicalNode()
	if logical == nil {
		return 1
	}

	curr := logical
	for {
		// Use reflection to call PreviousSibling() to avoid import cycles.
		currVal := reflect.ValueOf(curr)
		method := currVal.MethodByName("PreviousSibling")
		if !method.IsValid() {
			break
		}
		results := method.Call(nil)
		if len(results) == 0 || results[0].IsNil() {
			break
		}
		curr = results[0].Interface()

		// Get RenderObject from current logical node
		roMethod := reflect.ValueOf(curr).MethodByName("RenderObject")
		if !roMethod.IsValid() {
			break
		}
		roResults := roMethod.Call(nil)
		if len(roResults) == 0 || roResults[0].IsNil() {
			// Some nodes might not have a render object (e.g. DisplayNone)
			// But we should continue walking?
			// CSS says only DisplayListItem matters.
			continue
		}
		ro := roResults[0].Interface()

		// Check if the render object has the correct style
		type hasStyle interface {
			Style() *style.Computed
		}
		if s, ok := ro.(hasStyle); ok {
			comp := s.Style()
			if comp != nil && comp.Display == style.DisplayListItem {
				ordinal++
			} else {
				// Consecutive DisplayListItem nodes
				break
			}
		} else {
			break
		}
	}
	return ordinal
}
