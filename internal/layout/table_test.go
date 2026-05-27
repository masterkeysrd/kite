package layout

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

type mockTableCellNode struct {
	mockNode
	colSpan int
	rowSpan int
}

func (m *mockTableCellNode) ColSpan() int     { return m.colSpan }
func (m *mockTableCellNode) RowSpan() int     { return m.rowSpan }
func (m *mockTableCellNode) LogicalNode() any { return m }

func TestTableLayout_UniformGrid(t *testing.T) {
	cellStyle := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(10),
		Height:  style.Cells(2),
	}

	linkSiblings := func(nodes ...Node) Node {
		for i := 0; i < len(nodes)-1; i++ {
			switch n := nodes[i].(type) {
			case *mockTableCellNode:
				n.nextSibling = nodes[i+1]
			case *mockNode:
				n.nextSibling = nodes[i+1]
			}
		}
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}

	c11 := &mockTableCellNode{mockNode: mockNode{style: cellStyle}}
	c12 := &mockTableCellNode{mockNode: mockNode{style: cellStyle}}
	row1 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c11, c12),
	}

	cellStyle2 := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(15),
		Height:  style.Cells(2),
	}
	c21 := &mockTableCellNode{mockNode: mockNode{style: cellStyle2}}
	c22 := &mockTableCellNode{mockNode: mockNode{style: cellStyle}}
	row2 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c21, c22),
	}

	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
			Height:  style.Auto,
		},
		firstChild: linkSiblings(row1, row2),
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{}
	frag := algo.Layout(nil, table, space)

	if frag.Size.Width != 25 {
		t.Errorf("expected table width 25, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 4 {
		t.Errorf("expected table height 4, got %d", frag.Size.Height)
	}

	row1Frag := frag.Children[0].Fragment.Children[0].Fragment
	if row1Frag.Children[0].Offset.X != 0 {
		t.Errorf("cell 1,1 X expected 0, got %d", row1Frag.Children[0].Offset.X)
	}
	if row1Frag.Children[1].Offset.X != 15 {
		t.Errorf("cell 1,2 X expected 15, got %d", row1Frag.Children[1].Offset.X)
	}
}

func TestTableLayout_ColSpan(t *testing.T) {
	linkSiblings := func(nodes ...Node) Node {
		for i := 0; i < len(nodes)-1; i++ {
			switch n := nodes[i].(type) {
			case *mockTableCellNode:
				n.nextSibling = nodes[i+1]
			case *mockNode:
				n.nextSibling = nodes[i+1]
			}
		}
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}

	cellStyle10 := &style.Computed{Display: style.DisplayTableCell, Width: style.Cells(10), Height: style.Cells(1)}
	cellStyle30 := &style.Computed{Display: style.DisplayTableCell, Width: style.Cells(30), Height: style.Cells(1)}

	c11 := &mockTableCellNode{mockNode: mockNode{style: cellStyle10}}
	c12 := &mockTableCellNode{mockNode: mockNode{style: cellStyle10}}
	row1 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c11, c12),
	}

	c21 := &mockTableCellNode{
		mockNode: mockNode{style: cellStyle30},
		colSpan:  2,
	}
	row2 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c21),
	}

	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
			Height:  style.Auto,
		},
		firstChild: linkSiblings(row1, row2),
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{}
	frag := algo.Layout(nil, table, space)

	if frag.Size.Width != 30 {
		t.Errorf("expected table width 30, got %d", frag.Size.Width)
	}
}

func TestTableLayout_FaultTolerance_AnonymousRow(t *testing.T) {
	linkSiblings := func(nodes ...Node) Node {
		for i := 0; i < len(nodes)-1; i++ {
			switch n := nodes[i].(type) {
			case *mockTableCellNode:
				n.nextSibling = nodes[i+1]
			case *mockNode:
				n.nextSibling = nodes[i+1]
			}
		}
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}

	cellStyle10 := &style.Computed{Display: style.DisplayTableCell, Width: style.Cells(10), Height: style.Cells(1)}
	cellStyle20 := &style.Computed{Display: style.DisplayTableCell, Width: style.Cells(20), Height: style.Cells(1)}

	c1 := &mockTableCellNode{mockNode: mockNode{style: cellStyle10}}
	c2 := &mockTableCellNode{mockNode: mockNode{style: cellStyle20}}

	// Table directly containing cells (no rows)
	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
			Height:  style.Auto,
		},
		firstChild: linkSiblings(c1, c2),
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{}
	frag := algo.Layout(nil, table, space)

	if frag.Size.Width != 30 {
		t.Errorf("expected table width 30, got %d", frag.Size.Width)
	}

	// Should have 1 anonymous row child
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 row child, got %d", len(frag.Children))
	}

	rowFrag := frag.Children[0].Fragment.Children[0].Fragment
	if len(rowFrag.Children) != 2 {
		t.Fatalf("expected 2 cells in row, got %d", len(rowFrag.Children))
	}

	if rowFrag.Children[0].Offset.X != 0 {
		t.Errorf("cell 1 X offset expected 0, got %d", rowFrag.Children[0].Offset.X)
	}
	if rowFrag.Children[1].Offset.X != 10 {
		t.Errorf("cell 2 X offset expected 10, got %d", rowFrag.Children[1].Offset.X)
	}
}

