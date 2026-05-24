package uv

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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
	case uv.PasteEvent:
		return &event.RawBracketedPaste{
			Text: e.Content,
		}
	case *uv.PasteEvent:
		return &event.RawBracketedPaste{
			Text: e.Content,
		}
	case uv.UnknownOscEvent:
		s := string(e)
		return parseOsc(s)
	case *uv.UnknownOscEvent:
		s := string(*e)
		return parseOsc(s)
	case uv.UnknownCsiEvent:
		s := string(e)
		return &event.RawUnknownEvent{Payload: prefixCSI + s}
	case uv.UnknownDcsEvent:
		s := string(e)
		return &event.RawUnknownEvent{Payload: prefixDCS + s}
	case uv.UnknownApcEvent:
		s := string(e)
		return &event.RawUnknownEvent{Payload: prefixAPC + s}
	case uv.UnknownEvent:
		s := string(e)
		return &event.RawUnknownEvent{Payload: prefixUNK + s}
	case uv.BackgroundColorEvent:
		return &event.RawUnknownEvent{Payload: prefixBGC + e.String()}
	case uv.PixelSizeEvent:
		return &event.RawUnknownEvent{Payload: e}
	}

	slog.Info("UV: Unhandled event type", "type", fmt.Sprintf("%T", ev), "val", fmt.Sprintf("%#v", ev))
	return &event.RawUnknownEvent{
		Payload: ev,
	}
}

func parseOsc(s string) event.RawEvent {
	const escBEL = "\x07"

	// Strip ESC ] if present
	s = strings.TrimPrefix(s, escOSC)
	s = strings.TrimSuffix(s, escBEL)
	s = strings.TrimSuffix(s, escST)

	parts := strings.SplitN(s, ";", 2)
	code := 0
	data := ""
	if len(parts) == 2 {
		var err error
		code, err = strconv.Atoi(parts[0])
		if err != nil {
			slog.Warn("UV: Failed to parse OSC code", "raw", parts[0], "error", err)
		}
		data = parts[1]
	} else {
		data = s
	}

	// Handle OSC 52 response (Clipboard read)
	if code == oscClipboard {
		// Data format is "c;<base64>" or "p;<base64>" etc.
		oscParts := strings.SplitN(data, ";", 2)
		if len(oscParts) == 2 {
			b64 := oscParts[1]
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err == nil {
				return &event.RawBracketedPaste{Text: string(decoded)}
			}
		}
	}

	return &event.RawOscEvent{
		Code: code,
		Data: data,
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
