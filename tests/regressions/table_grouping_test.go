package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestTableGrouping_Sorting(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Create sections in "wrong" order: tfoot, tbody, thead
	table := element.Table(
		element.TFoot(element.TR(element.TD("Footer"))),
		element.TBody(element.TR(element.TD("Body"))),
		element.THead(element.TR(element.TD("Header"))),
	)
	doc.AppendChild(table)

	eng.Frame()

	tableFrag := table.RenderObject().Fragment()
	if len(tableFrag.Children) != 3 {
		t.Fatalf("expected 3 section fragments, got %d", len(tableFrag.Children))
	}

	// Verify order: thead, tbody, tfoot
	if tableFrag.Children[0].Fragment.Node.Style().Display != style.DisplayTableHeaderGroup {
		t.Errorf("expected first child to be thead (%d), got %d", style.DisplayTableHeaderGroup, tableFrag.Children[0].Fragment.Node.Style().Display)
	}
	if tableFrag.Children[1].Fragment.Node.Style().Display != style.DisplayTableRowGroup {
		t.Errorf("expected second child to be tbody (%d), got %d", style.DisplayTableRowGroup, tableFrag.Children[1].Fragment.Node.Style().Display)
	}
	if tableFrag.Children[2].Fragment.Node.Style().Display != style.DisplayTableFooterGroup {
		t.Errorf("expected third child to be tfoot (%d), got %d", style.DisplayTableFooterGroup, tableFrag.Children[2].Fragment.Node.Style().Display)
	}

	// Verify Y offsets (order)
	if tableFrag.Children[0].Offset.Y >= tableFrag.Children[1].Offset.Y {
		t.Errorf("thead should be above tbody")
	}
	if tableFrag.Children[1].Offset.Y >= tableFrag.Children[2].Offset.Y {
		t.Errorf("tbody should be above tfoot")
	}
}

func TestTableGrouping_AnonymousGroups(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Table with direct TRs
	table := element.Table(
		element.TR(element.TD("Row 1")),
		element.TR(element.TD("Row 2")),
	)
	doc.AppendChild(table)

	eng.Frame()

	tableFrag := table.RenderObject().Fragment()
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
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Table with wide cell in tbody and narrow cell in thead
	table := element.Table(
		element.THead(element.TR(element.TD("H"))),
		element.TBody(element.TR(element.TD("Wide Content"))),
	)
	doc.AppendChild(table)

	eng.Frame()

	tableFrag := table.RenderObject().Fragment()
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
