package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

func TestSelect_OpenDropdown(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusManager(fm)

	s := element.NewSelect(doc,
		element.Option("Option 1", "opt1"),
		element.Option("Option 2", "opt2"),
	)
	doc.AppendChild(s)

	if s.Value() != "" {
		t.Errorf("expected empty value, got %q", s.Value())
	}

	// Trigger open via UA button click
	uaRoot := dom.UARoot(s)
	if uaRoot == nil {
		t.Fatal("expected UA root")
	}
	btn, ok := uaRoot.FirstChild().(*element.ButtonElement)
	if !ok {
		t.Fatalf("expected UA root child to be ButtonElement, got %T", uaRoot.FirstChild())
	}

	btn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))

	// Verify overlay is added to document
	foundOverlay := false
	for range doc.Overlays() {
		foundOverlay = true
		break
	}
	if !foundOverlay {
		t.Error("expected overlay to be shown")
	}

	// Verify focus scope is pushed
	if fm.ActiveScope().Root == nil {
		t.Error("expected active scope root to be set")
	}
}

func TestSelect_KeyboardSelection(t *testing.T) {
	t.Skip("Skping until we can simulate focus changes in tests. See")
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusManager(fm)

	s := element.NewSelect(doc,
		element.Option("Option 1", "opt1"),
		element.Option("Option 2", "opt2"),
	)
	doc.AppendChild(s)

	// Focus select and press down to open
	fm.Focus(s, focus.ReasonKeyboard)
	s.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: key.KeyDown}))

	// NewSelect calls fm.PushScope which should have updated focus to autofocus
	current := fm.Current()
	if current == s {
		// If it's still s, let's manually focus autofocus to simulate what Engine does
		fm.Focus(fm.ActiveScope().Autofocus, focus.ReasonProgrammatic)
		current = fm.Current()
	}

	// Focus should be on the first button in the overlay
	current = fm.Current()
	if current == nil {
		t.Fatal("expected focused element")
	}

	t.Logf("Focused node type: %T", current)
	btn, ok := current.(*element.ButtonElement)
	if !ok {
		t.Fatalf("expected focused ButtonElement, got %T", current)
	}
	t.Logf("Button data: %v", btn.TextContent())

	// Press Enter on the focused button
	click := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	btn.DispatchEvent(click)

	if s.Value() != "opt1" {
		t.Errorf("expected value opt1, got %q", s.Value())
	}
}
