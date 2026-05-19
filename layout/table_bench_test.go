package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func BenchmarkTableLayout_50x10(b *testing.B) {
	// Create a 50 rows x 10 columns table
	linkSiblings := func(nodes []Node) Node {
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

	cellStyle := &style.Computed{
		Display: style.DisplayTableCell,
		Width:   style.Cells(5),
		Height:  style.Cells(1),
	}

	var rows []Node
	for range 50 {
		var cells []Node
		for range 10 {
			cells = append(cells, &mockTableCellNode{mockNode: mockNode{style: cellStyle}})
		}
		row := &mockNode{
			style:      &style.Computed{Display: style.DisplayTableRow},
			firstChild: linkSiblings(cells),
		}
		rows = append(rows, row)
	}

	table := &mockNode{
		style: &style.Computed{
			Display: style.DisplayTable,
			Width:   style.Percent(100),
			Height:  style.Auto,
		},
		firstChild: linkSiblings(rows),
	}

	space := NewConstraintSpaceBuilder(Size{Width: 200, Height: 1000}).ToConstraintSpace()

	for b.Loop() {
		// Clear cache manually to benchmark pure layout time
		for _, rowNode := range rows {
			rowNode.ClearDirtyLayout()
			rn := rowNode.(*mockNode)
			rn.cachedFragment = nil

			for cellNode := rn.firstChild; cellNode != nil; cellNode = cellNode.(*mockTableCellNode).nextSibling {
				cellNode.ClearDirtyLayout()
				cn := cellNode.(*mockTableCellNode)
				cn.cachedFragment = nil
			}
		}
		table.ClearDirtyLayout()
		table.cachedFragment = nil

		algo := &TableAlgorithm{Node: table, Space: space}
		_ = algo.Layout()
	}
}
