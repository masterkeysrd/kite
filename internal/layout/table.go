package layout

import (
	"github.com/masterkeysrd/kite/dom"
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// TableAlgorithm implements the DisplayTable layout.
type TableAlgorithm struct{}

// Layout executes the two-pass table layout algorithm and returns an immutable Fragment.
func (a *TableAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	if cached := node.CachedLayout(space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(Table)")()

	comp := node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right

	// Pass 1: Grid Sizing
	builder := AcquireTableFragmentBuilder(node, space)
	defer ReleaseTableFragmentBuilder(builder)
	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		display := child.Style().Display
		switch display {
		case style.DisplayTableHeaderGroup:
			builder.AddHeaderChild(child)
		case style.DisplayTableFooterGroup:
			builder.AddFooterChild(child)
		case style.DisplayTableRowGroup:
			builder.AddBodyChild(child)
		case style.DisplayTableRow:
			builder.AddRowChild(child)
		default:
			builder.AddNonRowChild(child)
		}
	}

	builder.BuildGrid(ctx)
	colMinMax := builder.colMinMax

	// Resolve the table's inline size.
	var resolvedInlineSize int
	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}

	// Subtract collapsed border widths.
	for _, overlap := range builder.grid.ColJunctionOverlap {
		if overlap {
			tableMinMax.Min--
			tableMinMax.Max--
		}
	}
	if builder.grid.LeftEdgeHasOverlap {
		tableMinMax.Min--
		tableMinMax.Max--
	}
	if builder.grid.RightEdgeHasOverlap {
		tableMinMax.Min--
		tableMinMax.Max--
	}

	// Add padding and borders.
	tableMinMax.Min += parentDecorX
	tableMinMax.Max += parentDecorX

	if space.IsFixedInlineSize {
		resolvedInlineSize = space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			resolvedInlineSize = min(max(tableMinMax.Min, space.AvailableSize.Width), tableMinMax.Max)
		case style.KindContent:
			resolvedInlineSize = tableMinMax.Max
		default:
			resolvedInlineSize = tableMinMax.Max
		}
	}
	resolvedInlineSize = max(resolvedInlineSize, tableMinMax.Min)

	// Distribute extra width among columns
	builder.ResolveWidths(resolvedInlineSize, parentDecorX)
	builder.boxBuilder.SetInlineSize(resolvedInlineSize)

	// Pass 2: Layout Sections
	sectionInsetX := border.Left + padding.Left
	childAvailWidthBase := resolvedInlineSize - parentDecorX
	if builder.grid.LeftEdgeHasOverlap {
		sectionInsetX = padding.Left
		childAvailWidthBase += border.Left
	}
	if builder.grid.RightEdgeHasOverlap {
		childAvailWidthBase += border.Right
	}

	rowIdx := 0
	for _, sectionNode := range builder.Sections() {
		childAvailWidth := childAvailWidthBase
		childAvailHeight := max(0, space.AvailableSize.Height-builder.CurrentBlockOffset()-padding.Top-padding.Bottom)

		childSpace := ConstraintSpace{
			AvailableSize:     geometry.Size{Width: childAvailWidth, Height: childAvailHeight},
			ContainingSpace:   geometry.Size{Width: resolvedInlineSize, Height: space.AvailableSize.Height},
			ContainerSpace:    geometry.Size{Width: childAvailWidth, Height: childAvailHeight},
			IsFixedInlineSize: true,
		}

		var childFrag *Fragment
		numRows := 0
		for c := sectionNode.FirstLayoutChild(); c != nil; c = sectionNode.NextLayoutSibling(c) {
			numRows++
		}

		if sectionNode.Style().Display == style.DisplayTableRow {
			var rowData *tableRowGrid
			if rowIdx < len(builder.grid.Rows) {
				rowData = builder.grid.Rows[rowIdx]
			}
			childFrag = tableRowAlgo.LayoutWithData(ctx, sectionNode, childSpace, builder, builder.colWidths, rowData)
			rowIdx++
		} else {
			var rowsData []*tableRowGrid
			if rowIdx+numRows <= len(builder.grid.Rows) {
				rowsData = builder.grid.Rows[rowIdx : rowIdx+numRows]
			}
			childFrag = tableSectionAlgo.LayoutWithData(ctx, sectionNode, childSpace, builder, builder.colWidths, rowsData)
			rowIdx += numRows
		}

		offset := geometry.Point{
			X: sectionInsetX,
			Y: builder.CurrentBlockOffset(),
		}

		builder.boxBuilder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childFrag.Size.Height)
	}

	lastRowHasBottom := builder.lastRowBorderBottom
	tableHasBottom := comp.Border.Edges.Bottom

	bottomDecor := border.Bottom + padding.Bottom
	if lastRowHasBottom && tableHasBottom {
		bottomDecor -= 1
	}
	builder.AdvanceBlockOffset(bottomDecor)

	if space.IsFixedBlockSize {
		builder.SetBlockSize(space.AvailableSize.Height)
	} else {
		resolvedHeight := builder.CurrentBlockOffset()
		if comp.Height.Kind() == style.KindCells {
			resolvedHeight = max(resolvedHeight, comp.Height.CellsValue())
		}
		builder.SetBlockSize(resolvedHeight)
	}

	frag := builder.ToFragment()
	node.SetCachedLayout(space, frag)
	return frag
}

