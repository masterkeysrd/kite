package paint

import (
	"image/color"

	"github.com/masterkeysrd/kite/layout"
)

// Surface is the drawing target provided by the backend. All paint
// operations are expressed in absolute terminal-cell coordinates.
//
// Implementations must be safe to call from the main goroutine only; no
// concurrent access is expected.
type Surface interface {
	// CellAt returns the cell at absolute position (x, y). If the position
	// is out of the surface's bounds, an empty Cell is returned.
	CellAt(x, y int) Cell

	// Set writes cell c into the absolute position (x, y). Calls outside
	// the surface's bounds are silently ignored.
	Set(x, y int, c Cell)

	// Bounds returns the total drawable area of this surface in absolute
	// terminal-cell coordinates.
	Bounds() layout.Rect

	// Clip returns a new Surface whose drawable area is restricted to the
	// intersection of this surface's bounds and r. Coordinates passed to
	// Set on the returned surface are still in absolute terms so that the
	// clip can be composed without coordinate translations.
	Clip(r layout.Rect) Surface
}

// CellAttrs holds the text-attribute bitmask for a terminal cell.
type CellAttrs uint8

const (
	// AttrBold makes the cell's text bold.
	AttrBold CellAttrs = 1 << iota
	// AttrItalic makes the cell's text italic.
	AttrItalic
	// AttrUnderline underlines the cell's text.
	AttrUnderline
	// AttrInverse swaps the foreground and background colors.
	AttrInverse
)

// BorderStyle selects the pre-defined glyph set for a border.
type BorderStyle uint8

const (
	// BorderNone draws no border.
	BorderNone BorderStyle = 0
	// BorderAscii draws an ASCII border.
	BorderAscii BorderStyle = 1
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
	// Content is the string to be rendered in this cell. It may be empty, but
	// must not be nil. The string may contain multiple Unicode code points, but
	// must not contain any combining characters. The width of the cell is
	// determined by the number of columns needed to render Content, which may
	// be zero for an empty string.
	Content string

	Width int
	// FG is the foreground (text) color. A nil value means terminal default.
	FG color.Color
	// BG is the background color. A nil value means terminal default.
	BG color.Color
	// Attrs is the bitmask of text attributes applied to this cell.
	Attrs CellAttrs
	// BorderStyle indicates the type of border glyph in this cell.
	BorderStyle BorderStyle
}
