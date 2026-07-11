package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/key"
)

func TestSelect_OpenDropdown(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

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
	if scope := doc.ActiveScope(); scope == nil || scope.Root == nil {
		t.Error("expected active scope root to be set")
	}
}

func TestSelect_KeyboardSelection(t *testing.T) {
	t.Skip("Skping until we can simulate focus changes in tests. See")
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

	s := element.NewSelect(doc,
		element.Option("Option 1", "opt1"),
		element.Option("Option 2", "opt2"),
	)
	doc.AppendChild(s)

	// Focus select and press down to open
	fm.SetFocus(s, focus.ReasonKeyboard)
	s.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: key.KeyDown}))

	// NewSelect calls fm.PushScope which should have updated focus to autofocus
	current := fm.Current()
	if current == s {
		scope := doc.ActiveScope()
		if scope == nil || scope.Autofocus == nil {
			t.Fatal("expected active scope with autofocus")
		}
		fm.SetFocus(scope.Autofocus, focus.ReasonProgrammatic)
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

func TestSelect_PlaceholderOption(t *testing.T) {
	doc := dom.NewDocument()
	s := element.NewSelect(doc,
		element.Option("Choose a language...", ""),
		element.Option("Go", "go"),
	)
	doc.AppendChild(s)

	uaRoot := dom.UARoot(s)
	if uaRoot == nil {
		t.Fatal("expected UA root")
	}
	btn, ok := uaRoot.FirstChild().(*element.ButtonElement)
	if !ok {
		t.Fatalf("expected UA root child to be ButtonElement, got %T", uaRoot.FirstChild())
	}

	// Because value is "", and we have an option with value "", the button text should match the option's text:
	if txt := btn.TextContent(); txt != "Choose a language...▼" {
		t.Errorf("expected button text to be 'Choose a language...▼', got %q", txt)
	}

	// If we set the value to "go", it should update to "Go▼"
	s.SetValue("go")
	if txt := btn.TextContent(); txt != "Go▼" {
		t.Errorf("expected button text to be 'Go▼', got %q", txt)
	}
}
