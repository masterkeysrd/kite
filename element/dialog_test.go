package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
)

func TestDialog_Lifecycle(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	dialog := element.Dialog(element.Box(), 100)

	// Mount triggers OnConnected
	eng.Document().AppendChild(dialog)

	fm := eng.FocusManager()

	// Verify Focus Scope was pushed
	activeScope := fm.ActiveScope()
	if activeScope == nil {
		t.Fatal("expected active focus scope")
	}
	if activeScope.Root != dialog {
		t.Errorf("expected dialog to be focus scope root")
	}

	// Verify it was added to overlays
	found := false
	for ovl := range eng.Document().Overlays() {
		if ovl == dialog {
			found = true
			break
		}
	}
	if !found {
		t.Error("dialog should be in document overlays")
	}

	// Unmount triggers OnDisconnected
	eng.Document().RemoveChild(dialog)

	// Verify Focus Scope was popped
	activeScope = fm.ActiveScope()
	if activeScope != nil && activeScope.Root == dialog {
		t.Error("dialog focus scope should have been popped")
	}
}
