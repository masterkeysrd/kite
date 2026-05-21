package uv

import (
	"log"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

func translateEvent(ev uv.Event) event.RawEvent {
	switch e := ev.(type) {
	case uv.KeyPressEvent:
		return &event.RawKeyEvent{
			Key: key.Key{
				Text:     e.Text,
				Code:     e.Code,
				IsRepeat: e.IsRepeat,
				Mod:      translateModifiers(e.Mod),
			},
			Up: false,
		}
	case uv.KeyReleaseEvent:
		return &event.RawKeyEvent{
			Key: key.Key{
				Text: e.Text,
				Code: e.Code,
				Mod:  translateModifiers(e.Mod),
			},
			Up: true,
		}
	case uv.WindowSizeEvent:
		return &event.RawResizeEvent{
			Width:  e.Width,
			Height: e.Height,
		}
	case uv.MouseClickEvent:
		return &event.RawMouseEvent{
			X:      e.X,
			Y:      e.Y,
			Button: translateMouseButton(e.Button),
			Up:     false,
			Move:   false,
			Mod:    translateModifiers(e.Mod),
		}
	case uv.MouseReleaseEvent:
		return &event.RawMouseEvent{
			X:      e.X,
			Y:      e.Y,
			Button: translateMouseButton(e.Button),
			Up:     true,
			Move:   false,
			Mod:    translateModifiers(e.Mod),
		}
	case uv.MouseMotionEvent:
		return &event.RawMouseEvent{
			X:      e.X,
			Y:      e.Y,
			Button: translateMouseButton(e.Button),
			Up:     false,
			Move:   true,
			Mod:    translateModifiers(e.Mod),
		}
	case uv.MouseWheelEvent:
		m := e.Mouse()
		deltaX, deltaY := 0, 0
		switch m.Button {
		case uv.MouseWheelUp:
			deltaY = -1
		case uv.MouseWheelDown:
			deltaY = 1
		case uv.MouseWheelLeft:
			deltaX = -1
		case uv.MouseWheelRight:
			deltaX = 1
		}
		return &event.RawMouseEvent{
			X:      m.X,
			Y:      m.Y,
			DeltaX: deltaX,
			DeltaY: deltaY,
			Mod:    translateModifiers(m.Mod),
		}
	}

	log.Printf("unhandled event: %#v", ev)
	return &event.RawUnknownEvent{
		Payload: ev,
	}
}

func translateModifiers(mod uv.KeyMod) key.Mod {
	var m key.Mod
	if mod&uv.ModShift != 0 {
		m |= key.ModShift
	}
	if mod&uv.ModCtrl != 0 {
		m |= key.ModCtrl
	}
	if mod&uv.ModAlt != 0 {
		m |= key.ModAlt
	}
	if mod&uv.ModMeta != 0 {
		m |= key.ModMeta
	}
	if mod&uv.ModSuper != 0 {
		m |= key.ModSuper
	}
	if mod&uv.ModHyper != 0 {
		m |= key.ModHyper
	}
	if mod&uv.ModCapsLock != 0 {
		m |= key.ModCapsLock
	}
	if mod&uv.ModNumLock != 0 {
		m |= key.ModNumLock
	}
	if mod&uv.ModScrollLock != 0 {
		m |= key.ModScrollLock
	}
	return m
}

func translateMouseButton(button uv.MouseButton) event.MouseButton {
	switch button {
	case uv.MouseLeft:
		return event.ButtonLeft
	case uv.MouseMiddle:
		return event.ButtonMiddle
	case uv.MouseRight:
		return event.ButtonRight
	default:
		return event.ButtonNone
	}
}
