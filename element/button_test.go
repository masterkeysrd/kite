package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

func TestButton_Click_Mouse(t *testing.T) {
	btn := element.Button("Click Me")
	clicked := false
	btn.OnEvent(event.EventClick, func(e event.Event) {
		clicked = true
	})

	d := event.NewDispatcher()
	path := []event.EventTarget{btn}

	// MouseDown
	down := event.NewMouseEvent(event.EventMouseDown, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(down, path)
	if clicked {
		t.Error("clicked fired prematurely on MouseDown")
	}

	// MouseUp
	up := event.NewMouseEvent(event.EventMouseUp, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(up, path)

	// In Kite, EventClick is synthesized by the engine's Synthesizer.
	// Since we are using a raw Dispatcher here, we must dispatch it manually.
	click := event.NewMouseEvent(event.EventClick, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(click, path)

	if !clicked {
		t.Error("clicked did not fire")
	}
}

func TestButton_Click_Keyboard(t *testing.T) {
	tests := []struct {
		name string
		key  key.Key
	}{
		{"Space", key.Key{Code: ' ', Text: " "}},
		{"Enter", key.Key{Code: key.KeyEnter}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			btn := element.Button("Click Me")
			clicked := false
			btn.OnEvent(event.EventClick, func(e event.Event) {
				clicked = true
			})

			d := event.NewDispatcher()
			path := []event.EventTarget{btn}

			// KeyDown
			kd := event.NewKeyEvent(event.EventKeyDown, tt.key)
			d.Dispatch(kd, path)
			if !clicked {
				t.Errorf("clicked did not fire on KeyDown(%s)", tt.name)
			}
		})
	}
}

func TestButton_IsFocusable(t *testing.T) {
	btn := element.Button("Focus Me")
	if !btn.IsFocusable() {
		t.Error("Button should be focusable")
	}
}

func TestButton_ActiveStyle(t *testing.T) {
	btn := element.Button("Active")
	ds := btn.DefaultStyle()
	if ds.Reverse.UnwrapOr(false) {
		t.Error("button should not be reversed by default")
	}

	d := event.NewDispatcher()
	path := []event.EventTarget{btn}

	// MouseDown
	down := event.NewMouseEvent(event.EventMouseDown, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(down, path)

	ds = btn.DefaultStyle()
	if !ds.Reverse.UnwrapOr(false) {
		t.Error("button should be reversed when active")
	}

	// MouseUp
	up := event.NewMouseEvent(event.EventMouseUp, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(up, path)

	ds = btn.DefaultStyle()
	if ds.Reverse.UnwrapOr(false) {
		t.Error("button should not be reversed after mouse up")
	}
}

func TestButton_Disabled_Click(t *testing.T) {
	btn := element.Button("Click Me").Disabled(true)
	clicked := false
	btn.OnEvent(event.EventClick, func(e event.Event) {
		clicked = true
	})

	d := event.NewDispatcher()
	path := []event.EventTarget{btn}

	// 1. Mouse Click attempt
	click := event.NewMouseEvent(event.EventClick, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(click, path)
	if clicked {
		t.Error("clicked fired on disabled button for Mouse EventClick")
	}

	// 2. Keyboard Space attempt
	clicked = false
	kd := event.NewKeyEvent(event.EventKeyDown, key.Key{Code: ' ', Text: " "})
	d.Dispatch(kd, path)
	if clicked {
		t.Error("clicked fired on disabled button for KeyDown Space")
	}
}

func TestButton_Disabled_Click_TestEnv(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	clicked := false
	btn := element.Button("Click Me").Disabled(true).OnEvent(event.EventClick, func(e event.Event) {
		clicked = true
	})

	env.Mount(btn)
	env.Flush()

	// Click at the button's position.
	env.Click(0, 0)
	env.Flush()

	if clicked {
		t.Error("clicked fired on disabled button inside testenv simulation")
	}
}
