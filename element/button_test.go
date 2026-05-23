package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
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
	down := event.NewMouseEvent(event.EventMouseDown, layout.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(down, path)
	if clicked {
		t.Error("clicked fired prematurely on MouseDown")
	}

	// MouseUp
	up := event.NewMouseEvent(event.EventMouseUp, layout.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(up, path)
	if !clicked {
		t.Error("clicked did not fire on MouseUp")
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
	is := btn.IntrinsicStyle()
	if is.Reverse.UnwrapOr(false) {
		t.Error("button should not be reversed by default")
	}

	d := event.NewDispatcher()
	path := []event.EventTarget{btn}

	// MouseDown
	down := event.NewMouseEvent(event.EventMouseDown, layout.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(down, path)

	is = btn.IntrinsicStyle()
	if !is.Reverse.UnwrapOr(false) {
		t.Error("button should be reversed when active")
	}

	// MouseUp
	up := event.NewMouseEvent(event.EventMouseUp, layout.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	d.Dispatch(up, path)

	is = btn.IntrinsicStyle()
	if is.Reverse.UnwrapOr(false) {
		t.Error("button should not be reversed after mouse up")
	}
}
