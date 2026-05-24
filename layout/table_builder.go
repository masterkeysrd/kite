package layout

import (
	"math"

	"github.com/masterkeysrd/kite/style"
)

// TableFragmentBuilder manages the mutable state during the two-pass
// table layout algorithm. It handles section grouping, column matrix math,
// and implicit border overlapping calculations.
type TableFragmentBuilder struct {
	node       Node
	space      ConstraintSpace
	boxBuilder *BoxFragmentBuilder

	// Grouping
	headers         []Node
	bodies          []Node
	footers         []Node
	currentAnonBody *anonymousTableSection

	// Sizing and Grid
	grid          tableGrid
	colMinMax     []MinMaxSizes
	resolvedWidth int
	colWidths     []int

	// Overlap tracking
	lastRowBorderBottom bool
	lastCellBorderRight map[int]bool // track right border of last cell per column
}

func NewTableFragmentBuilder(node Node, space ConstraintSpace) *TableFragmentBuilder {
	comp := node.Style()
	hasTopBorder := comp.Border.Edges.Top
	return &TableFragmentBuilder{
		node:                node,
		space:               space,
		boxBuilder:          NewBoxFragmentBuilder(node, space),
		lastCellBorderRight: make(map[int]bool),
		lastRowBorderBottom: hasTopBorder, // Snap first row to table top if it exists
	}
}

// -- Grouping --

func (b *TableFragmentBuilder) flushAnon() {
	if b.currentAnonBody != nil {
		b.bodies = append(b.bodies, b.currentAnonBody)
		b.currentAnonBody = nil
	}
}

func (b *TableFragmentBuilder) AddHeaderChild(node Node) {
	b.flushAnon()
	b.headers = append(b.headers, node)
}

func (b *TableFragmentBuilder) AddFooterChild(node Node) {
	b.flushAnon()
	b.footers = append(b.footers, node)
}

func (b *TableFragmentBuilder) AddBodyChild(node Node) {
	b.flushAnon()
	b.bodies = append(b.bodies, node)
}

func (b *TableFragmentBuilder) AddRowChild(node Node) {
	if b.currentAnonBody == nil {
		b.currentAnonBody = &anonymousTableSection{
			parent:  b.node,
			display: style.DisplayTableRowGroup,
		}
	}
	b.currentAnonBody.children = append(b.currentAnonBody.children, node)
}

func (b *TableFragmentBuilder) AddNonRowChild(node Node) {
	if b.currentAnonBody == nil {
		b.currentAnonBody = &anonymousTableSection{
			parent:  b.node,
			display: style.DisplayTableRowGroup,
		}
	}
	var anonRow *anonymousTableRow
	if len(b.currentAnonBody.children) > 0 {
		if last, ok := b.currentAnonBody.children[len(b.currentAnonBody.children)-1].(*anonymousTableRow); ok {
			anonRow = last
		}
	}
	if anonRow == nil {
		anonRow = &anonymousTableRow{parent: b.currentAnonBody}
		b.currentAnonBody.children = append(b.currentAnonBody.children, anonRow)
	}
	anonRow.children = append(anonRow.children, node)
}

func (b *TableFragmentBuilder) Sections() []Node {
	b.flushAnon()
	return append(append(b.headers, b.bodies...), b.footers...)
}

// -- Grid Sizing --

func (b *TableFragmentBuilder) BuildGrid(ctx *Context) {
	sections := b.Sections()

	// Flatten rows
	var rows []Node
	for _, section := range sections {
		for row := range section.LayoutChildren() {
			rows = append(rows, row)
		}
	}

	b.grid = b.buildTableGrid(rows)
	b.colMinMax = make([]MinMaxSizes, b.grid.NumCols)

	// First pass: single-column cells
	for _, row := range b.grid.Rows {
		for _, cell := range row.Cells {
			if cell.ColSpan == 1 {
				minMax := IntrinsicMinMaxSizes(ctx, cell.Node)
				b.colMinMax[cell.ColStart].Encompass(minMax)
			}
		}
	}

	// Second pass: multi-column cells
	for span := 2; span <= b.grid.NumCols; span++ {
		for _, row := range b.grid.Rows {
			for _, cell := range row.Cells {
				if cell.ColSpan == span {
					b.DistributeSpan(ctx, cell.Node, cell.ColStart, cell.ColSpan)
				}
			}
		}
	}
}

