package style

import "image/color"

type Cursor struct {
	// Shape is the visual shape of the terminal hardware cursor.
	Shape Optional[CursorShape]

	// Blink controls whether the cursor blinks. The default is true.
	Blink Optional[bool]

	// Color is the terminal hardware cursor color. The default is the foreground
	// color.
	Color Optional[color.Color]
}

// Merge returns a new [Cursor] where each set field in override overlays
// the corresponding field in s.
func (s Cursor) Merge(override Cursor) Cursor {
	return Cursor{
		Shape: s.Shape.Merge(override.Shape),
		Blink: s.Blink.Merge(override.Blink),
		Color: s.Color.Merge(override.Color),
	}
}

// CursorShape represents the visual shape of the terminal hardware cursor.
// Defined here (in the style package) so that it can be referenced by
// [Style.CursorShape] and [Computed.CursorShape] without creating an import
// cycle between the style and cursor packages.
type CursorShape uint8

const (
	// CursorShapeBlockBlink is a blinking block cursor (default).
	CursorBlock CursorShape = iota

	// CursorShapeBlockSteady is a steady (non-blinking) block cursor.
	CursorBar

	// CursorShapeUnderlineBlink is a blinking underline cursor.
	CursorUnderline
)
