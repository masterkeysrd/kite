package uv

import (
	"fmt"
	kitelog "github.com/masterkeysrd/kite/log"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

func translateEvent(ev uv.Event) (out backend.RawEvent) {
	switch e := ev.(type) {
	case uv.KeyPressEvent:
		return &backend.RawKeyEvent{
			Key: key.Key{
				Text:     e.Text,
				Code:     e.Code,
				IsRepeat: e.IsRepeat,
				Mod:      translateModifiers(e.Mod),
			},
			Up: false,
		}
	case uv.KeyReleaseEvent:
		return &backend.RawKeyEvent{
			Key: key.Key{
				Text: e.Text,
				Code: e.Code,
				Mod:  translateModifiers(e.Mod),
			},
			Up: true,
		}
	case uv.WindowSizeEvent:
		return &backend.RawResizeEvent{
			Width:  e.Width,
			Height: e.Height,
		}
	case uv.MouseClickEvent:
		return &backend.RawMouseEvent{
			X:      e.X,
			Y:      e.Y,
			Button: translateMouseButton(e.Button),
			Up:     false,
			Move:   false,
			Mod:    translateModifiers(e.Mod),
		}
	case uv.MouseReleaseEvent:
		return &backend.RawMouseEvent{
			X:      e.X,
			Y:      e.Y,
			Button: translateMouseButton(e.Button),
			Up:     true,
			Move:   false,
			Mod:    translateModifiers(e.Mod),
		}
	case uv.MouseMotionEvent:
		return &backend.RawMouseEvent{
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
		return &backend.RawMouseEvent{
			X:      m.X,
			Y:      m.Y,
			DeltaX: deltaX,
			DeltaY: deltaY,
			Mod:    translateModifiers(m.Mod),
		}
	case uv.PasteEvent:
		return &backend.RawBracketedPaste{
			Text: e.Content,
		}
	case *uv.PasteEvent:
		return &backend.RawBracketedPaste{
			Text: e.Content,
		}
	case uv.ClipboardEvent:
		if ev, ok := translateClipboardEvent(e); ok {
			return ev
		}
	case *uv.ClipboardEvent:
		if ev, ok := translateClipboardEvent(*e); ok {
			return ev
		}
	case uv.UnknownOscEvent:
		s := string(e)
		kitelog.Info("UV: Received Unknown OSC", "raw", fmt.Sprintf("%q", s))
		return parseOsc(s)
	case *uv.UnknownOscEvent:
		s := string(*e)
		kitelog.Info("UV: Received Unknown OSC (ptr)", "raw", fmt.Sprintf("%q", s))
		return parseOsc(s)
	case uv.UnknownCsiEvent:
		s := string(e)
		kitelog.Info("UV: Received Unknown CSI", "raw", fmt.Sprintf("%q", s))
		return &backend.RawUnknownEvent{Payload: prefixCSI + s}
	case uv.UnknownDcsEvent:
		s := string(e)
		kitelog.Info("UV: Received Unknown DCS", "raw", fmt.Sprintf("%q", s))
		return &backend.RawUnknownEvent{Payload: prefixDCS + s}
	case uv.UnknownApcEvent:
		s := string(e)
		return &backend.RawUnknownEvent{Payload: prefixAPC + s}
	case uv.UnknownEvent:
		s := string(e)
		return &backend.RawUnknownEvent{Payload: prefixUNK + s}
	case uv.BackgroundColorEvent:
		return &backend.RawUnknownEvent{Payload: prefixBGC + e.String()}
	case uv.PixelSizeEvent:
		return &backend.RawUnknownEvent{Payload: e}
	case uv.ModeReportEvent:
		s := fmt.Sprintf("\x1b[?%d;%d$y", e.Mode, e.Value)
		return &backend.RawUnknownEvent{Payload: s}
	case *uv.ModeReportEvent:
		s := fmt.Sprintf("\x1b[?%d;%d$y", e.Mode, e.Value)
		return &backend.RawUnknownEvent{Payload: s}
	}

	kitelog.Info("UV: Unhandled event type", "type", fmt.Sprintf("%T", ev), "val", fmt.Sprintf("%#v", ev))
	return &backend.RawUnknownEvent{
		Payload: ev,
	}
}

func parseOsc(s string) backend.RawEvent {
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
			kitelog.Warn("UV: Failed to parse OSC code", "raw", parts[0], "error", err)
		}
		data = parts[1]
	} else {
		data = s
	}

	return &backend.RawOscEvent{
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

func translateClipboardEvent(e uv.ClipboardEvent) (backend.RawEvent, bool) {
	var selection rune = 0
	switch e.Selection {
	case uv.PrimaryClipboard:
		selection = event.PrimaryClipboard
	case uv.SystemClipboard:
		selection = event.SystemClipboard
	case 0:
		selection = event.UnknownClipboard
	default:
		kitelog.Info("UV: Unknown clipboard selection", "selection", e.Selection)
		return nil, false
	}
	return &backend.RawClipboardEvent{
		Selection: selection,
		Content:   e.Content,
	}, true
}
