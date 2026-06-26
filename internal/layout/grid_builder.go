package layout

import (
	"sync"

	"github.com/masterkeysrd/kite/style"
)

// gridItem tracks a node's resolved placement in the grid matrix.
type gridItem struct {
	node     Node
	colStart int
	colSpan  int
	rowStart int
	rowSpan  int
}

var gridBuilderPool = sync.Pool{
	New: func() any {
		return &GridBuilder{
			items: make([]gridItem, 0, 16),
		}
	},
}

// AcquireGridBuilder gets a builder from the pool and initializes it.
func AcquireGridBuilder(node Node, space ConstraintSpace) *GridBuilder {
	b := gridBuilderPool.Get().(*GridBuilder)
	b.Init(node, space)
	return b
}

// ReleaseGridBuilder returns a builder to the pool.
func ReleaseGridBuilder(b *GridBuilder) {
	b.node = nil
	for i := range b.items {
		b.items[i].node = nil
	}
	b.items = b.items[:0]
	gridBuilderPool.Put(b)
}

// GridBuilder handles the auto-placement algorithm and track sizing for CSS Grid.
type GridBuilder struct {
	node     Node
	space    ConstraintSpace
	items    []gridItem
	occupied []uint64 // bitset: (row * occCols + col)
	occCols  int
	occRows  int
	maxCol   int
	maxRow   int

	colTemplate []style.GridTrackSize
	rowTemplate []style.GridTrackSize
}

// Init initializes the GridBuilder for the given node and space.
// This allows for stack allocation or reuse.
func (b *GridBuilder) Init(node Node, space ConstraintSpace) {
	comp := node.Style()
	b.node = node
	b.space = space
	b.colTemplate = comp.GridTemplateColumns
	b.rowTemplate = comp.GridTemplateRows

	// Reset or pre-allocate
	if b.items == nil {
		b.items = make([]gridItem, 0, 16)
	} else {
		b.items = b.items[:0]
	}

	cols := len(b.colTemplate)
	if cols == 0 {
		cols = 4
	}
	rows := len(b.rowTemplate)
	if rows == 0 {
		rows = 4
	}

	if b.occupied == nil || b.occCols*b.occRows < cols*rows {
		b.occupied = make([]uint64, (cols*rows+63)/64)
		b.occCols = cols
		b.occRows = rows
	} else {
		// Clear bitset
		for i := range b.occupied {
			b.occupied[i] = 0
		}
	}

	b.maxCol = 0
	b.maxRow = 0
}

// NewGridBuilder creates a new GridBuilder for the given node and space.
func NewGridBuilder(node Node, space ConstraintSpace) *GridBuilder {
	b := &GridBuilder{}
	b.Init(node, space)
	return b
}

// ResolveTrackSizes resolves fixed and percentage track sizes against the available space.
func ResolveTrackSizes(templates []style.GridTrackSize, available int, gap int) []int {
	if len(templates) == 0 {
		return nil
	}

	resolved := make([]int, len(templates))

	totalGap := (len(templates) - 1) * gap
	contentAvailable := max(0, available-totalGap)

	for i, t := range templates {
		switch t.Kind() {
		case style.KindCells:
			resolved[i] = t.CellsValue()
		case style.KindPercent:
			if available < InfiniteBlockSize {
				resolved[i] = int(float32(contentAvailable) * t.PercentValue() / 100.0)
			} else {
				resolved[i] = 0
			}
		default:
			resolved[i] = 0
		}
	}

	return resolved
}
func (b *GridBuilder) PlaceItems() {
	// Pass 1: Fully explicit (both row and col defined)
	for child := b.node.FirstLayoutChild(); child != nil; child = b.node.NextLayoutSibling(child) {
		comp := child.Style()
		startCol, colSpan := resolvePlacement(comp.GridColumn)
		startRow, rowSpan := resolvePlacement(comp.GridRow)

		if startCol >= 0 && startRow >= 0 {
			b.placeItem(child, startCol, colSpan, startRow, rowSpan)
		}
	}

	// Pass 2: Implicit (one or both are auto)
	cursorCol, cursorRow := 0, 0
	for child := b.node.FirstLayoutChild(); child != nil; child = b.node.NextLayoutSibling(child) {
		if b.isPlaced(child) {
			continue
		}
		b.resolveAndPlaceImplicit(child, &cursorCol, &cursorRow)
	}
}

