package layout

import (
	"iter"

	"github.com/masterkeysrd/kite/style"
)

// TableAlgorithm implements the DisplayTable layout.
type TableAlgorithm struct {
	Node  Node
	Space ConstraintSpace
}

// Layout executes the two-pass table layout algorithm and returns an immutable Fragment.
func (a *TableAlgorithm) Layout() *Fragment {
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right

	// Pass 1: Grid Sizing
	builder := NewTableFragmentBuilder(a.Node, a.Space)
	for child := range a.Node.LayoutChildren() {
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

	builder.BuildGrid()
	colMinMax := builder.colMinMax

	// Resolve the table's inline size.
	var resolvedInlineSize int
	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}

	// Subtract collapsed border widths: only junctions where both adjacent cells
	// actually have touching borders, plus table-edge overlaps.
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

	// Add padding and borders (overlaps are subtracted above).
	tableMinMax.Min += parentDecorX
	tableMinMax.Max += parentDecorX

	if a.Space.IsFixedInlineSize {
		resolvedInlineSize = a.Space.AvailableSize.Width
	} else {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedInlineSize = int(float32(a.Space.PercentageResolutionSize.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedInlineSize = comp.Width.CellsValue()
		case style.KindAuto:
			// Shrink to fit, up to available width
			resolvedInlineSize = min(max(tableMinMax.Min, a.Space.AvailableSize.Width), tableMinMax.Max)
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
	// Determine the horizontal inset and available width for sections.
	// When the table's left or right border overlaps with cell borders (border-collapse),
	// the section expands to cover those border columns instead of being inset past them.
	sectionInsetX := border.Left + padding.Left
	childAvailWidthBase := resolvedInlineSize - parentDecorX
	if builder.grid.LeftEdgeHasOverlap {
		sectionInsetX = padding.Left // no extra left-border gap; cells share the border column
		childAvailWidthBase += border.Left
	}
	if builder.grid.RightEdgeHasOverlap {
		childAvailWidthBase += border.Right
	}

	rowIdx := 0
	for _, sectionNode := range builder.Sections() {
		childAvailWidth := childAvailWidthBase
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-padding.Top-padding.Bottom)

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetIsFixedInlineSize(true)

		childSpace := childSpaceBuilder.ToConstraintSpace()
		childAlgo := NewAlgorithm(sectionNode, childSpace)

		numRows := 0
		for range sectionNode.LayoutChildren() {
			numRows++
		}

		if sectionAlgo, ok := childAlgo.(*TableSectionAlgorithm); ok {
			sectionAlgo.Builder = builder
			sectionAlgo.ColumnWidths = builder.colWidths
			if rowIdx+numRows <= len(builder.grid.Rows) {
				sectionAlgo.RowsData = builder.grid.Rows[rowIdx : rowIdx+numRows]
			}
			rowIdx += numRows
		} else if rowAlgo, ok := childAlgo.(*TableRowAlgorithm); ok {
			rowAlgo.Builder = builder
			rowAlgo.ColumnWidths = builder.colWidths
			if rowIdx < len(builder.grid.Rows) {
				rowAlgo.RowData = builder.grid.Rows[rowIdx]
			}
			rowIdx++
		}

		childFrag := childAlgo.Layout()
		offset := Point{
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

	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		resolvedHeight := builder.CurrentBlockOffset()
		if comp.Height.Kind() == style.KindCells {
			resolvedHeight = max(resolvedHeight, comp.Height.CellsValue())
		}
		builder.SetBlockSize(resolvedHeight)
	}

	frag := builder.ToFragment()
	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

func (a *TableAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}

	builder := NewTableFragmentBuilder(a.Node, a.Space)
	for child := range a.Node.LayoutChildren() {
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
	builder.BuildGrid()

	comp := a.Node.Style()
	colMinMax := builder.colMinMax
	borderX := comp.Border.Widths()
	parentDecorX := borderX.Left + borderX.Right + comp.Padding.Left + comp.Padding.Right
	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}

	// Subtract collapsed border widths: only actual junction overlaps.
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

	a.Node.SetCachedMinMaxSizes(tableMinMax)
	return tableMinMax
}

// TableSectionAlgorithm implements the layout for table header, body, and footer groups.
type TableSectionAlgorithm struct {
	Node         Node
	Space        ConstraintSpace
	Builder      *TableFragmentBuilder
	ColumnWidths []int
	RowsData     []*tableRowGrid
}

func (a *TableSectionAlgorithm) Layout() *Fragment {
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}

	comp := a.Node.Style()
	border := comp.Border.Widths()
	padding := comp.Padding

	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	if a.Space.IsFixedInlineSize {
		builder.SetInlineSize(a.Space.AvailableSize.Width)
	}

	rowIdx := 0
	for rowNode := range a.Node.LayoutChildren() {
		childAvailWidth := max(0, a.Space.AvailableSize.Width-border.Left-border.Right-padding.Left-padding.Right)
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-(border.Top+border.Bottom+padding.Top+padding.Bottom))

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{Width: a.Space.AvailableSize.Width, Height: childAvailHeight})
		childSpaceBuilder.SetIsFixedInlineSize(true)

		childSpace := childSpaceBuilder.ToConstraintSpace()
		childAlgo := NewAlgorithm(rowNode, childSpace)

		if rowAlgo, ok := childAlgo.(*TableRowAlgorithm); ok {
			rowAlgo.Builder = a.Builder
			rowAlgo.ColumnWidths = a.ColumnWidths
			if rowIdx < len(a.RowsData) {
				rowAlgo.RowData = a.RowsData[rowIdx]
			}
		}

		childFrag := childAlgo.Layout()
		offset := Point{
			X: 0,
			Y: builder.CurrentBlockOffset(),
		}

		hasTopBorder := false
		hasBottomBorder := false
		if rowIdx < len(a.RowsData) {
			hasTopBorder = a.RowsData[rowIdx].HasTopBorder
			hasBottomBorder = a.RowsData[rowIdx].HasBottomBorder
		} else {
			hasTopBorder = rowNode.Style().Border.Edges.Top
			hasBottomBorder = rowNode.Style().Border.Edges.Bottom
		}

		shift := 0
		if a.Builder != nil {
			shift = a.Builder.AdjustRowOffset(hasTopBorder, hasBottomBorder)
		}
		offset.Y += shift
		builder.AdvanceBlockOffset(shift)

		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childFrag.Size.Height)
		rowIdx++
	}

	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		builder.SetBlockSize(builder.CurrentBlockOffset())
	}

	frag := builder.ToFragment()
	a.Node.SetCachedLayout(a.Space, frag)
	return frag
}

