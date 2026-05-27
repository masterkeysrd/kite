package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

func TestCheckbox_Toggle(t *testing.T) {
	cb := element.Checkbox(false)
	if cb.Checked() {
		t.Error("expected unchecked by default")
	}

	root := dom.UARoot(cb)
	if root == nil {
		t.Fatal("expected UA root")
	}
	if root.TextContent() != "[ ]" {
		t.Errorf("expected [ ], got %q", root.TextContent())
	}

	changed := false
	cb.OnEvent(event.EventChange, func(e event.Event) {
		changed = true
	})

	d := event.NewDispatcher()
	path := []event.EventTarget{cb}

	// Click
	click := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(click, path)

	if !cb.Checked() {
		t.Error("expected checked after click")
	}
	if root.TextContent() != "[X]" {
		t.Errorf("expected [X], got %q", root.TextContent())
	}
	if !changed {
		t.Error("expected EventChange after click")
	}

	changed = false
	// Space key
	kd := event.NewKeyEvent(event.EventKeyDown, key.Key{Code: ' ', Text: " "})
	d.Dispatch(kd, path)

	if cb.Checked() {
		t.Error("expected unchecked after space")
	}
	if root.TextContent() != "[ ]" {
		t.Errorf("expected [ ], got %q", root.TextContent())
	}
}

func TestCheckbox_SetChecked(t *testing.T) {
	cb := element.Checkbox(false)
	cb.SetChecked(true)
	if !cb.Checked() {
		t.Error("expected checked")
	}
	root := dom.UARoot(cb)
	if root.TextContent() != "[X]" {
		t.Errorf("expected [X], got %q", root.TextContent())
	}
}

func TestCheckbox_SetGlyphs(t *testing.T) {
	cb := element.Checkbox(false)
	cb.SetGlyphs("N", "Y")
	root := dom.UARoot(cb)
	if root.TextContent() != "N" {
		t.Errorf("expected N, got %q", root.TextContent())
	}
	cb.SetChecked(true)
	if root.TextContent() != "Y" {
		t.Errorf("expected Y, got %q", root.TextContent())
	}
}
