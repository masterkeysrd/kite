package element

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func TestTableCell_Span(t *testing.T) {
	doc := dom.NewDocument()
	td := NewTableCell(doc)

	if td.ColSpan() != 1 {
		t.Errorf("expected default ColSpan 1, got %d", td.ColSpan())
	}
	if td.RowSpan() != 1 {
		t.Errorf("expected default RowSpan 1, got %d", td.RowSpan())
	}

	td.SetColSpan(3)
	if td.ColSpan() != 3 {
		t.Errorf("expected ColSpan 3, got %d", td.ColSpan())
	}

	td.SetRowSpan(2)
	if td.RowSpan() != 2 {
		t.Errorf("expected RowSpan 2, got %d", td.RowSpan())
	}

	// Test negative or zero span
	td.SetColSpan(0)
	if td.ColSpan() != 1 {
		t.Errorf("expected ColSpan 1 when setting to 0, got %d", td.ColSpan())
	}

	td.SetRowSpan(-1)
	if td.RowSpan() != 1 {
		t.Errorf("expected RowSpan 1 when setting to -1, got %d", td.RowSpan())
	}
}
