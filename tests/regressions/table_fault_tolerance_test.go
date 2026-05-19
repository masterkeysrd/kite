package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
)

func TestRegression_TableFaultToleranceAndInvalidation(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Create a malformed table: direct child is TableCell (no TableRow)
	table := element.NewTable(doc)
	td := element.NewTableCell(doc)
	textNode := element.NewText(doc, "Hello")
	td.AppendChild(textNode)
	table.AppendChild(td)
	doc.AppendChild(table)

	// First frame: Should group into anonymous row and measure correctly
	eng.Frame()

	// Check table layout width
	tableFrag := table.RenderObject().Fragment()
	if tableFrag == nil {
		t.Fatal("expected table to have a fragment")
	}

	// "Hello" is 5 cells wide
	initialWidth := tableFrag.Size.Width
	if initialWidth != 5 {
		t.Fatalf("expected initial table width to be 5, got %d", initialWidth)
	}

	// Now modify the text inside the cell
	textNode.SetData("Hello World")

	// The text update should dirty the DOM and propagate DirtyLayout to the table
	eng.Frame()

	tableFrag2 := table.RenderObject().Fragment()
	if tableFrag2 == nil {
		t.Fatal("expected table to have a fragment")
	}

	// "Hello World" is 11 cells wide
	updatedWidth := tableFrag2.Size.Width
	if updatedWidth != 11 {
		t.Fatalf("expected updated table width to be 11, got %d", updatedWidth)
	}

	// Ensure the malformed table actually created an anonymous row in the fragment tree
	if len(tableFrag2.Children) != 1 {
		t.Fatalf("expected 1 child fragment (the anonymous row), got %d", len(tableFrag2.Children))
	}

	rowFrag := tableFrag2.Children[0].Fragment
	if rowFrag == nil {
		t.Fatal("expected a valid row fragment")
	}
	if len(rowFrag.Children) != 1 {
		t.Fatalf("expected 1 cell fragment inside the row, got %d", len(rowFrag.Children))
	}

	// Cell node should match the anonymous row's child
	cellFrag := rowFrag.Children[0].Fragment
	if cellFrag.Node.LogicalNode() != td {
		t.Fatal("expected cell fragment to belong to td")
	}
}
