package layout

import (
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

// GridBuilder handles the auto-placement algorithm and track sizing for CSS Grid.
type GridBuilder struct {
	node     Node
	space    ConstraintSpace
	items    []*gridItem
	occupied map[Point]bool
	maxCol   int
	maxRow   int
}

// NewGridBuilder creates a new GridBuilder for the given node and space.
func NewGridBuilder(node Node, space ConstraintSpace) *GridBuilder {
	return &GridBuilder{
		node:     node,
		space:    space,
		occupied: make(map[Point]bool),
	}
}

// ResolveTrackSizes resolves fixed and percentage track sizes against the available space.
// Auto and Fr tracks are not fully resolved here (they require a measure pass),
// but we determine the initial sizes for fixed-length tracks.
func ResolveTrackSizes(templates []style.GridTrackSize, available int, gap int) []int {
	if len(templates) == 0 {
		return nil
	}

	resolved := make([]int, len(templates))

	// Calculate total gaps
	totalGap := (len(templates) - 1) * gap
	contentAvailable := max(0, available-totalGap)

	for i, t := range templates {
		switch t.Kind() {
		case style.KindCells:
			resolved[i] = t.CellsValue()
		case style.KindPercent:
			resolved[i] = int(float32(contentAvailable) * t.PercentValue() / 100.0)
		default:
			// Auto and Fr are handled later in the full algorithm (TSK-054).
			resolved[i] = 0
		}
	}

	return resolved
}

// PlaceItems performs the two-pass grid placement algorithm.
func (b *GridBuilder) PlaceItems() {
	var children []Node
	for child := range b.node.LayoutChildren() {
		children = append(children, child)
	}

	// Pass 1: Fully explicit (both row and col defined)
	for _, child := range children {
		comp := child.Style()
		startCol, _ := resolvePlacement(comp.GridColumn)
		startRow, _ := resolvePlacement(comp.GridRow)

		if startCol >= 0 && startRow >= 0 {
			item := b.resolveExplicitPlacement(child)
			b.placeItem(item)
		}
	}

	// Pass 2: Implicit (one or both are auto)
	cursorCol, cursorRow := 0, 0
	for _, child := range children {
		if b.isPlaced(child) {
			continue
		}
		item := b.resolveImplicitPlacement(child, &cursorCol, &cursorRow)
		b.placeItem(item)
	}
}

func (b *GridBuilder) isPlaced(node Node) bool {
	for _, item := range b.items {
		if item.node == node {
			return true
		}
	}
	return false
}

func (b *GridBuilder) placeItem(item *gridItem) {
	b.items = append(b.items, item)

	// Mark all occupied cells
	for r := item.rowStart; r < item.rowStart+item.rowSpan; r++ {
		for c := item.colStart; c < item.colStart+item.colSpan; c++ {
			b.occupied[Point{X: c, Y: r}] = true
			if c >= b.maxCol {
				b.maxCol = c + 1
			}
			if r >= b.maxRow {
				b.maxRow = r + 1
			}
		}
	}
}

func (b *GridBuilder) resolveExplicitPlacement(node Node) *gridItem {
	comp := node.Style()
	startCol, colSpan := resolvePlacement(comp.GridColumn)
	startRow, rowSpan := resolvePlacement(comp.GridRow)
	return &gridItem{
		node:     node,
		colStart: startCol,
		colSpan:  colSpan,
		rowStart: startRow,
		rowSpan:  rowSpan,
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

func (b *GridBuilder) resolveImplicitPlacement(node Node, cursorCol, cursorRow *int) *gridItem {
	comp := node.Style()

	// Normalized placement info
	startCol, colSpan := resolvePlacement(comp.GridColumn)
	startRow, rowSpan := resolvePlacement(comp.GridRow)

	tmplCols := len(b.node.Style().GridTemplateColumns)

	if startCol >= 0 && startRow < 0 {
		// Column is fixed, row is auto
		r := 0
		for {
			if b.fits(startCol, r, colSpan, rowSpan) {
				startRow = r
				break
			}
			r++
		}
	} else if startRow >= 0 && startCol < 0 {
		// Row is fixed, column is auto
		c := 0
		for {
			if b.fits(c, startRow, colSpan, rowSpan) {
				startCol = c
				break
			}
			c++
		}
	} else {
		// Fully auto
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
		// Update cursor for next item
		*cursorCol = c
		*cursorRow = r
	}

	return &gridItem{
		node:     node,
		colStart: startCol,
		colSpan:  colSpan,
		rowStart: startRow,
		rowSpan:  rowSpan,
	}
}

func (b *GridBuilder) fits(col, row, colSpan, rowSpan int) bool {
	if col < 0 || row < 0 {
		return false
	}
	for r := row; r < row+rowSpan; r++ {
		for c := col; c < col+colSpan; c++ {
			if b.occupied[Point{X: c, Y: r}] {
				return false
			}
		}
	}
	return true
}
