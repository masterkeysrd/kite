// Package event defines event types, the Dispatcher, the Synthesizer, and
// KeyStroke helpers for kite/x (kite v2).
package event

import (
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
)

// EventType is a typed string identifier for event.
type EventType string

// Standard event type constants.
const (
	EventKeyDown   EventType = "keydown"
	EventKeyUp     EventType = "keyup"
	EventKeyPress  EventType = "keypress"
	EventMouseDown EventType = "mousedown"
	EventMouseUp   EventType = "mouseup"
	EventMouseMove EventType = "mousemove"
	EventClick     EventType = "click"
	EventDrag      EventType = "drag"
	EventWheel     EventType = "wheel"
	EventFocus     EventType = "focus"
	EventBlur      EventType = "blur"
	EventFocusIn   EventType = "focusin"
	EventFocusOut  EventType = "focusout"
	EventResize    EventType = "resize"
	EventPaste     EventType = "paste"
	EventCopy      EventType = "copy"
	EventCut       EventType = "cut"
)

// EventPhase represents the current dispatch phase.
type EventPhase uint8

const (
	// PhaseNone is the default phase (not dispatching).
	PhaseNone EventPhase = iota
	// PhaseCapture is the top-down capture phase.
	PhaseCapture
	// PhaseTarget is the target phase.
	PhaseTarget
	// PhaseBubble is the bottom-up bubble phase.
	PhaseBubble
)

// Event is the interface implemented by all event types. Concrete event
// embed BaseEvent which provides default implementations.
type Event interface {
	// Type returns the event type identifier.
	Type() EventType

	// Target returns the element that originally received the event.
	Target() render.Object

	// CurrentTarget returns the element whose listener is currently
	// being invoked.
	CurrentTarget() render.Object

	// Phase returns the current dispatch phase.
	Phase() EventPhase

	// Bubbles reports whether the event bubbles through the ancestor chain.
	Bubbles() bool

	// StopPropagation prevent the event from reaching further listeners
	// in subsequent phases.
	StopPropagation()

	// PropagationStopped reports whether StopPropagation was called.
	PropagationStopped() bool

	// PreventDefault signals that the default action should be suppressed.
	PreventDefault()

	// DefaultPrevented reports whether PreventDefault was called.
	DefaultPrevented() bool

	// setTarget sets the event target. Used internally by Dispatcher.
	setTarget(render.Object)

	// setCurrentTarget sets the current target. Used internally by Dispatcher.
	setCurrentTarget(render.Object)

	// setPhase sets the dispatch phase. Used internally by Dispatcher.
	setPhase(EventPhase)
}

// BaseEvent holds common dispatch state shared by all event types.
// Concrete event types embed BaseEvent.
type BaseEvent struct {
	typ              EventType
	target           render.Object
	currentTarget    render.Object
	phase            EventPhase
	propagationStop  bool
	defaultPrevented bool
	bubbles          bool
}

// Type returns the event type identifier.
func (b *BaseEvent) Type() EventType { return b.typ }

// Target returns the element that originally received the event.
func (b *BaseEvent) Target() render.Object { return b.target }

// CurrentTarget returns the element whose listener is currently being invoked.
func (b *BaseEvent) CurrentTarget() render.Object { return b.currentTarget }

// Phase returns the current dispatch phase.
func (b *BaseEvent) Phase() EventPhase { return b.phase }

// Bubbles reports whether the event bubbles.
func (b *BaseEvent) Bubbles() bool { return b.bubbles }

// StopPropagation prevent the event from reaching further listeners.
func (b *BaseEvent) StopPropagation() { b.propagationStop = true }

// PropagationStopped reports whether StopPropagation was called.
func (b *BaseEvent) PropagationStopped() bool { return b.propagationStop }

// PreventDefault signals that the default action should be suppressed.
func (b *BaseEvent) PreventDefault() { b.defaultPrevented = true }

// DefaultPrevented reports whether PreventDefault was called.
func (b *BaseEvent) DefaultPrevented() bool { return b.defaultPrevented }

