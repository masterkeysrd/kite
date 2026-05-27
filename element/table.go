package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/internal/render"
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

// TableHeaderElement represents a table header group element.
type TableHeaderElement struct {
	elementBase[TableHeaderElement]
}

var _ Element = (*TableHeaderElement)(nil)

// NewTableHeader creates a new table header group element.
func NewTableHeader(doc dom.Document) *TableHeaderElement {
	th := &TableHeaderElement{}
	th.initBase(doc.CreateElement("thead", th), th, style.Style{
		Display: style.Some(style.DisplayTableHeaderGroup),
	})
	return th
}

// THead creates a new table header group element with the given children.
func THead(children ...any) *TableHeaderElement {
	th := NewTableHeader(orphanDocument)
	processChildren(th, children)
	return th
}

// TableHeader is an alias for THead.
func TableHeader(children ...any) *TableHeaderElement {
	return THead(children...)
}

// TableBodyElement represents a table body group element.
type TableBodyElement struct {
	elementBase[TableBodyElement]
}

var _ Element = (*TableBodyElement)(nil)

// NewTableBody creates a new table body group element.
func NewTableBody(doc dom.Document) *TableBodyElement {
	tb := &TableBodyElement{}
	tb.initBase(doc.CreateElement("tbody", tb), tb, style.Style{
		Display: style.Some(style.DisplayTableRowGroup),
	})
	return tb
}

// TBody creates a new table body group element with the given children.
func TBody(children ...any) *TableBodyElement {
	tb := NewTableBody(orphanDocument)
	processChildren(tb, children)
	return tb
}

// TableBody is an alias for TBody.
func TableBody(children ...any) *TableBodyElement {
	return TBody(children...)
}

// TableFooterElement represents a table footer group element.
type TableFooterElement struct {
	elementBase[TableFooterElement]
}

var _ Element = (*TableFooterElement)(nil)

// NewTableFooter creates a new table footer group element.
func NewTableFooter(doc dom.Document) *TableFooterElement {
	tf := &TableFooterElement{}
	tf.initBase(doc.CreateElement("tfoot", tf), tf, style.Style{
		Display: style.Some(style.DisplayTableFooterGroup),
	})
	return tf
}

// TFoot creates a new table footer group element with the given children.
func TFoot(children ...any) *TableFooterElement {
	tf := NewTableFooter(orphanDocument)
	processChildren(tf, children)
	return tf
}

// TableFooter is an alias for TFoot.
func TableFooter(children ...any) *TableFooterElement {
	return TFoot(children...)
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