func (a *TableAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	if sizes, ok := node.CachedMinMaxSizes(); ok {
		return sizes
	}
	defer ctx.Begin("Layout(Table):ComputeMinMaxSizes")()

	builder := AcquireTableFragmentBuilder(node, ConstraintSpace{})
	defer ReleaseTableFragmentBuilder(builder)
	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
		display := child.Style().Display
		switch display {
		case style.DisplayTableHeaderGroup:
			builder.AddHeaderChild(child)
		case style.DisplayTableFooterGroup:
			builder.AddFooterChild(child)
		case style.DisplayTableRowGroup:
			builder.AddBodyChild(child)
		case style.DisplayTableRow:
			builder.AddRowChild(child)
		default:
			builder.AddNonRowChild(child)
		}
	}
	builder.BuildGrid(ctx)

	comp := node.Style()
	colMinMax := builder.colMinMax
	borderX := comp.Border.Widths()
	parentDecorX := borderX.Left + borderX.Right + comp.Padding.Left + comp.Padding.Right
	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}

	for _, overlap := range builder.grid.ColJunctionOverlap {
		if overlap {
			tableMinMax.Min--
			tableMinMax.Max--
		}
	}
	if builder.grid.LeftEdgeHasOverlap {
		tableMinMax.Min--
		tableMinMax.Max--
	}
	if builder.grid.RightEdgeHasOverlap {
		tableMinMax.Min--
		tableMinMax.Max--
	}

	tableMinMax.Min += parentDecorX
	tableMinMax.Max += parentDecorX

	node.SetCachedMinMaxSizes(tableMinMax)
	return tableMinMax
}

// TableSectionAlgorithm implements the layout for table header, body, and footer groups.
type TableSectionAlgorithm struct{}

func (a *TableSectionAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	return a.LayoutWithData(ctx, node, space, nil, nil, nil)
}