func (b *BaseEvent) setTarget(o render.Object)        { b.target = o }
func (b *BaseEvent) setCurrentTarget(o render.Object) { b.currentTarget = o }
func (b *BaseEvent) setPhase(p EventPhase)            { b.phase = p }

// --- Key types ---------------------------------------------------------------

// Modifiers is a bitmask of keyboard modifier keys.
type Modifiers = key.Mod

const (
	// ModCtrl is the Ctrl modifier.
	ModCtrl = key.ModCtrl
	// ModAlt is the Alt modifier.
	ModAlt = key.ModAlt
	// ModShift is the Shift modifier.
	ModShift = key.ModShift
	// ModMeta is the Meta (Cmd/Win) modifier.
	ModMeta = key.ModMeta
)

// MouseButton identifies which mouse button was involved in the event.
type MouseButton uint8

const (
	// ButtonNone indicates no button (e.g. mouse move).
	ButtonNone MouseButton = iota
	// ButtonLeft is the primary mouse button.
	ButtonLeft
	// ButtonMiddle is the middle mouse button.
	ButtonMiddle
	// ButtonRight is the right mouse button.
	ButtonRight
)

// --- Concrete event types ----------------------------------------------------

// KeyEvent is dispatched on key-press, key-up, and key-down.
type KeyEvent struct {
	BaseEvent
	key.Key
}

// NewKeyEvent creates a KeyEvent of the given type.
func NewKeyEvent(typ EventType, k key.Key) *KeyEvent {
	return &KeyEvent{
		BaseEvent: BaseEvent{typ: typ, bubbles: true},
		Key:       k,
	}
}

// HitResult caches the hit-test result computed when a MouseEvent is
// synthesized. It is valid only for the lifetime of the event.
type HitResult struct {
	// Object is the deepest render object at the hit point.
	Object render.Object
}

// MouseEvent is dispatched for mouse-button, move, and click event.
// MouseEvent bubbles.
type MouseEvent struct {
	BaseEvent

	// Screen holds the absolute screen-space coordinates.
	Screen layout.Point
	// Local holds the coordinates local to the Target's render bounds.
	Local layout.Point
	// Hit is the cached hit-test result from when the event was synthesized.
	Hit HitResult
	// Button is the mouse button involved.
	Button MouseButton
	// Mods holds the active modifier keys.
	Mods Modifiers
}

// NewMouseEvent creates a MouseEvent of the given type.
func NewMouseEvent(typ EventType, screen layout.Point, button MouseButton, mods Modifiers) *MouseEvent {
	return &MouseEvent{
		BaseEvent: BaseEvent{typ: typ, bubbles: true},
		Screen:    screen,
		Button:    button,
		Mods:      mods,
	}
}

// WheelEvent is dispatched when the scroll wheel is turned. WheelEvent bubbles.
type WheelEvent struct {
	BaseEvent

	// Screen holds the absolute screen-space coordinates.
	Screen layout.Point
	// Local holds the coordinates local to the Target's render bounds.
	Local layout.Point
	// DeltaX is the horizontal scroll delta (positive = right).
	DeltaX int
	// DeltaY is the vertical scroll delta (positive = down).
	DeltaY int
	// Mods holds the active modifier keys.
	Mods Modifiers
}

// NewWheelEvent creates a WheelEvent.
func NewWheelEvent(screen layout.Point, dx, dy int, mods Modifiers) *WheelEvent {
	return &WheelEvent{
		BaseEvent: BaseEvent{typ: EventWheel, bubbles: true},
		Screen:    screen,
		DeltaX:    dx,
		DeltaY:    dy,
		Mods:      mods,
	}
}

// FocusEvent is dispatched when an element gains or loses keyboard focus.
// "focus" and "blur" do not bubble; "focusin" and "focusout" do.
type FocusEvent struct {
	BaseEvent

	// RelatedTarget is the element losing focus (for focus event) or
	// gaining focus (for blur event), or nil if none.
	RelatedTarget render.Object
}