func (a *TableSectionAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	return MinMaxSizes{} // Table level handles sizing
}

// TableRowAlgorithm implements the DisplayTableRow layout.
type TableRowAlgorithm struct {
	Node         Node
	Space        ConstraintSpace
	Builder      *TableFragmentBuilder
	ColumnWidths []int
	RowData      *tableRowGrid
}

// anonymousTableSection represents a virtual layout group created to wrap direct table row children.
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

func (a *anonymousTableSection) LayoutChildren() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, child := range a.children {
			if !yield(child) {
				return
			}
		}
	}
}

func (a *anonymousTableSection) LogicalNode() any    { return nil }
func (a *anonymousTableSection) IsDirtyLayout() bool { return true }
func (a *anonymousTableSection) ClearDirtyLayout()   {}
func (a *anonymousTableSection) Fragment() *Fragment { return nil }

func (a *anonymousTableSection) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableSection) Layout() *Fragment {
	return nil
}

func (a *anonymousTableSection) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *anonymousTableSection) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *anonymousTableSection) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *anonymousTableSection) ComputeMinMaxSizes() MinMaxSizes {
	return MinMaxSizes{}
}

// anonymousTableRow represents a virtual layout row created to wrap contiguous runs of non-row content inside a table.
type anonymousTableRow struct {
	parent      Node
	children    []Node
	cachedSpace ConstraintSpace
}

var _ Node = (*anonymousTableRow)(nil)

func (a *anonymousTableRow) Style() *style.Computed {
	// Anonymous rows inherit styles from their parent table and have DisplayTableRow.
	s := *a.parent.Style()
	s.Display = style.DisplayTableRow
	s.Margin = style.EdgeValues[int]{}
	s.Padding = style.EdgeValues[int]{}
	s.Border = style.Border{}
	s.Width = style.Auto
	s.Height = style.Auto
	return &s
}

func (a *anonymousTableRow) LayoutChildren() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for _, child := range a.children {
			if !yield(child) {
				return
			}
		}
	}
}

func (a *anonymousTableRow) LogicalNode() any    { return nil }
func (a *anonymousTableRow) IsDirtyLayout() bool { return true }
func (a *anonymousTableRow) ClearDirtyLayout()   {}
func (a *anonymousTableRow) Fragment() *Fragment { return nil }

func (a *anonymousTableRow) CachedLayout(space ConstraintSpace) *Fragment {
	return nil
}

