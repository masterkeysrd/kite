package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
)

// TestRegression_StartupAutofocus verifies that the first focusable element
// is automatically focused on the first frame, and its cursor is shown.
func TestRegression_StartupAutofocus(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "initial")
	eng.Mount(inp)

	// Before any frame, focus is nil.
	if eng.FocusManager().Current() != nil {
		t.Fatal("pre-frame focus should be nil")
	}

	// Trigger the first frame.
	eng.Frame()

	// 1. Focus should be on the input.
	if eng.FocusManager().Current() != inp {
		t.Errorf("after first frame: focused element = %v, want the input", eng.FocusManager().Current())
	}

	// 2. The hardware cursor should be visible in the backend.
	if !b.Cursor.Visible {
		t.Error("after first frame: hardware cursor is hidden, want visible")
	}
}
