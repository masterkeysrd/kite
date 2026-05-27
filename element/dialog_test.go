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

	doc := eng.Document()

	// Verify Focus Scope was pushed
	activeScope := doc.ActiveScope()
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

	eng.Document().RemoveChild(dialog)
	eng.Frame()

	// Verify it was removed from overlays
	found = false
	for ovl := range eng.Document().Overlays() {
		if ovl == dialog {
			found = true
			break
		}
	}
	if found {
		t.Error("expected dialog to be removed from document overlays after unmount")
	}

	// Verify Focus Scope was popped
	activeScope = doc.ActiveScope()
	if activeScope == nil {
		t.Fatal("expected active focus scope")
	}
	if activeScope != nil && activeScope.Root == dialog {
		t.Error("dialog focus scope should have been popped")
	}
}
