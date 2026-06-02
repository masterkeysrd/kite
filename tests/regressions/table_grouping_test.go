package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestTableGrouping_Sorting(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Create sections in "wrong" order: tfoot, tbody, thead
	table := element.Table(
		element.TFoot(element.TR(element.TD("Footer"))),
		element.TBody(element.TR(element.TD("Body"))),
		element.THead(element.TR(element.TD("Header"))),
	)
	env.Mount(table)
	env.RenderFrame()

	// Verify structure and that first column cells have equal width.
	testenv.Expect(t, table).
		ToHaveTableStructure([]string{"thead", "tbody", "tfoot"}).
		CellsInColumn(0).ToHaveEqualWidth()
}

func TestTableGrouping_AnonymousGroups(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Table with direct TRs
	table := element.Table(
		element.TR(element.TD("Row 1")),
		element.TR(element.TD("Row 2")),
	)
	env.Mount(table)
	env.RenderFrame()

	tableFrag := env.RenderObject(table).Fragment()
	// Should have 1 child (anonymous tbody)
	if len(tableFrag.Children) != 1 {
		t.Fatalf("expected 1 anonymous section fragment, got %d", len(tableFrag.Children))
	}

	tbodyFrag := tableFrag.Children[0].Fragment
	if tbodyFrag.Node.Style().Display != style.DisplayTableRowGroup {
		t.Errorf("expected anonymous section to be tbody (%d), got %d", style.DisplayTableRowGroup, tbodyFrag.Node.Style().Display)
	}

	if len(tbodyFrag.Children) != 2 {
		t.Errorf("expected 2 row fragments in anonymous tbody, got %d", len(tbodyFrag.Children))
	}
}

func TestTableGrouping_ColumnSynchronization(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Table with wide cell in tbody and narrow cell in thead
	table := element.Table(
		element.THead(element.TR(element.TD("H"))),
		element.TBody(element.TR(element.TD("Wide Content"))),
	)
	env.Mount(table)
	env.RenderFrame()

	tableFrag := env.RenderObject(table).Fragment()
	// Total width should be at least 12 (width of "Wide Content")
	if tableFrag.Size.Width < 12 {
		t.Errorf("expected table width to be at least 12, got %d", tableFrag.Size.Width)
	}

	theadFrag := tableFrag.Children[0].Fragment
	tbodyFrag := tableFrag.Children[1].Fragment

	headerRowFrag := theadFrag.Children[0].Fragment
	bodyRowFrag := tbodyFrag.Children[0].Fragment

	headerCellFrag := headerRowFrag.Children[0].Fragment
	bodyCellFrag := bodyRowFrag.Children[0].Fragment

	if headerCellFrag.Size.Width != bodyCellFrag.Size.Width {
		t.Errorf("expected header cell and body cell to have same width, got %d and %d",
			headerCellFrag.Size.Width, bodyCellFrag.Size.Width)
	}
}
