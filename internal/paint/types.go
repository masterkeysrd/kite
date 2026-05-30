package paint

import (
	"image/color"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
)

// Surface is the drawing target provided by the backend. All paint
// operations are expressed in absolute terminal-cell coordinates.
//
// Implementations must be safe to call from the main goroutine only; no
// concurrent access is expected.
type Surface interface {
	// Set writes cell c into position (x, y).
	Set(x, y int, c Cell)

	// CellAt returns the cell at absolute position (x, y).
	CellAt(x, y int) Cell

	// Bounds returns the total drawable area of this surface in absolute
	// terminal-cell coordinates.
	Bounds() geom.Rect

	// Clip returns a new Surface whose drawable area is restricted to the
	// intersection of this surface's bounds and r. Coordinates passed to
	// Set on the returned surface are still in absolute terms so that the
	// clip can be composed without coordinate translations.
	Clip(r geom.Rect) Surface
}

// CellAttrs holds the text-attribute bitmask for a terminal cell.
type CellAttrs = backend.CellStyle

const (
	// AttrBold makes the cell's text bold.
	AttrBold = backend.CellBold
	// AttrItalic makes the cell's text italic.
	AttrItalic = backend.CellItalic
	// AttrUnderline underlines the cell's text.
	AttrUnderline = backend.CellUnderline
	// AttrInverse swaps the foreground and background colors.
	AttrInverse = backend.CellReverse
)

// BorderStyle selects the pre-defined glyph set for a border.
type BorderStyle uint8

const (
	// BorderNone draws no border.
	BorderNone BorderStyle = 0
	// BorderASCII draws an ASCII border.
	BorderASCII BorderStyle = 1
	// BorderRounded draws a rounded border.
	BorderRounded BorderStyle = 2
	// BorderSingle draws a single-line border.
	BorderSingle BorderStyle = 3
	// BorderDouble draws a double-line border.
	BorderDouble BorderStyle = 4
	// BorderThick draws a heavy-line border.
	BorderThick BorderStyle = 5
)

// Cell represents the content and styling of a single terminal cell.
type Cell struct {
	backend.Cell
	// Content is the string to be rendered in this cell. It may be empty, but
	// must not be nil. The string may contain multiple Unicode code points, but
	// must not contain any combining characters. The width of the cell is
	// determined by the number of columns needed to render Content, which may
	// be zero for an empty string.
	// BorderStyle indicates the type of border glyph in this cell.
	BorderStyle BorderStyle
}

// SelectionRect represents a physical rectangle of selected content.
type SelectionRect struct {
	Rect geom.Rect
	Fg   color.Color
	Bg   color.Color
}
