package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// Table represents a table element.
type Table struct {
	elementBase[Table]
}

var _ Element = (*Table)(nil)

// NewTable creates a new table element.
func NewTable(doc dom.Document) *Table {
	t := &Table{}
	t.initBase(doc.CreateElement("table", t), t, style.Style{
		Display: style.Some(style.DisplayTable),
	})
	return t
}

// TableRow represents a table row element.
type TableRow struct {
	elementBase[TableRow]
}

var _ Element = (*TableRow)(nil)

// NewTableRow creates a new table row element.
func NewTableRow(doc dom.Document) *TableRow {
	tr := &TableRow{}
	tr.initBase(doc.CreateElement("tr", tr), tr, style.Style{
		Display: style.Some(style.DisplayTableRow),
	})
	return tr
}

// TableCell represents a table cell element.
type TableCell struct {
	elementBase[TableCell]
	colSpan int
	rowSpan int
}

var _ Element = (*TableCell)(nil)

// NewTableCell creates a new table cell element.
func NewTableCell(doc dom.Document) *TableCell {
	td := &TableCell{
		colSpan: 1,
		rowSpan: 1,
	}
	td.initBase(doc.CreateElement("td", td), td, style.Style{
		Display: style.Some(style.DisplayTableCell),
	})
	return td
}

// SetColSpan sets the number of columns the cell should span.
func (td *TableCell) SetColSpan(span int) *TableCell {
	if span < 1 {
		span = 1
	}
	td.colSpan = span
	if ro := td.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout)
	}
	return td
}

// ColSpan returns the number of columns the cell spans.
func (td *TableCell) ColSpan() int {
	return td.colSpan
}

// SetRowSpan sets the number of rows the cell should span.
func (td *TableCell) SetRowSpan(span int) *TableCell {
	if span < 1 {
		span = 1
	}
	td.rowSpan = span
	if ro := td.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout)
	}
	return td
}

// RowSpan returns the number of rows the cell spans.
func (td *TableCell) RowSpan() int {
	return td.rowSpan
}
