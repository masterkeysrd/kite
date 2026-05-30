package cursor

import "github.com/masterkeysrd/kite/style"

// State holds the engine-side cursor model.
type State struct {
	// Visible determines if the hardware cursor is shown.
	Visible bool
	// X and Y are coordinates relative to the component's top-left corner.
	X, Y int

	// Style defines the cursor's visual appearance.
	Style style.Cursor
}

// Provider is implemented by render objects that want to control the terminal's
// hardware cursor. When a node with such a render object gains focus, the engine
// queries this interface to position the cursor.
type Provider interface {
	CursorState() State
}