func (a *TableSectionAlgorithm) LayoutWithData(ctx *Context, node Node, space ConstraintSpace, tableBuilder *TableFragmentBuilder, colWidths []int, rowsData []*tableRowGrid) *Fragment {
	if cached := node.CachedLayout(space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(TableSection)")()

	comp := node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding

	builder := AcquireBoxFragmentBuilder(node, space)
	if space.IsFixedInlineSize {
		builder.SetInlineSize(space.AvailableSize.Width)
	}

	rowIdx := 0
	for rowNode := node.FirstLayoutChild(); rowNode != nil; rowNode = node.NextLayoutSibling(rowNode) {
		childAvailWidth := max(0, space.AvailableSize.Width-border.Left-border.Right-padding.Left-padding.Right)
		childAvailHeight := max(0, space.AvailableSize.Height-builder.CurrentBlockOffset()-(border.Top+border.Bottom+padding.Top+padding.Bottom))

		childSpace := ConstraintSpace{
			AvailableSize:     geometry.Size{Width: childAvailWidth, Height: childAvailHeight},
			ContainingSpace:   geometry.Size{Width: space.AvailableSize.Width, Height: space.AvailableSize.Height},
			ContainerSpace:    geometry.Size{Width: childAvailWidth, Height: childAvailHeight},
			IsFixedInlineSize: true,
		}

		var rowData *tableRowGrid
		if rowIdx < len(rowsData) {
			rowData = rowsData[rowIdx]
		}
		childFrag := tableRowAlgo.LayoutWithData(ctx, rowNode, childSpace, tableBuilder, colWidths, rowData)

		offset := geometry.Point{
			X: 0,
			Y: builder.CurrentBlockOffset(),
		}

		hasTopBorder := false
		hasBottomBorder := false
		if rowIdx < len(rowsData) {
			hasTopBorder = rowsData[rowIdx].HasTopBorder
			hasBottomBorder = rowsData[rowIdx].HasBottomBorder
		} else {
			hasTopBorder = rowNode.Style().Border.Edges.Top
			hasBottomBorder = rowNode.Style().Border.Edges.Bottom
		}

		shift := 0
		if tableBuilder != nil {
			shift = tableBuilder.AdjustRowOffset(hasTopBorder, hasBottomBorder)
		}
		offset.Y += shift
		builder.AdvanceBlockOffset(shift)

		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childFrag.Size.Height)
		rowIdx++
	}

	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

	if space.IsFixedBlockSize {
		builder.SetBlockSize(space.AvailableSize.Height)
	} else {
		builder.SetBlockSize(builder.CurrentBlockOffset())
	}

	frag := builder.ToFragment()
	node.SetCachedLayout(space, frag)
	return frag
}

func (a *TableSectionAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return MinMaxSizes{}
}

// TableRowAlgorithm implements the DisplayTableRow layout.
type TableRowAlgorithm struct{}

func (a *TableRowAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	return a.LayoutWithData(ctx, node, space, nil, nil, nil)
}

func (a *TableRowAlgorithm) LayoutWithData(ctx *Context, node Node, space ConstraintSpace, tableBuilder *TableFragmentBuilder, colWidths []int, rowData *tableRowGrid) *Fragment {
	defer ctx.Begin("Layout(TableRow)")()

	comp := node.Style()
	padding := comp.Padding

	builder := AcquireBoxFragmentBuilder(node, space)
	if space.IsFixedInlineSize {
		builder.SetInlineSize(space.AvailableSize.Width)
	}
	builder.currentBlockOffset = 0

	maxCellHeight := 0
	totalShiftX := 0

	if tableBuilder != nil {
		tableBuilder.ResetRow()
	}

	if rowData != nil {
		for _, cell := range rowData.Cells {
			cellWidth := 0
			for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
				if c < len(colWidths) {
					cellWidth += colWidths[c]
				}
			}
			if tableBuilder != nil && cell.ColSpan > 1 {
				for j := cell.ColStart; j < cell.ColStart+cell.ColSpan-1; j++ {
					if j < len(tableBuilder.grid.ColJunctionOverlap) && tableBuilder.grid.ColJunctionOverlap[j] {
						cellWidth--
					}
				}
			}

			childMargin := cell.Node.Style().Margin
			cellAvailWidth := max(0, cellWidth-childMargin.Left-childMargin.Right)

			childSpace := ConstraintSpace{
				AvailableSize:     geometry.Size{Width: cellAvailWidth, Height: space.AvailableSize.Height},
				ContainingSpace:   geometry.Size{Width: cellWidth, Height: space.AvailableSize.Height},
				ContainerSpace:    geometry.Size{Width: cellAvailWidth, Height: space.AvailableSize.Height},
				IsFixedInlineSize: true,
			}

			childAlgo := GetAlgorithm(cell.Node)
			childFrag := childAlgo.Layout(ctx, cell.Node, childSpace)

			xOffset := 0
			for c := 0; c < cell.ColStart; c++ {
				if c < len(colWidths) {
					xOffset += colWidths[c]
				}
			}

			offset := geometry.Point{
				X: xOffset - totalShiftX,
				Y: builder.CurrentBlockOffset(),
			}

			if tableBuilder != nil {
				edges := cell.Node.Style().Border.Edges
				shift := tableBuilder.GetCellShift(cell.ColStart, cell.ColSpan, edges.Left, edges.Right)
				totalShiftX += shift
				offset.X -= shift
			}

			builder.AddChild(childFrag, offset)
			if childFrag.Size.Height > maxCellHeight {
				maxCellHeight = childFrag.Size.Height
			}
		}
	}

	builder.AdvanceBlockOffset(maxCellHeight + padding.Bottom)

	if space.IsFixedBlockSize {
		builder.SetBlockSize(space.AvailableSize.Height)
	} else {
		builder.SetBlockSize(builder.CurrentBlockOffset())
	}

	return builder.ToFragment()
}

