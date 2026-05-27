package backend

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

// RawBracketedPaste is the backend representation of a bracketed-paste
// sequence (ESC[200~ … ESC[201~).
type RawBracketedPaste struct {
	Text string
}

func (RawBracketedPaste) isRawEvent() {}

// RawOscEvent is a raw OSC sequence.
type RawOscEvent struct {
	Code int
	Data string
}

func (RawOscEvent) isRawEvent() {}

// RawUnknownEvent is a catch-all for backend event that the engine does not
// recognize or handle explicitly.
type RawUnknownEvent struct {
	Payload any
}

func (RawUnknownEvent) isRawEvent() {}

type RawClipboardEvent struct {
	Selection rune // e.g. 0 = primary, 1 = secondary, 2 = clipboard
	Content   string
}

func (RawClipboardEvent) isRawEvent() {}

// --- Raw backend event ------------------------------------------------------

// RawEvent is the interface implemented by all raw backend-level input event.
// It is processed by the Synthesizer to produce high-level structured event.
type RawEvent interface {
	isRawEvent()
}

// RawMouseEvent is the backend representation of a mouse action.
type RawMouseEvent struct {
	X, Y   int
	Button event.MouseButton
	Up     bool // true for button-release
	Move   bool // true when no button change (motion)
	DeltaX int  // wheel
	DeltaY int  // wheel
	Mod    event.Modifiers
}

func (RawMouseEvent) isRawEvent() {}

// RawKeyEvent is the backend representation of a key press or release.
type RawKeyEvent struct {
	key.Key
	Up bool // true for key-release
}

func (RawKeyEvent) isRawEvent() {}

// RawResizeEvent is the backend representation of a terminal resize.
type RawResizeEvent struct {
	Width, Height int
}

func (RawResizeEvent) isRawEvent() {}