// NewFocusEvent creates a FocusEvent of the given type.
func NewFocusEvent(typ EventType, related render.Object) *FocusEvent {
	bubbles := typ == EventFocusIn || typ == EventFocusOut
	return &FocusEvent{
		BaseEvent:     BaseEvent{typ: typ, bubbles: bubbles},
		RelatedTarget: related,
	}
}

// ResizeEvent is dispatched when the terminal viewport dimensions change.
// ResizeEvent does not bubble.
type ResizeEvent struct {
	BaseEvent

	// Width is the new viewport width in columns.
	Width int
	// Height is the new viewport height in rows.
	Height int
}

// NewResizeEvent creates a ResizeEvent.
func NewResizeEvent(width, height int) *ResizeEvent {
	return &ResizeEvent{
		BaseEvent: BaseEvent{typ: EventResize, bubbles: false},
		Width:     width,
		Height:    height,
	}
}

// PasteEvent is dispatched when the user pastes text (bracketed paste or
// Ctrl+V). PasteEvent does not bubble past the focused element.
type PasteEvent struct {
	BaseEvent

	// Text is the pasted text content.
	Text string
}

// NewPasteEvent creates a PasteEvent.
func NewPasteEvent(text string) *PasteEvent {
	return &PasteEvent{
		BaseEvent: BaseEvent{typ: EventPaste, bubbles: false},
		Text:      text,
	}
}

// ClipboardType identifies the clipboard operation.
type ClipboardType uint8

const (
	// ClipboardCopy is a copy operation (Ctrl+C with active selection).
	ClipboardCopy ClipboardType = iota
	// ClipboardCut is a cut operation (Ctrl+X with active selection).
	ClipboardCut
	// ClipboardPaste is a paste operation (Ctrl+V or bracketed paste).
	ClipboardPaste
)

// ClipboardEvent is dispatched for copy, cut, and paste clipboard operations.
// ClipboardEvent does not bubble.
type ClipboardEvent struct {
	BaseEvent

	// ClipType identifies whether this is a copy, cut, or paste.
	ClipType ClipboardType
	// Data is the clipboard text data involved.
	Data string
}

// NewClipboardEvent creates a ClipboardEvent.
func NewClipboardEvent(typ EventType, ct ClipboardType, data string) *ClipboardEvent {
	return &ClipboardEvent{
		BaseEvent: BaseEvent{typ: typ, bubbles: false},
		ClipType:  ct,
		Data:      data,
	}
}

// Scrollable is implemented by render objects that handle wheel event.
// The Dispatcher stops wheel-event bubbling at the first ancestor that
// implements Scrollable and calls OnWheel.
type Scrollable interface {
	OnWheel(e *WheelEvent)
}

// SelectionProvider is implemented by elements that can report an active text
// selection. The Synthesizer uses it to decide whether to emit Copy/Cut
// Clipboardevent.
type SelectionProvider interface {
	// SelectedText returns the currently selected text, or "" if nothing is
	// selected.
	SelectedText() string
}

// --- Raw backend event ------------------------------------------------------

// RawEvent is the interface implemented by all raw backend-level input event.
// It is processed by the Synthesizer to produce high-level structured event.
type RawEvent interface {
	isRawEvent()
}

// RawMouseEvent is the backend representation of a mouse action.
type RawMouseEvent struct {
	X, Y   int
	Button MouseButton
	Up     bool // true for button-release
	Move   bool // true when no button change (motion)
	DeltaX int  // wheel
	DeltaY int  // wheel
	Mod    Modifiers
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

// RawBracketedPaste is the backend representation of a bracketed-paste
// sequence (ESC[200~ … ESC[201~).
type RawBracketedPaste struct {
	Text string
}

func (RawBracketedPaste) isRawEvent() {}

// RawUnknownEvent is a catch-all for backend event that the engine does not
// recognize or handle explicitly.
type RawUnknownEvent struct {
	Payload any
}

func (RawUnknownEvent) isRawEvent() {}