func (a *TableRowAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return MinMaxSizes{}
}

// anonymousTableSection and anonymousTableRow Node implementations (omitted, remain same)
// Wait, I should include them to ensure the file is complete.
type anonymousTableSection struct {
	parent      Node
	children    []Node
	display     style.Display
	cachedSpace ConstraintSpace
}

var _ Node = (*anonymousTableSection)(nil)

func (a *anonymousTableSection) Style() *style.Computed {
	s := *a.parent.Style()
	s.Display = a.display
	s.Margin = style.EdgeValues[int]{}
	s.Padding = style.EdgeValues[int]{}
	s.Border = style.Border{}
	s.Width = style.Auto
	s.Height = style.Auto
	return &s
}

func (a *anonymousTableSection) FirstLayoutChild() Node {
	if len(a.children) == 0 {
		return nil
	}
	return a.children[0]
}

func (a *anonymousTableSection) NextLayoutSibling(child Node) Node {
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

func (a *anonymousTableSection) LogicalNode() dom.Node    { return nil }
func (a *anonymousTableSection) IsDirtyLayout() bool      { return true }
func (a *anonymousTableSection) IsDirtyPaint() bool       { return true }
func (a *anonymousTableSection) HasChildNeedsPaint() bool { return true }
func (a *anonymousTableSection) ClearDirtyLayout()        {}
func (a *anonymousTableSection) Fragment() *Fragment      { return nil }

func (a *anonymousTableSection) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableSection) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableSection) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *anonymousTableSection) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *anonymousTableSection) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *anonymousTableSection) CachedBlockSize(width int) (int, bool) { return 0, false }
func (a *anonymousTableSection) SetCachedBlockSize(width, height int)  {}

func (a *anonymousTableSection) SetOffset(p geometry.Point) {}

func (a *anonymousTableSection) IsAnonymous() bool {
	return true
}

func (a *anonymousTableSection) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return MinMaxSizes{}
}

type anonymousTableRow struct {
	parent      Node
	children    []Node
	cachedSpace ConstraintSpace
}

var _ Node = (*anonymousTableRow)(nil)

func (a *anonymousTableRow) Style() *style.Computed {
	s := *a.parent.Style()
	s.Display = style.DisplayTableRow
	s.Margin = style.EdgeValues[int]{}
	s.Padding = style.EdgeValues[int]{}
	s.Border = style.Border{}
	s.Width = style.Auto
	s.Height = style.Auto
	return &s
}

func (a *anonymousTableRow) FirstLayoutChild() Node {
	if len(a.children) == 0 {
		return nil
	}
	return a.children[0]
}

func (a *anonymousTableRow) NextLayoutSibling(child Node) Node {
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

func (a *anonymousTableRow) LogicalNode() dom.Node    { return nil }
func (a *anonymousTableRow) IsDirtyLayout() bool      { return true }
func (a *anonymousTableRow) IsDirtyPaint() bool       { return true }
func (a *anonymousTableRow) HasChildNeedsPaint() bool { return true }
func (a *anonymousTableRow) ClearDirtyLayout()        {}
func (a *anonymousTableRow) Fragment() *Fragment      { return nil }

func (a *anonymousTableRow) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableRow) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableRow) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *anonymousTableRow) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *anonymousTableRow) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *anonymousTableRow) CachedBlockSize(width int) (int, bool) { return 0, false }
func (a *anonymousTableRow) SetCachedBlockSize(width, height int)  {}

func (a *anonymousTableRow) SetOffset(p geometry.Point) {}

func (a *anonymousTableRow) IsAnonymous() bool {
	return true
}

func (a *anonymousTableRow) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return MinMaxSizes{}
}