func TestTableLayout_BorderOverlap(t *testing.T) {
	linkSiblings := func(nodes ...Node) Node {
		for i := 0; i < len(nodes)-1; i++ {
			switch n := nodes[i].(type) {
			case *mockTableCellNode:
				n.nextSibling = nodes[i+1]
			case *mockNode:
				n.nextSibling = nodes[i+1]
			}
		}
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}

	cellStyleBordered := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(10),
		Height:  style.Auto,
		Border: style.Border{
			Edges: style.EdgeAll(true),
		},
	}

	// Two bordered cells side by side.
	// Expected behavior: The second cell's X coordinate is offset by -1.
	c1 := &mockTableCellNode{mockNode: mockNode{style: cellStyleBordered}}
	c2 := &mockTableCellNode{mockNode: mockNode{style: cellStyleBordered}}

	row1 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTableRow,
			Border: style.Border{
				Edges: style.EdgeAll(true),
			},
		},
		firstChild: linkSiblings(c1, c2),
	}

	row2 := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTableRow,
			Border: style.Border{
				Edges: style.EdgeAll(true),
			},
		},
		firstChild: linkSiblings(&mockTableCellNode{mockNode: mockNode{style: cellStyleBordered}}),
	}

	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
		},
		firstChild: linkSiblings(row1, row2),
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	frag := (&TableAlgorithm{}).Layout(nil, table, space)

	// Anonymous section
	tbodyFrag := frag.Children[0].Fragment

	// Row 1
	row1Frag := tbodyFrag.Children[0]
	if row1Frag.Offset.Y != 0 {
		t.Errorf("expected row1 Y offset 0, got %d", row1Frag.Offset.Y)
	}

	cell1 := row1Frag.Fragment.Children[0]
	cell2 := row1Frag.Fragment.Children[1]

	// With border collapse, cells start at X=0 within the row (no inset).
	// Cell1's left border overlaps the row's left border at X=0.
	if cell1.Offset.X != 0 {
		t.Errorf("expected cell1 X offset 0, got %d", cell1.Offset.X)
	}

	// Cell2 shifts left by 1 because cell1's right border and cell2's left
	// border occupy the same terminal column (border collapse).
	if cell2.Offset.X != 9 {
		t.Errorf("expected cell2 X offset 9, got %d", cell2.Offset.X)
	}

	// Row 2
	row2Frag := tbodyFrag.Children[1]
	// With border-collapse, the row height equals maxCellHeight (borders are at
	// the cell's top/bottom edges, not added separately). Row1 cell height = 2
	// (border.top + border.bottom, no content). Row1 height = 2.
	// Row2.top border shares Row1.bottom border → AdjustRowOffset returns -1.
	// Row2.Y = 2 + (-1) = 1.
	if row2Frag.Offset.Y != 1 {
		t.Errorf("expected row2 Y offset 1 due to border-collapse overlap, got %d", row2Frag.Offset.Y)
	}
}

// TestTableLayout_ColSpanBorderCollapse verifies that a spanning cell's width
// is reduced by the number of internal collapsed junctions within its span.
// Without the fix the cell would be 1 cell too wide (junction overlap not
// subtracted), causing its right border to overflow past the table edge.
func TestTableLayout_ColSpanBorderCollapse(t *testing.T) {
	linkSiblings := func(nodes ...Node) Node {
		for i := 0; i < len(nodes)-1; i++ {
			switch n := nodes[i].(type) {
			case *mockTableCellNode:
				n.nextSibling = nodes[i+1]
			case *mockNode:
				n.nextSibling = nodes[i+1]
			}
		}
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}

	// Bordered cells in the first row establish the ColJunctionOverlap.
	borderedCell := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(10),
		Border:  style.Border{Edges: style.EdgeAll(true)},
	}
	spanningCell := &style.Computed{
		Display: style.DisplayTableCell,
		Border:  style.Border{Edges: style.EdgeAll(true)},
	}

	c11 := &mockTableCellNode{mockNode: mockNode{style: borderedCell}}
	c12 := &mockTableCellNode{mockNode: mockNode{style: borderedCell}}
	row1 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c11, c12),
	}

	// Row 2: single cell spanning both columns.
	c21 := &mockTableCellNode{
		mockNode: mockNode{style: spanningCell},
		colSpan:  2,
	}
	row2 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c21),
	}

	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
		},
		firstChild: linkSiblings(row1, row2),
	}

	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	frag := (&TableAlgorithm{}).Layout(nil, table, space)

	// Table width: col0(10) + col1(10) - 1(junction) = 19.
	if frag.Size.Width != 19 {
		t.Fatalf("expected table width 19, got %d", frag.Size.Width)
	}

	// The spanning cell in row2 must fit exactly within the table:
	// width = colWidths[0]+colWidths[1] - 1(junction) = table.width = 19.
	tbodyFrag := frag.Children[0].Fragment
	row2Frag := tbodyFrag.Children[1].Fragment
	if len(row2Frag.Children) != 1 {
		t.Fatalf("expected 1 child in row2, got %d", len(row2Frag.Children))
	}
	spanFrag := row2Frag.Children[0].Fragment
	if spanFrag.Size.Width != 19 {
		t.Errorf("spanning cell width expected 19 (colWidths sum minus junction), got %d", spanFrag.Size.Width)
	}
	// Spanning cell must start at X=0 (no shift since colStart=0).
	if row2Frag.Children[0].Offset.X != 0 {
		t.Errorf("spanning cell X offset expected 0, got %d", row2Frag.Children[0].Offset.X)
	}
}
