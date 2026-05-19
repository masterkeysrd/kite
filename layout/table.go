package layout

import (
	"math"

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
	border := comp.Border.Width
	padding := comp.Padding
	insetX := border.Left + padding.Left
	parentDecorX := border.Left + border.Right + padding.Left + padding.Right
	parentDecorY := border.Top + border.Bottom + padding.Top + padding.Bottom

	// Pass 1: Grid Sizing
	grid := buildTableGrid(a.Node)
	colMinMax := computeColumnMinMax(grid)

	// Resolve the table's inline size
	var resolvedInlineSize int
	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}
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
	distributableWidth := max(0, resolvedInlineSize-parentDecorX)
	colWidths := distributeTableWidth(colMinMax, distributableWidth)

	// Setup Builder
	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	builder.SetInlineSize(resolvedInlineSize)

	// Pass 2: Layout Rows
	contentWidth := distributableWidth
	children := a.Node.LayoutChildren()
	rowIdx := 0

	for rowNode := range children {
		if rowNode.Style().Display != style.DisplayTableRow {
			continue // Skip non-row children (fault tolerance in future)
		}

		childMargin := rowNode.Style().Margin
		childAvailWidth := max(0, contentWidth-childMargin.Left-childMargin.Right)
		childAvailHeight := max(0, a.Space.AvailableSize.Height-builder.CurrentBlockOffset()-childMargin.Top-childMargin.Bottom-parentDecorY)

		childSpaceBuilder := NewConstraintSpaceBuilder(Size{Width: childAvailWidth, Height: childAvailHeight})
		childSpaceBuilder.SetPercentageResolutionSize(Size{Width: contentWidth, Height: childAvailHeight})
		childSpaceBuilder.SetIsFixedInlineSize(true)

		childSpace := childSpaceBuilder.ToConstraintSpace()
		childAlgo := NewAlgorithm(rowNode, childSpace)

		if rowAlgo, ok := childAlgo.(*TableRowAlgorithm); ok {
			rowAlgo.ColumnWidths = colWidths
			if rowIdx < len(grid.Rows) {
				rowAlgo.RowData = grid.Rows[rowIdx]
			}
		}

		childFrag := childAlgo.Layout()
		offset := Point{
			X: insetX + childMargin.Left,
			Y: builder.CurrentBlockOffset() + childMargin.Top,
		}
		builder.AddChild(childFrag, offset)
		builder.AdvanceBlockOffset(childMargin.Top + childFrag.Size.Height + childMargin.Bottom)
		rowIdx++
	}

	builder.AdvanceBlockOffset(border.Bottom + padding.Bottom)

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

	grid := buildTableGrid(a.Node)
	colMinMax := computeColumnMinMax(grid)

	var tableMinMax MinMaxSizes
	for _, m := range colMinMax {
		tableMinMax.Min += m.Min
		tableMinMax.Max += m.Max
	}

	comp := a.Node.Style()
	parentDecorX := comp.Border.Width.Left + comp.Border.Width.Right + comp.Padding.Left + comp.Padding.Right
	tableMinMax.Min += parentDecorX
	tableMinMax.Max += parentDecorX

	a.Node.SetCachedMinMaxSizes(tableMinMax)
	return tableMinMax
}

// TableRowAlgorithm implements the DisplayTableRow layout.
type TableRowAlgorithm struct {
	Node         Node
	Space        ConstraintSpace
	ColumnWidths []int
	RowData      *tableRowGrid
}