// DistributeSpan handles the complex math of stretching minimum widths across multiple columns.
func (b *TableFragmentBuilder) DistributeSpan(ctx *Context, cell Node, colIndex int, colSpan int) {
	minMax := IntrinsicMinMaxSizes(ctx, cell)

	currentMin := 0
	currentMax := 0
	for c := colIndex; c < colIndex+colSpan; c++ {
		currentMin += b.colMinMax[c].Min
		currentMax += b.colMinMax[c].Max
	}

	if minMax.Min > currentMin {
		extra := minMax.Min - currentMin
		perCol := extra / colSpan
		rem := extra % colSpan
		for c := colIndex; c < colIndex+colSpan; c++ {
			b.colMinMax[c].Min += perCol
			if rem > 0 {
				b.colMinMax[c].Min++
				rem--
			}
		}
	}

	if minMax.Max > currentMax {
		extra := minMax.Max - currentMax
		perCol := extra / colSpan
		rem := extra % colSpan
		for c := colIndex; c < colIndex+colSpan; c++ {
			b.colMinMax[c].Max += perCol
			if rem > 0 {
				b.colMinMax[c].Max++
				rem--
			}
		}
	}
}

func (b *TableFragmentBuilder) ResolveWidths(resolvedInlineSize int, parentDecorX int) {
	distributableWidth := max(0, resolvedInlineSize-parentDecorX)

	// Add back the collapsed border widths so the distributable budget reflects
	// the actual number of columns that the cells can fill.
	for _, overlap := range b.grid.ColJunctionOverlap {
		if overlap {
			distributableWidth++
		}
	}
	if b.grid.LeftEdgeHasOverlap {
		distributableWidth++
	}
	if b.grid.RightEdgeHasOverlap {
		distributableWidth++
	}

	b.resolvedWidth = resolvedInlineSize
	b.colWidths = b.distributeTableWidth(b.colMinMax, distributableWidth)
	b.boxBuilder.SetInlineSize(resolvedInlineSize)
}

func (b *TableFragmentBuilder) SetBlockSize(size int) {
	b.boxBuilder.SetBlockSize(size)
}

func (b *TableFragmentBuilder) CurrentBlockOffset() int {
	return b.boxBuilder.CurrentBlockOffset()
}

func (b *TableFragmentBuilder) AdvanceBlockOffset(amount int) {
	b.boxBuilder.AdvanceBlockOffset(amount)
}

// -- Overlap Math & Layout --

// ResetRow tracks the start of a new row for horizontal border collapsing.
func (b *TableFragmentBuilder) ResetRow() {
	b.lastCellBorderRight = make(map[int]bool)
}

// AdjustRowOffset handles the -1 Y coordinate adjustment for intersecting row borders.
func (b *TableFragmentBuilder) AdjustRowOffset(hasTopBorder, hasBottomBorder bool) int {
	shift := 0
	if b.lastRowBorderBottom && hasTopBorder {
		shift = -1
	}
	b.lastRowBorderBottom = hasBottomBorder
	return shift
}

// GetCellShift calculates the -1 X coordinate adjustment for intersecting cell borders
// and updates the tracking state for the next cell. It is only called for cells after
// the first one (colStart > 0); the row's own left border is handled by the caller as
// an initial X inset, not via this shift mechanism.
func (b *TableFragmentBuilder) GetCellShift(colStart int, colSpan int, hasLeftBorder, hasRightBorder bool) int {
	shift := 0
	if colStart > 0 {
		// Collapse this cell's left border with the previous cell's right border.
		if b.lastCellBorderRight[colStart-1] && hasLeftBorder {
			shift = 1
		}
	}

	// Update the tracking map for the rightmost column this cell occupies.
	b.lastCellBorderRight[colStart+colSpan-1] = hasRightBorder
	return shift
}

