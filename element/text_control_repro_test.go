package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/key"
)

func TestTextArea_Repro_StepByStep(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	content := "ABC\nDEF\nGHI"
	txa := element.TextArea(content)
	env.Mount(txa)
	env.Flush()

	// 1. Move to start of Line 2 ('D', offset 4).
	txa.SetSelectionRange(4, 4)
	env.Flush()

	sel := env.Document().Selection()

	// 2. Shift + Right. Should be "D".
	env.KeyPress("right", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "D" {
		t.Errorf("Step 1 (Shift+Right): expected 'D', got %q", got)
	}

	// 3. Shift + Right again. Should be "DE".
	env.KeyPress("right", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "DE" {
		t.Errorf("Step 2 (Shift+Right): expected 'DE', got %q", got)
	}

	// 4. Shift + Left. Should be back to "D".
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "D" {
		t.Errorf("Step 3 (Shift+Left): expected 'D', got %q", got)
	}

	// 5. Shift + Left. Should be empty (collapsed at 4).
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "" {
		t.Errorf("Step 4 (Shift+Left): expected '', got %q", got)
	}

	// 6. Shift + Left. Should be "\n" (backward selection from 4 to 3).
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "\n" {
		t.Errorf("Step 5 (Shift+Left): expected '\\n', got %q", got)
	}
}