func (a *anonymousTableRow) Layout() *Fragment {
	// Handled directly by TableRowAlgorithm invocation in TableAlgorithm.Layout()
	return nil
}

func (a *anonymousTableRow) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	a.cachedSpace = space
}

func (a *anonymousTableRow) CachedMinMaxSizes() (MinMaxSizes, bool) {
	return MinMaxSizes{}, false
}

func (a *anonymousTableRow) SetCachedMinMaxSizes(sizes MinMaxSizes) {}

func (a *anonymousTableRow) ComputeMinMaxSizes() MinMaxSizes {
	return MinMaxSizes{}
}

func (a *TableRowAlgorithm) Layout() *Fragment {
	// Disable cache since layout depends on injected properties not in ConstraintSpace

	comp := a.Node.Style()
	padding := comp.Padding

	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	if a.Space.IsFixedInlineSize {
		builder.SetInlineSize(a.Space.AvailableSize.Width)
	}
	// Table rows use border-collapse: cells start at (0,0) within the row so
	// their borders physically overlap with the row's own borders at the same
	// pixel. The paint engine's resolver then merges them into junction glyphs.
	builder.currentBlockOffset = 0

	maxCellHeight := 0
	totalShiftX := 0

	if a.Builder != nil {
		a.Builder.ResetRow()
	}

	// Cells are placed starting at X=0, Y=0 within the row so that cell borders
	// physically overlap with the row's own border at the same pixel. The paint
	// engine's border-resolver then collapses them into junction glyphs.
	cellInsetX := 0

	// Layout cells
	if a.RowData != nil {
		for _, cell := range a.RowData.Cells {
			// Calculate cell available width based on ColSpan and ColumnWidths.
			// For spanning cells, subtract 1 for each internal collapsed junction
			// within the span (those junctions don't exist inside a spanning cell).
			cellWidth := 0
			for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
				if c < len(a.ColumnWidths) {
					cellWidth += a.ColumnWidths[c]
				}
			}
			if a.Builder != nil && cell.ColSpan > 1 {
				for j := cell.ColStart; j < cell.ColStart+cell.ColSpan-1; j++ {
					if j < len(a.Builder.grid.ColJunctionOverlap) && a.Builder.grid.ColJunctionOverlap[j] {
						cellWidth--
					}
				}
			}

			childMargin := cell.Node.Style().Margin
			cellAvailWidth := max(0, cellWidth-childMargin.Left-childMargin.Right)

			// Cells act as BFCs with rigid constraints passed by the row
			childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: cellAvailWidth, Height: a.Space.AvailableSize.Height})
			childSpaceBuilder.SetPercentageResolutionSize(Size{Width: cellWidth, Height: a.Space.AvailableSize.Height})
			childSpaceBuilder.SetIsFixedInlineSize(true)
			childSpace := childSpaceBuilder.ToConstraintSpace()

			childAlgo := NewAlgorithm(cell.Node, childSpace)
			childFrag := childAlgo.Layout()

			// X offset = row's left inset + sum of preceding column widths - collapsed borders.
			xOffset := cellInsetX
			for c := 0; c < cell.ColStart; c++ {
				if c < len(a.ColumnWidths) {
					xOffset += a.ColumnWidths[c]
				}
			}

			offset := Point{
				X: xOffset - totalShiftX,
				Y: builder.CurrentBlockOffset(),
			}

			hasLeftBorder := cell.Node.Style().Border.Edges.Left
			hasRightBorder := cell.Node.Style().Border.Edges.Right

			if a.Builder != nil {
				shift := a.Builder.GetCellShift(cell.ColStart, cell.ColSpan, hasLeftBorder, hasRightBorder)
				totalShiftX += shift
				offset.X -= shift
			}

			builder.AddChild(childFrag, offset)

			if childFrag.Size.Height > maxCellHeight {
				maxCellHeight = childFrag.Size.Height
			}
		}
	}

	// With border-collapse the row's bottom border is drawn at the bottom edge
	// of the cells (Y = maxCellHeight-1). Adding an extra border.Bottom would
	// push the row's bottom border one row below the cells, breaking collapse.
	builder.AdvanceBlockOffset(maxCellHeight + padding.Bottom)

	if a.Space.IsFixedBlockSize {
		builder.SetBlockSize(a.Space.AvailableSize.Height)
	} else {
		builder.SetBlockSize(builder.CurrentBlockOffset())
	}

	return builder.ToFragment()
}

func (a *TableRowAlgorithm) ComputeMinMaxSizes() MinMaxSizes {
	return MinMaxSizes{} // Intrinsic size is determined by the table pass
}