func (b *GridBuilder) isPlaced(node Node) bool {
	for i := range b.items {
		if b.items[i].node == node {
			return true
		}
	}
	return false
}

func (b *GridBuilder) ensureCapacity(col, row int) {
	if col < b.occCols && row < b.occRows {
		return
	}

	newCols := b.occCols
	if col >= newCols {
		newCols = max(newCols*2, col+1)
	}
	newRows := b.occRows
	if row >= newRows {
		newRows = max(newRows*2, row+1)
	}

	needed := (newCols*newRows + 63) / 64
	newOccupied := make([]uint64, needed)

	for r := 0; r < b.occRows; r++ {
		for c := 0; c < b.occCols; c++ {
			idx := r*b.occCols + c
			if (b.occupied[idx/64] & (1 << (uint(idx) % 64))) != 0 {
				newIdx := r*newCols + c
				newOccupied[newIdx/64] |= (1 << (uint(newIdx) % 64))
			}
		}
	}

	b.occupied = newOccupied
	b.occCols = newCols
	b.occRows = newRows
}

func (b *GridBuilder) markOccupied(col, row, colSpan, rowSpan int) {
	b.ensureCapacity(col+colSpan-1, row+rowSpan-1)
	for r := row; r < row+rowSpan; r++ {
		for c := col; c < col+colSpan; c++ {
			idx := r*b.occCols + c
			b.occupied[idx/64] |= (1 << (uint(idx) % 64))
		}
	}
}

func (b *GridBuilder) isOccupied(col, row int) bool {
	if col >= b.occCols || row >= b.occRows {
		return false
	}
	idx := row*b.occCols + col
	return (b.occupied[idx/64] & (1 << (uint(idx) % 64))) != 0
}

func (b *GridBuilder) placeItem(node Node, colStart, colSpan, rowStart, rowSpan int) {
	b.items = append(b.items, gridItem{
		node:     node,
		colStart: colStart,
		colSpan:  colSpan,
		rowStart: rowStart,
		rowSpan:  rowSpan,
	})

	b.markOccupied(colStart, rowStart, colSpan, rowSpan)

	if colStart+colSpan > b.maxCol {
		b.maxCol = colStart + colSpan
	}
	if rowStart+rowSpan > b.maxRow {
		b.maxRow = rowStart + rowSpan
	}
}

func resolvePlacement(p style.GridPlacement) (start, span int) {
	if p.Start != 0 && p.End != 0 {
		start = p.Start - 1
		span = p.End - p.Start
		if span <= 0 {
			span = 1
		}
		return start, span
	}

	span = p.Span
	if span <= 0 {
		span = 1
	}

	if p.Start != 0 {
		start = p.Start - 1
	} else if p.End != 0 {
		start = p.End - 1 - span
	} else {
		start = -1
	}

	return start, span
}

func (b *GridBuilder) resolveAndPlaceImplicit(node Node, cursorCol, cursorRow *int) {
	comp := node.Style()
	startCol, colSpan := resolvePlacement(comp.GridColumn)
	startRow, rowSpan := resolvePlacement(comp.GridRow)

	tmplCols := len(b.colTemplate)

	if startCol >= 0 && startRow < 0 {
		r := 0
		for {
			if b.fits(startCol, r, colSpan, rowSpan) {
				startRow = r
				break
			}
			r++
		}
	} else if startRow >= 0 && startCol < 0 {
		c := 0
		for {
			if b.fits(c, startRow, colSpan, rowSpan) {
				startCol = c
				break
			}
			c++
		}
	} else {
		c, r := *cursorCol, *cursorRow
		for {
			if b.fits(c, r, colSpan, rowSpan) {
				startCol, startRow = c, r
				break
			}
			c++
			if tmplCols > 0 && c+colSpan > tmplCols {
				c = 0
				r++
			}
		}
		*cursorCol = c
		*cursorRow = r
	}

	b.placeItem(node, startCol, colSpan, startRow, rowSpan)
}

func (b *GridBuilder) fits(col, row, colSpan, rowSpan int) bool {
	if col < 0 || row < 0 {
		return false
	}
	for r := row; r < row+rowSpan; r++ {
		for c := col; c < col+colSpan; c++ {
			if b.isOccupied(c, r) {
				return false
			}
		}
	}
	return true
}
