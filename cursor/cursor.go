package cursor

import "github.com/masterkeysrd/kite/style"

// Shape represents the visual shape of the terminal cursor.
//
// Shape is a type alias for [style.CursorShape]. Both names refer to the same
// underlying type; existing code that references cursor.Shape or cursor.ShapeXxx
// constants continues to compile unchanged.
type Shape = style.CursorShape

// Cursor shape constants. These are aliases for the corresponding
// style.CursorShapeXxx constants provided for backward compatibility and
// ergonomic use in cursor-related code.
const (
	// ShapeBlockBlink is a blinking block cursor (default).
	ShapeBlockBlink = style.CursorShapeBlockBlink
	// ShapeBlockSteady is a steady (non-blinking) block cursor.
	ShapeBlockSteady = style.CursorShapeBlockSteady
	// ShapeBarBlink is a blinking vertical bar cursor.
	ShapeBarBlink = style.CursorShapeBarBlink
	// ShapeBarSteady is a steady vertical bar cursor.
	ShapeBarSteady = style.CursorShapeBarSteady
	// ShapeUnderlineBlink is a blinking underline cursor.
	ShapeUnderlineBlink = style.CursorShapeUnderlineBlink
	// ShapeUnderlineSteady is a steady underline cursor.
	ShapeUnderlineSteady = style.CursorShapeUnderlineSteady
)

// State holds the engine-side cursor model.
type State struct {
	// Visible determines if the hardware cursor is shown.
	Visible bool
	// X and Y are coordinates relative to the component's top-left corner.
	X, Y int
	// Shape determines the visual appearance of the cursor.
	Shape Shape
}

// Provider is implemented by render objects that want to control the terminal's
// hardware cursor. When a node with such a render object gains focus, the engine
// queries this interface to position the cursor.
type Provider interface {
	CursorState() State
}
