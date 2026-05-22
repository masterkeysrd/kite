package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
)

// TestRegression_StartupAutofocus verifies that the first focusable element
// is automatically focused on the first frame, and its cursor is shown.
func TestRegression_StartupAutofocus(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	inp := element.NewInput(env.Document(), "initial")
	env.Mount(inp)

	// Before any frame, focus should be nil (no focused node).
	if env.CurrentFocus() != nil {
		t.Fatal("pre-frame focus should be nil")
	}

	// Trigger the first frame.
	env.RenderFrame()

	// 1. Focus should be on the input.
	testenv.Expect(t, inp).ToHaveFocus(env)

	// 2. The hardware cursor should be visible in the backend.
	testenv.Expect(t, inp).ExpectHardwareCursorVisible(env)
}
