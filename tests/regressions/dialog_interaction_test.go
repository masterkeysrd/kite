package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

// TestDialogInteraction ensures that the dialog correctly handles focus and
// closing events (Enter/Esc).
func TestDialogInteraction(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	var activeDialog *element.DialogElement

	// Setup a simple app that opens a dialog on 'd'
	env.Engine.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)

		if activeDialog != nil {
			if ke.MatchString("enter") || ke.MatchString("escape") {
				env.Engine.Document().RemoveChild(activeDialog)
				activeDialog = nil
				e.StopPropagation()
				return
			}
		}

		if ke.MatchString("d") && activeDialog == nil {
			activeDialog = element.Dialog(element.Box("Modal Content"), 100)
			env.Engine.Document().AppendChild(activeDialog)
		}
	})

	env.Flush()

	// 1. Initially, no dialog
	if activeDialog != nil {
		t.Fatal("expected no active dialog initially")
	}

	// 2. Press 'd' to open
	env.SendKey(key.Key{Code: 'd'})
	env.Flush()

	if activeDialog == nil {
		t.Fatal("expected dialog to be open after pressing 'd'")
	}

	// 3. Verify dialog is focused (because we added IsFocusable and Autofocus)
	focused := env.Engine.Document().CurrentFocus()
	if focused != activeDialog {
		t.Errorf("expected dialog to be focused, got %T", focused)
	}

	// 4. Press 'Enter' to close
	env.SendKey(key.Key{Code: 13}) // Enter
	env.Flush()

	if activeDialog != nil {
		t.Fatal("expected dialog to be closed after pressing 'Enter'")
	}

	// 5. Open again and press 'Esc' to close
	env.SendKey(key.Key{Code: 'd'})
	env.Flush()
	if activeDialog == nil {
		t.Fatal("expected dialog to be open again")
	}

	env.SendKey(key.Key{Code: 27}) // Escape
	env.Flush()

	if activeDialog != nil {
		t.Fatal("expected dialog to be closed after pressing 'Esc'")
	}
}
