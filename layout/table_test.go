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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{Node: table, Space: space}
	frag := algo.Layout()

	if frag.Size.Width != 25 {
		t.Errorf("expected table width 25, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 4 {
		t.Errorf("expected table height 4, got %d", frag.Size.Height)
	}

	row1Frag := frag.Children[0].Fragment
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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{Node: table, Space: space}
	frag := algo.Layout()

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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &TableAlgorithm{Node: table, Space: space}
	frag := algo.Layout()

	if frag.Size.Width != 30 {
		t.Errorf("expected table width 30, got %d", frag.Size.Width)
	}

	// Should have 1 anonymous row child
	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 row child, got %d", len(frag.Children))
	}

	rowFrag := frag.Children[0].Fragment
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

func TestTableLayout_StretchAndShrink(t *testing.T) {
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
