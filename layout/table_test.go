package layout

import (
	"testing"

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
	// Test case 1: A 2x2 table correctly aligns cell widths so columns are uniform.

	// Cell styles: fixed size
	cellStyle := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(10),
		Height:  style.Cells(2),
	}

	// Better setup function for siblings
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

	// Make the second row cells slightly different width to test encompass/alignment
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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{Node: table, Space: space}
	frag := algo.Layout()

	// Column widths should be max of each column
	// Col 0: max(10, 15) = 15
	// Col 1: max(10, 10) = 10
	// Total width = 25

	if frag.Size.Width != 25 {
		t.Errorf("expected table width 25, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 4 {
		t.Errorf("expected table height 4, got %d", frag.Size.Height)
	}

	// Check child positions (rows)
	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(frag.Children))
	}

	row1Frag := frag.Children[0].Fragment
	if len(row1Frag.Children) != 2 {
		t.Fatalf("expected 2 cells in row 1, got %d", len(row1Frag.Children))
	}

	// Cell 11 should have width 15 and x offset 0
	if row1Frag.Children[0].Offset.X != 0 {
		t.Errorf("cell 1,1 X expected 0, got %d", row1Frag.Children[0].Offset.X)
	}

	// Cell 12 should have x offset 15
	if row1Frag.Children[1].Offset.X != 15 {
		t.Errorf("cell 1,2 X expected 15, got %d", row1Frag.Children[1].Offset.X)
	}
}

func TestTableLayout_ColSpan(t *testing.T) {
	// Test case 2: A cell with ColSpan(2) correctly forces the combined width of the two columns to encompass its text.

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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{Node: table, Space: space}
	frag := algo.Layout()

	// Row 1 has col 0=10, col 1=10
	// Row 2 has colSpan=2, width=30
	// Current total min = 20, max = 20.
	// Spanned cell min=30, max=30.
	// Extra min=10, Extra max=10.
	// Distributed to col 0 and 1 equally: 5 to each.
	// So Col 0 = 15, Col 1 = 15. Total = 30.

	if frag.Size.Width != 30 {
		t.Errorf("expected table width 30, got %d", frag.Size.Width)
	}

	row1Frag := frag.Children[0].Fragment
	if row1Frag.Children[0].Offset.X != 0 {
		t.Errorf("expected cell 1,1 offset 0, got %d", row1Frag.Children[0].Offset.X)
	}
	if row1Frag.Children[1].Offset.X != 15 {
		t.Errorf("expected cell 1,2 offset 15, got %d", row1Frag.Children[1].Offset.X)
	}

	row2Frag := frag.Children[1].Fragment
	if row2Frag.Children[0].Fragment.Size.Width != 30 {
		t.Errorf("expected colspanned cell width 30, got %d", row2Frag.Children[0].Fragment.Size.Width)
	}
}

func TestTableLayout_StretchAndShrink(t *testing.T) {
	// Verify that a DisplayTable correctly shrinks to fit its content if Width is Auto, or stretches if Width is Percent.

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
	c11 := &mockTableCellNode{mockNode: mockNode{style: cellStyle10}}
	row1 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c11),
	}

	// 1. Auto width -> shrink to content
	tableAuto := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Auto,
		},
		firstChild: linkSiblings(row1),
	}
	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	frag1 := (&TableAlgorithm{Node: tableAuto, Space: space}).Layout()
	if frag1.Size.Width != 10 {
		t.Errorf("expected Auto table to shrink to 10, got %d", frag1.Size.Width)
	}

	// 2. Percent width -> stretch
	c11_2 := &mockTableCellNode{mockNode: mockNode{style: cellStyle10}}
	row1_2 := &mockNode{
		style:      &style.Computed{Display: style.DisplayTableRow},
		firstChild: linkSiblings(c11_2),
	}
	tablePercent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Percent(100),
		},
		firstChild: linkSiblings(row1_2),
	}
	frag2 := (&TableAlgorithm{Node: tablePercent, Space: space}).Layout()
	if frag2.Size.Width != 100 {
		t.Errorf("expected Percent table to stretch to 100, got %d", frag2.Size.Width)
	}
}