func (a *TableRowAlgorithm) Layout() *Fragment {
	// Disable cache since layout depends on injected properties not in ConstraintSpace

	comp := a.Node.Style()
	border := comp.Border.Width
	padding := comp.Padding
	insetX := border.Left + padding.Left

	builder := NewBoxFragmentBuilder(a.Node, a.Space)
	if a.Space.IsFixedInlineSize {
		builder.SetInlineSize(a.Space.AvailableSize.Width)
	}

	maxCellHeight := 0

	// Layout cells
	if a.RowData != nil {
		for _, cell := range a.RowData.Cells {
			// Calculate cell available width based on ColSpan and ColumnWidths
			cellWidth := 0
			for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
				if c < len(a.ColumnWidths) {
					cellWidth += a.ColumnWidths[c]
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

			// Calculate X offset
			xOffset := insetX
			for c := 0; c < cell.ColStart; c++ {
				if c < len(a.ColumnWidths) {
					xOffset += a.ColumnWidths[c]
				}
			}
			xOffset += childMargin.Left

			offset := Point{
				X: xOffset,
				Y: builder.CurrentBlockOffset() + childMargin.Top,
			}
			builder.AddChild(childFrag, offset)

			if childFrag.Size.Height+childMargin.Top+childMargin.Bottom > maxCellHeight {
				maxCellHeight = childFrag.Size.Height + childMargin.Top + childMargin.Bottom
			}
		}
	}

	builder.AdvanceBlockOffset(maxCellHeight + border.Bottom + padding.Bottom)

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

// -- Grid Utilities --

type tableCellGrid struct {
	Node     Node
	ColStart int
	ColSpan  int
	RowSpan  int
}

type tableRowGrid struct {
	Cells []tableCellGrid
}

type tableGrid struct {
	Rows    []*tableRowGrid
	NumCols int
}

func buildTableGrid(tableNode Node) tableGrid {
	var grid tableGrid
	occupied := make(map[int]map[int]bool) // row -> col -> true

	isOccupied := func(r, c int) bool {
		if occupied[r] == nil {
			return false
		}
		return occupied[r][c]
	}

	markOccupied := func(r, c int) {
		if occupied[r] == nil {
			occupied[r] = make(map[int]bool)
		}
		occupied[r][c] = true
	}

	rowIdx := 0
	children := tableNode.LayoutChildren()
	for rowNode := range children {
		if rowNode.Style().Display != style.DisplayTableRow {
			continue
		}

		row := &tableRowGrid{}
		colIdx := 0
		cellChildren := rowNode.LayoutChildren()
		for cellNode := range cellChildren {
			if cellNode.Style().Display != style.DisplayTableCell {
				continue
			}

			// Find next free column
			for isOccupied(rowIdx, colIdx) {
				colIdx++
			}

			colSpan := getColSpan(cellNode)
			rowSpan := getRowSpan(cellNode)

			row.Cells = append(row.Cells, tableCellGrid{
				Node:     cellNode,
				ColStart: colIdx,
				ColSpan:  colSpan,
				RowSpan:  rowSpan,
			})

			for r := 0; r < rowSpan; r++ {
				for c := 0; c < colSpan; c++ {
					markOccupied(rowIdx+r, colIdx+c)
				}
			}

			colIdx += colSpan
			if colIdx > grid.NumCols {
				grid.NumCols = colIdx
			}
		}
		grid.Rows = append(grid.Rows, row)
		rowIdx++
	}

	return grid
}

func computeColumnMinMax(grid tableGrid) []MinMaxSizes {
	cols := make([]MinMaxSizes, grid.NumCols)

	// First pass: single-column cells
	for _, row := range grid.Rows {
		for _, cell := range row.Cells {
			if cell.ColSpan == 1 {
				minMax := IntrinsicMinMaxSizes(cell.Node)
				cols[cell.ColStart].Encompass(minMax)
			}
		}
	}

	// Second pass: multi-column cells (distribute equally to spanned columns)
	for span := 2; span <= grid.NumCols; span++ {
		for _, row := range grid.Rows {
			for _, cell := range row.Cells {
				if cell.ColSpan == span {
					minMax := IntrinsicMinMaxSizes(cell.Node)

					// Calculate current min/max of the spanned columns
					currentMin := 0
					currentMax := 0
					for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
						currentMin += cols[c].Min
						currentMax += cols[c].Max
					}

					// Distribute excess min
					if minMax.Min > currentMin {
						extra := minMax.Min - currentMin
						perCol := extra / cell.ColSpan
						rem := extra % cell.ColSpan
						for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
							cols[c].Min += perCol
							if rem > 0 {
								cols[c].Min++
								rem--
							}
						}
					}

					// Distribute excess max
					if minMax.Max > currentMax {
						extra := minMax.Max - currentMax
						perCol := extra / cell.ColSpan
						rem := extra % cell.ColSpan
						for c := cell.ColStart; c < cell.ColStart+cell.ColSpan; c++ {
							cols[c].Max += perCol
							if rem > 0 {
								cols[c].Max++
								rem--
							}
						}
					}
				}
			}
		}
	}

	return cols
}

func distributeTableWidth(colMinMax []MinMaxSizes, availableWidth int) []int {
	widths := make([]int, len(colMinMax))
	totalMin := 0
	totalMax := 0

	for i, m := range colMinMax {
		widths[i] = m.Min
		totalMin += m.Min
		totalMax += m.Max
	}

	extra := availableWidth - totalMin
	if extra <= 0 {
		return widths
	}

	distributableMax := totalMax - totalMin
	if distributableMax > 0 {
		// Distribute proportionally up to max
		for i, m := range colMinMax {
			maxDiff := m.Max - m.Min
			if maxDiff > 0 {
				portion := int(math.Round(float64(extra) * float64(maxDiff) / float64(distributableMax)))
				// don't exceed max
				added := min(portion, maxDiff)
				widths[i] += added
				extra -= added
			}
		}
	}

	// If still extra, distribute equally
	if extra > 0 && len(widths) > 0 {
		perCol := extra / len(widths)
		rem := extra % len(widths)
		for i := range widths {
			widths[i] += perCol
			if rem > 0 {
				widths[i]++
				rem--
			}
		}
	}

	return widths
}
