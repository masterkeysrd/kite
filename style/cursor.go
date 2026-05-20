package style

// CursorShape represents the visual shape of the terminal hardware cursor.
// Defined here (in the style package) so that it can be referenced by
// [Style.CursorShape] and [Computed.CursorShape] without creating an import
// cycle between the style and cursor packages.
type CursorShape uint8

const (
	// CursorShapeBlockBlink is a blinking block cursor (default).
	CursorShapeBlockBlink CursorShape = iota
	// CursorShapeBlockSteady is a steady (non-blinking) block cursor.
	CursorShapeBlockSteady
	// CursorShapeBarBlink is a blinking vertical bar cursor.
	CursorShapeBarBlink
	// CursorShapeBarSteady is a steady vertical bar cursor.
	CursorShapeBarSteady
	// CursorShapeUnderlineBlink is a blinking underline cursor.
	CursorShapeUnderlineBlink
	// CursorShapeUnderlineSteady is a steady underline cursor.
	CursorShapeUnderlineSteady
)
