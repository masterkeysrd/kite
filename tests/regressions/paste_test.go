// Regression tests for Clipboard — covers reports of Ctrl+V / Alt+V async paste failure.
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/key"
)

func TestPaste_CtrlV(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Set initial clipboard content in backend.
	// env.Backend.SetClipboard("pasted text") // REPLACED: backend no longer has direct clipboard methods.

	input := element.Input("").WithID("input")
	env.Mount(input)
	env.Flush()

	// Focus the input.
	env.Engine.FocusManager().Focus(input, focus.ReasonProgrammatic)
	env.Flush()

	// Simulate Ctrl+V (using 0x16 as code).
	// New behavior: key press alone does NOT paste immediately, even if cached.
	env.SendKey(key.Key{Code: 0x16, Mod: 0})
	env.Flush()

	if got := input.Value(); got != "" {
		t.Errorf("Value() after Ctrl+V key should still be empty (async flow), got %q", got)
	}

	// Simulate the terminal's OSC 52 response arriving as RawBracketedPaste.
	env.Engine.ProcessRawEvent(&backend.RawBracketedPaste{Text: "pasted text"})
	env.Flush()

	if got := input.Value(); got != "pasted text" {
		t.Errorf("Value() after async response = %q, want %q", got, "pasted text")
	}

	// Reset input.
	input.SetValue("")
	env.Flush()

	// Simulate Alt+V (key code 'v' + ModAlt).
	env.SendKey(key.Key{Code: 'v', Mod: key.ModAlt})
	env.Flush()

	if input.Value() != "" {
		t.Errorf("Value() should be empty before async response (Alt+V), got %q", input.Value())
	}

	// Simulate the terminal's OSC 52 response.
	env.Engine.ProcessRawEvent(&backend.RawBracketedPaste{Text: "alt+v text"})
	env.Flush()

	if got := input.Value(); got != "alt+v text" {
		t.Errorf("Value() after async Alt+V response = %q, want %q", got, "alt+v text")
	}

	// Test 3: uv.ClipboardEvent (mapped via RawBracketedPaste)
	input.SetValue("")
	env.Flush()
	// This simulates what uv.Backend sends when it receives a uv.ClipboardEvent from the terminal.
	env.Engine.ProcessRawEvent(&backend.RawBracketedPaste{Text: "direct clipboard event"})
	env.Flush()

	if got := input.Value(); got != "direct clipboard event" {
		t.Errorf("Value() after uv.ClipboardEvent (RawBracketedPaste) = %q, want %q", got, "direct clipboard event")
	}
}
