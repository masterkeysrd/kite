package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// TableElement represents a table element.
type TableElement struct {
	elementBase[TableElement]
}

var _ Element = (*TableElement)(nil)

// NewTable creates a new table element.
func NewTable(doc dom.Document) *TableElement {
	t := &TableElement{}
	t.initBase(doc.CreateElement("table", t), t, style.Style{
		Display: style.Some(style.DisplayTable),
	})
	return t
}

// Table creates a new table element with the given children.
func Table(children ...any) *TableElement {
	t := NewTable(orphanDocument)
	processChildren(t, children)
	return t
}

// TableRowElement represents a table row element.
type TableRowElement struct {
	elementBase[TableRowElement]
}

var _ Element = (*TableRowElement)(nil)

// NewTableRow creates a new table row element.
func NewTableRow(doc dom.Document) *TableRowElement {
	tr := &TableRowElement{}
	tr.initBase(doc.CreateElement("tr", tr), tr, style.Style{
		Display: style.Some(style.DisplayTableRow),
	})
	return tr
}

// TR creates a new table row element with the given children.
func TR(children ...any) *TableRowElement {
	tr := NewTableRow(orphanDocument)
	processChildren(tr, children)
	return tr
}

// TableCellElement represents a table cell element.
type TableCellElement struct {
	elementBase[TableCellElement]
	colSpan int
	rowSpan int
}

var _ Element = (*TableCellElement)(nil)

// NewTableCell creates a new table cell element.
func NewTableCell(doc dom.Document) *TableCellElement {
	td := &TableCellElement{
		colSpan: 1,
		rowSpan: 1,
	}
	td.initBase(doc.CreateElement("td", td), td, style.Style{
		Display: style.Some(style.DisplayTableCell),
	})
	return td
}

// TD creates a new table cell element with the given children.
func TD(children ...any) *TableCellElement {
	td := NewTableCell(orphanDocument)
	processChildren(td, children)
	return td
}

// SetColSpan sets the number of columns the cell should span.
func (td *TableCellElement) SetColSpan(span int) *TableCellElement {
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
func (td *TableCellElement) ColSpan() int {
	return td.colSpan
}

// SetRowSpan sets the number of rows the cell should span.
func (td *TableCellElement) SetRowSpan(span int) *TableCellElement {
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
func (td *TableCellElement) RowSpan() int {
	return td.rowSpan
}
