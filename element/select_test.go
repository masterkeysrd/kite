package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/layout"
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
	btn := uaRoot.FirstChild().(*element.ButtonElement)
	
	btn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))

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

func TestSelect_Selection(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusManager(fm)

	opt1 := element.Option("Option 1", "opt1")
	s := element.NewSelect(doc,
		opt1,
		element.Option("Option 2", "opt2"),
	)
	doc.AppendChild(s)

	// Open dropdown
	uaRoot := dom.UARoot(s)
	btn := uaRoot.FirstChild().(*element.ButtonElement)
	btn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))

	// Find the button in overlay that corresponds to opt1
	var overlay element.Element
	for o := range doc.Overlays() {
		overlay = o.(element.Element)
		break
	}

	// The overlay root is a Box, which contains buttons for options.
	// We'll just trigger a click on the first button in the overlay.
	overlayContent := overlay.FirstChild().(element.Element)
	firstOptBtn := overlayContent.FirstChild().(*element.ButtonElement)

	changed := false
	s.OnEvent(event.EventChange, func(e event.Event) {
		changed = true
	})

	firstOptBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))

	if s.Value() != "opt1" {
		t.Errorf("expected value opt1, got %q", s.Value())
	}
	if !changed {
		t.Error("expected EventChange")
	}

	// Verify overlay is hidden
	foundOverlay := false
	for range doc.Overlays() {
		foundOverlay = true
	}
	if foundOverlay {
		t.Error("expected overlay to be hidden")
	}
}
