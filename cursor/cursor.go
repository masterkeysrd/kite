// Package cursor provides a unified abstraction for terminal hardware cursor
// management. It decouples the engine's focus management from the render
// objects' local coordinate systems.
package cursor

// Shape represents the visual shape of the terminal cursor.
type Shape uint8

const (
	// ShapeBlockBlink is a blinking block cursor (default).
	ShapeBlockBlink Shape = iota
	// ShapeBlockSteady is a steady (non-blinking) block cursor.
	ShapeBlockSteady
	// ShapeBarBlink is a blinking vertical bar cursor.
	ShapeBarBlink
	// ShapeBarSteady is a steady vertical bar cursor.
	ShapeBarSteady
	// ShapeUnderlineBlink is a blinking underline cursor.
	ShapeUnderlineBlink
	// ShapeUnderlineSteady is a steady underline cursor.
	ShapeUnderlineSteady
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
