package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/style"
)

// TestTextArea_Regression_ScrollCursorPos verifies that the hardware cursor
// stays aligned with the text content when the textarea is scrolled.
// This handles the regression where content was panned incorrectly relative
// to the cursor due to border/padding insets in the paint engine's clamping logic.
func TestTextArea_Regression_ScrollCursorPos(t *testing.T) {
	b := mock.New(80, 26)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// 5 lines of text
	initialText := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	txa := element.NewTextArea(eng.Document(), initialText)
	txa.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Height:  style.Some(style.Cells(7)), // 3 lines visible + 4 cells inset (border=1, padding=1)
		Padding: style.Some(style.Edges(1, 1)),
		Border:  style.SingleBorder().Some(),
	})

	root := element.NewBox(eng.Document())
	root.AppendChild(txa)
	eng.Mount(root)

	// Focus the textarea so the engine tracks its cursor.
	eng.FocusManager().Focus(txa, focus.ReasonProgrammatic)
	eng.Frame()

	// Place cursor at "Line 3" (offset 14)
	txa.Buffer().SetOffset(14)
	txa.SyncBuffer()
	eng.Frame()

	// Line 3 is at local Y=2.
	// state.Y = insetTop + localY = 2 + 2 = 4.
	// Initial scroll is (0,0) because Line 3 fits in contentH=3.

	if !b.Cursor.Visible {
		t.Errorf("Initial cursor should be visible")
	}
	if b.Cursor.Y != 4 {
		t.Errorf("Initial hardware cursor Y = %d, want 4 (state.Y=4, scrollY=0)", b.Cursor.Y)
	}

	// Scroll down by 1 line.
	// state.Y = 4. scrollY = 1.
	// b.Cursor.Y = 4 - 1 = 3.
	txa.ScrollTo(0, 1)
	eng.Frame()

	if b.Cursor.Y != 3 {
		t.Errorf("Cursor pos after scroll(0, 1) = %d, want 3 (state.Y=4, scrollY=1)", b.Cursor.Y)
	}

	// Scroll down by 2 lines.
	// state.Y = 4. scrollY = 2.
	// b.Cursor.Y = 4 - 2 = 2.
	txa.ScrollTo(0, 2)
	eng.Frame()

	if b.Cursor.Y != 2 {
		t.Errorf("Cursor pos after scroll(0, 2) = %d, want 2 (state.Y=4, scrollY=2)", b.Cursor.Y)
	}
}
