package engine

import (
	"github.com/masterkeysrd/kite/style"
)

// cursorState holds the engine-side cursor model. The engine emits the
// corresponding OSC/CSI sequences during the Sync phase.
type cursorState struct {
	visible bool
	pos     position
	shape   style.CursorShape
}

// position is an (x, y) coordinate in terminal-cell space.
type position struct{ x, y int }

// CursorController is the public API through which widgets drive cursor state.
// Callers obtained via Engine.Cursor() are the only authorised writers;
// the engine applies the state during the Sync phase.
//
// All methods must be called from the main goroutine.
type CursorController struct {
	state *cursorState
}

// Show sets the cursor visibility. Pass true to show the cursor, false to
// hide it. Changes take effect at the next Sync phase.
func (c *CursorController) Show(visible bool) {
	c.state.visible = visible
}

// SetPos sets the cursor position to the given terminal-cell coordinates.
// Changes take effect at the next Sync phase.
func (c *CursorController) SetPos(x, y int) {
	c.state.pos = position{x: x, y: y}
}

// SetShape sets the cursor visual shape. Blink rate is terminal-controlled
// when a *Blink shape is set; the engine emits no software blink ticker.
// Changes take effect at the next Sync phase.
func (c *CursorController) SetShape(shape style.CursorShape) {
	c.state.shape = shape
}
