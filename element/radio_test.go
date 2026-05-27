package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
)

func TestRadioGroup(t *testing.T) {
	r1 := element.Radio("1")
	r2 := element.Radio("2")
	rg := element.RadioGroup(r1, r2)

	if rg.Value() != "" {
		t.Errorf("expected empty value, got %q", rg.Value())
	}

	root1 := dom.UARoot(r1)
	root2 := dom.UARoot(r2)

	if root1.TextContent() != "( )" || root2.TextContent() != "( )" {
		t.Errorf("expected unchecked glyphs, got %q and %q", root1.TextContent(), root2.TextContent())
	}

	changed := false
	rg.OnEvent(event.EventChange, func(e event.Event) {
		changed = true
	})

	d := event.NewDispatcher()

	// Click r2
	click := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(click, []event.EventTarget{rg, r2})

	if rg.Value() != "2" {
		t.Errorf("expected value 2, got %q", rg.Value())
	}
	if root1.TextContent() != "( )" || root2.TextContent() != "(•)" {
		t.Errorf("expected r2 checked, got %q and %q", root1.TextContent(), root2.TextContent())
	}
	if !changed {
		t.Error("expected EventChange on group")
	}

	// Click r1
	d.Dispatch(click, []event.EventTarget{rg, r1})
	if rg.Value() != "1" {
		t.Errorf("expected value 1, got %q", rg.Value())
	}
	if root1.TextContent() != "(•)" || root2.TextContent() != "( )" {
		t.Errorf("expected r1 checked, got %q and %q", root1.TextContent(), root2.TextContent())
	}
}

func TestRadioGroup_InitialValue(t *testing.T) {
	r1 := element.Radio("1")
	_ = element.RadioGroup(r1).SetValue("1")

	root1 := dom.UARoot(r1)
	if root1.TextContent() != "(•)" {
		t.Errorf("expected checked glyph, got %q", root1.TextContent())
	}
}

func TestRadioGroup_DynamicAddition(t *testing.T) {
	rg := element.RadioGroup().SetValue("1")
	r1 := element.Radio("1")
	rg.AddChild(r1)

	root1 := dom.UARoot(r1)
	if root1.TextContent() != "(•)" {
		t.Errorf("expected checked glyph, got %q", root1.TextContent())
	}
}

func TestRadioGroup_Nested(t *testing.T) {
	r1 := element.Radio("1")
	box := element.Box(r1)
	rg := element.RadioGroup(box)

	rg.SetValue("1")
	root1 := dom.UARoot(r1)
	if root1.TextContent() != "(•)" {
		t.Errorf("expected checked glyph, got %q", root1.TextContent())
	}

	r1.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if rg.Value() != "1" {
		t.Errorf("expected value 1, got %q", rg.Value())
	}
}
