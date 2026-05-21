package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

// TestRegression_InputCursorPosition verifies that the hardware cursor is placed
// correctly inside the input field's content box, accounting for border and padding.
func TestRegression_InputCursorPosition(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello")
	inp.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Border:  style.SingleBorder().Some(),
		Padding: style.Some(style.Edges(0, 2)), // 2 cells padding-left/right
	})

	eng.Mount(inp)
	eng.Frame()

	// Initial focus (autofocus) should land on 'inp'.
	// Origin should be (0,0) as it's the root.
	// Border: Top=1, Left=1. Padding: Left=2.
	// Content start X = 1 (border) + 2 (padding) = 3.
	// Content start Y = 1 (border) + 0 (padding) = 1.

	// Ensure we are at the start of the buffer.
	inp.Buffer().MoveToStart()
	inp.SyncBuffer()
	eng.Frame()

	// "hello" at offset 0 should be at (3, 1).
	if b.Cursor.X != 3 || b.Cursor.Y != 1 {
		t.Errorf("cursor position with 'hello' at offset 0: got (%d,%d), want (3,1)", b.Cursor.X, b.Cursor.Y)
	}

	// "hello" at offset 5 should be at (3+5, 1).
	inp.Buffer().MoveToEnd()
	inp.SyncBuffer()
	eng.Frame()

	if b.Cursor.X != 8 || b.Cursor.Y != 1 { // 3 + 5 = 8
		t.Errorf("cursor position with 'hello' at offset 5: got (%d,%d), want (8,1)", b.Cursor.X, b.Cursor.Y)
	}
}

// TestRegression_EmptyInputCursorPosition verifies that the cursor is placed
// inside the content box even when the input is empty.
func TestRegression_EmptyInputCursorPosition(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Border:  style.SingleBorder().Some(),
		Padding: style.Some(style.Edges(0, 1)), // 1 cell padding
	})

	eng.Mount(inp)
	eng.Frame()

	// Content start X = 1 (border) + 1 (padding) = 2.
	// Content start Y = 1 (border) + 0 (padding) = 1.
	if b.Cursor.X != 2 || b.Cursor.Y != 1 {
		t.Errorf("empty input cursor position: got (%d,%d), want (2,1)", b.Cursor.X, b.Cursor.Y)
	}
}