func (b *TableFragmentBuilder) ToFragment() *Fragment {
	return b.boxBuilder.ToFragment()
}

type tableCellGrid struct {
	Node     Node
	ColStart int
	ColSpan  int
	RowSpan  int
}

type tableRowGrid struct {
	Cells           []tableCellGrid
	HasTopBorder    bool
	HasBottomBorder bool
}

// tableGrid is the resolved cell-placement grid for a table.
// ColJunctionOverlap[j] is true if any row has a cell ending at column j
// with a right border AND a cell starting at column j+1 with a left border,
// meaning those two cell borders should be collapsed into one.
// LeftEdgeHasOverlap and RightEdgeHasOverlap record whether the table's
// own left/right border intersects with the first/last column's cell borders.
type tableGrid struct {
	Rows                []*tableRowGrid
	NumCols             int
	ColJunctionOverlap  []bool // len = NumCols-1
	LeftEdgeHasOverlap  bool
	RightEdgeHasOverlap bool
}

func (b *TableFragmentBuilder) buildTableGrid(rows []Node) tableGrid {
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
	for _, rowNode := range rows {
		row := &tableRowGrid{}

		rowStyle := rowNode.Style()
		row.HasTopBorder = rowStyle.Border.Edges.Top
		row.HasBottomBorder = rowStyle.Border.Edges.Bottom

		colIdx := 0
		cellChildren := rowNode.LayoutChildren()
		for cellNode := range cellChildren {
			if cellNode.Style().Display != style.DisplayTableCell {
				continue
			}

			cellStyle := cellNode.Style()
			if cellStyle.Border.Edges.Top {
				row.HasTopBorder = true
			}
			if cellStyle.Border.Edges.Bottom {
				row.HasBottomBorder = true
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

	// Compute per-column-junction and table-edge border overlaps.
	// A junction between column j and j+1 has an overlap when any row has a
	// cell whose rightmost column is j with a right border AND a cell whose
	// leftmost column is j+1 with a left border.
	numCols := grid.NumCols
	if numCols > 1 {
		grid.ColJunctionOverlap = make([]bool, numCols-1)
	}
	tableComp := b.node.Style()

	// Build a lookup: for each (rowIdx, colStart) → cell border info.
	type cellBorders struct{ left, right bool }
	cellBorderMap := make(map[[2]int]cellBorders)
	for ri, row := range grid.Rows {
		for _, cell := range row.Cells {
			edges := cell.Node.Style().Border.Edges
			key := [2]int{ri, cell.ColStart}
			cellBorderMap[key] = cellBorders{left: edges.Left, right: edges.Right}
		}
	}

	// Scan junctions and table edges.
	for ri, row := range grid.Rows {
		for _, cell := range row.Cells {
			info := cellBorderMap[[2]int{ri, cell.ColStart}]
			rightCol := cell.ColStart + cell.ColSpan - 1

			// Left edge: table left border vs this cell's left border.
			if cell.ColStart == 0 && tableComp.Border.Edges.Left && info.left {
				grid.LeftEdgeHasOverlap = true
			}
			// Right edge: table right border vs this cell's right border.
			if rightCol == numCols-1 && tableComp.Border.Edges.Right && info.right {
				grid.RightEdgeHasOverlap = true
			}

			// Internal junction to the right of this cell.
			if info.right && rightCol < numCols-1 {
				// Find the cell that starts at rightCol+1 in the same row.
				neighbour := cellBorderMap[[2]int{ri, rightCol + 1}]
				if neighbour.left {
					grid.ColJunctionOverlap[rightCol] = true
				}
			}
		}
	}

	return grid
}

func (b *TableFragmentBuilder) distributeTableWidth(colMinMax []MinMaxSizes, availableWidth int) []int {
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
