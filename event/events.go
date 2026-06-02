// Package event defines event types, the Dispatcher, the Synthesizer, and
// KeyStroke helpers for kite/x (kite v2).
package event

import (
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

// EventType is a typed string identifier for event.
type EventType string

// Standard event type constants.
const (
	EventKeyDown         EventType = "keydown"
	EventKeyUp           EventType = "keyup"
	EventKeyPress        EventType = "keypress"
	EventMouseDown       EventType = "mousedown"
	EventMouseUp         EventType = "mouseup"
	EventMouseMove       EventType = "mousemove"
	EventMouseEnter      EventType = "mouseenter"
	EventMouseLeave      EventType = "mouseleave"
	EventMouseOver       EventType = "mouseover"
	EventMouseOut        EventType = "mouseout"
	EventClick           EventType = "click"
	EventDrag            EventType = "drag"
	EventWheel           EventType = "wheel"
	EventFocus           EventType = "focus"
	EventBlur            EventType = "blur"
	EventFocusIn         EventType = "focusin"
	EventFocusOut        EventType = "focusout"
	EventResize          EventType = "resize"
	EventChange          EventType = "change"
	EventInput           EventType = "input"
	EventPaste           EventType = "paste"
	EventCopy            EventType = "copy"
	EventCut             EventType = "cut"
	EventClipboard       EventType = "clipboard"
	EventScroll          EventType = "scroll"
	EventSelectionChange EventType = "selectionchange"
	EventSubmit          EventType = "submit"
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
	// During dispatch, this may be retargeted as the event crosses
	// shadow boundaries.
	Target() EventTarget

	// OriginalTarget returns the raw target that originally received the
	// event, without retargeting. This is an engine-internal accessor used to
	// build the dispatch path.
	OriginalTarget() EventTarget

	// CurrentTarget returns the element whose listener is currently
	// being invoked.
	CurrentTarget() EventTarget

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
}

// InternalEvent is the interface implemented by event types that can be
// updated during the dispatch process. It is used by the Dispatcher and
// Synthesizer in internal/event.
type InternalEvent interface {
	Event
	// SetTarget sets the event target. Used internally by Dispatcher.
	SetTarget(EventTarget)

	// SetCurrentTarget sets the current target. Used internally by Dispatcher.
	SetCurrentTarget(EventTarget)

	// SetPhase sets the dispatch phase. Used internally by Dispatcher.
	SetPhase(EventPhase)
}

// BaseEvent holds common dispatch state shared by all event types.
// Concrete event types embed BaseEvent.
type BaseEvent struct {
	typ              EventType
	target           EventTarget
	currentTarget    EventTarget
	phase            EventPhase
	propagationStop  bool
	defaultPrevented bool
	bubbles          bool
}

// Type returns the event type identifier.
func (b *BaseEvent) Type() EventType { return b.typ }

// Target returns the element that originally received the event.
func (b *BaseEvent) Target() EventTarget { return b.target }

// OriginalTarget returns the raw target that originally received the event.
func (b *BaseEvent) OriginalTarget() EventTarget { return b.target }

// CurrentTarget returns the element whose listener is currently being invoked.
func (b *BaseEvent) CurrentTarget() EventTarget { return b.currentTarget }

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

func (b *BaseEvent) SetTarget(o EventTarget)        { b.target = o }
func (b *BaseEvent) SetCurrentTarget(o EventTarget) { b.currentTarget = o }
func (b *BaseEvent) SetPhase(p EventPhase)          { b.phase = p }

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
	// Target is the deepest event target at the hit point.
	Target EventTarget
}

// MouseEvent is dispatched for mouse-button, move, and click event.
// MouseEvent bubbles.
type MouseEvent struct {
	BaseEvent

	// Screen holds the absolute screen-space coordinates.
	Screen geom.Point
	// Local holds the coordinates local to the Target's render bounds.
	Local geom.Point
	// Hit is the cached hit-test result from when the event was synthesized.
	Hit HitResult
	// Button is the mouse button involved.
	Button MouseButton
	// Mods holds the active modifier keys.
	Mods Modifiers
}

// NewMouseEvent creates a MouseEvent of the given type.
func NewMouseEvent(typ EventType, screen geom.Point, button MouseButton, mods Modifiers) *MouseEvent {
	bubbles := typ != EventMouseEnter && typ != EventMouseLeave
	return &MouseEvent{
		BaseEvent: BaseEvent{typ: typ, bubbles: bubbles},
		Screen:    screen,
		Button:    button,
		Mods:      mods,
	}
}

// WheelEvent is dispatched when the scroll wheel is turned. WheelEvent bubbles.
type WheelEvent struct {
	BaseEvent

	// Screen holds the absolute screen-space coordinates.
	Screen geom.Point
	// Local holds the coordinates local to the Target's render bounds.
	Local geom.Point
	// DeltaX is the horizontal scroll delta (positive = right).
	DeltaX int
	// DeltaY is the vertical scroll delta (positive = down).
	DeltaY int
	// Mods holds the active modifier keys.
	Mods Modifiers
}

// NewWheelEvent creates a WheelEvent.
func NewWheelEvent(screen geom.Point, dx, dy int, mods Modifiers) *WheelEvent {
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
	RelatedTarget EventTarget
}

// NewFocusEvent creates a FocusEvent of the given type.
func NewFocusEvent(typ EventType, related EventTarget) *FocusEvent {
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

// NewBaseEvent creates a BaseEvent with the given type and target.
func NewBaseEvent(typ EventType, target EventTarget, bubbles bool) *BaseEvent {
	return &BaseEvent{
		typ:     typ,
		target:  target,
		bubbles: bubbles,
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

// ChangeEvent is dispatched when the value of an input or textarea changes.
type ChangeEvent struct {
	BaseEvent
	Value string
}

// NewChange creates a ChangeEvent.
func NewChange(value string) *ChangeEvent {
	return &ChangeEvent{
		BaseEvent: BaseEvent{typ: EventChange, bubbles: true},
		Value:     value,
	}
}

// InputEvent is dispatched for every user-initiated change to an input or textarea value.
type InputEvent struct {
	BaseEvent
	Value string
}

// NewInput creates an InputEvent.
func NewInput(value string) *InputEvent {
	return &InputEvent{
		BaseEvent: BaseEvent{typ: EventInput, bubbles: true},
		Value:     value,
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

const (
	// MimeTextPlain is the standard MIME type for plain text.
	MimeTextPlain = "text/plain"
)

const (
	UnknownClipboard = 0   // An unknown or unsupported clipboard type.
	SystemClipboard  = 'c' // The system clipboard (e.g. Ctrl+C/Ctrl+V).
	PrimaryClipboard = 'p' // The primary selection (e.g. mouse selection on Linux).
)

// ClipboardEvent is dispatched for copy, cut, and paste clipboard operations.
// ClipboardEvent bubbles.
type ClipboardEvent struct {
	BaseEvent

	// ClipType identifies whether this is a copy, cut, or paste.
	ClipType ClipboardType

	// Items stores payloads keyed by their MIME type.
	// Common keys include "text/plain".
	Items map[string][]byte
}

// NewClipboardEvent creates a ClipboardEvent.
func NewClipboardEvent(typ EventType, ct ClipboardType) *ClipboardEvent {
	return &ClipboardEvent{
		BaseEvent: BaseEvent{typ: typ, bubbles: true},
		ClipType:  ct,
		Items:     make(map[string][]byte),
	}
}

// SetText sets the "text/plain" item.
func (c *ClipboardEvent) SetText(text string) {
	c.Items[MimeTextPlain] = []byte(text)
}

// Text returns the "text/plain" item as a string, or an empty string if not present.
func (c *ClipboardEvent) Text() string {
	if data, ok := c.Items[MimeTextPlain]; ok {
		return string(data)
	}
	return ""
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

// ScrollEvent is dispatched when an element's scroll offset changes.
// ScrollEvent bubbles.
type ScrollEvent struct {
	BaseEvent
	X, Y           int // New absolute offset
	DeltaX, DeltaY int // Change from previous offset
}

// NewScrollEvent creates a ScrollEvent with the given offset and delta.
func NewScrollEvent(x, y, dx, dy int) *ScrollEvent {
	return &ScrollEvent{
		BaseEvent: BaseEvent{typ: EventScroll, bubbles: true},
		X:         x,
		Y:         y,
		DeltaX:    dx,
		DeltaY:    dy,
	}
}

// SubmitEvent is dispatched when a form is submitted.
type SubmitEvent struct {
	BaseEvent
	FormData map[string]any
}

// NewSubmitEvent creates a SubmitEvent.
func NewSubmitEvent(formData map[string]any) *SubmitEvent {
	return &SubmitEvent{
		BaseEvent: BaseEvent{typ: EventSubmit, bubbles: true},
		FormData:  formData,
	}
}

// --- Interfaces and Target --------------------------------------------------

// Subscription is a cancellable event listener registration. Call Cancel to
// remove the listener from its target. Subscription values are safe to cancel
// from any goroutine; the actual removal is deferred to the next listener
// invocation to avoid map mutation under iteration.
type Subscription interface {
	// Cancel removes this listener registration. It is idempotent: calling
	// Cancel more than once is safe and has no additional effect.
	Cancel()
}

// Listener is a function that handles an Event.
type Listener func(Event)

// EventTarget is the interface implemented by objects that can receive events
// and have listeners registered on them.
type EventTarget interface {
	// AddEventListener registers fn as a listener for event of type typ on this
	// target. Options control the phase (capture vs bubble), auto-cancellation
	// (once), and the passive hint. The returned Subscription can be used to
	// remove the listener without pointer comparison.
	AddEventListener(typ EventType, fn Listener, opts ...Option) Subscription

	// DispatchTo fires listeners on this target for the given event. It
	// respects the phase and the once flag.
	DispatchTo(e Event)

	// DispatchToTarget invokes capture-registered listeners followed by
	// bubble-registered listeners for the target phase.
	DispatchToTarget(e Event)

	// RemoveRegistration removes the registration with the given id.
	RemoveRegistration(id uint64)

	// EventTarget returns the user-visible event target for this object.
	// For logical nodes in a UA shadow subtree, this returns the host element;
	// otherwise it returns the object itself.
	EventTarget() EventTarget
}

// Option configures how a listener is registered.
type Option func(any)

// OptionSetter is the interface implemented by internal registration objects
// to allow public options to configure them.
type OptionSetter interface {
	SetCapture(bool)
	SetOnce(bool)
	SetPassive(bool)
}

// Capture returns an Option that registers the listener for the capture phase
// (default is bubble phase).
func Capture() Option {
	return func(r any) {
		if s, ok := r.(OptionSetter); ok {
			s.SetCapture(true)
		}
	}
}

// Once returns an Option that causes the listener to auto-cancel after its
// first invocation.
func Once() Option {
	return func(r any) {
		if s, ok := r.(OptionSetter); ok {
			s.SetOnce(true)
		}
	}
}

// Passive returns an Option that hints the engine that the handler will never
// call PreventDefault (used for performance; not enforced).
func Passive() Option {
	return func(r any) {
		if s, ok := r.(OptionSetter); ok {
			s.SetPassive(true)
		}
	}
}

// --- Registration API -------------------------------------------------------

// HitTester resolves the event target at a screen-space point. This is
// typically implemented by the engine.
type HitTester interface {
	HitTest(x, y int) EventTarget
}

// Dispatcher performs 3-phase (capture → target → bubble) event dispatch.
type Dispatcher interface {
	// Dispatch routes e through the ancestor chain described by path.
	// path must be ordered root → target (index 0 = root, last = target).
	Dispatch(e Event, path []EventTarget)

	// DispatchWheel routes a WheelEvent through the ancestor chain, stopping at
	// the first ancestor that implements Scrollable.
	DispatchWheel(e *WheelEvent, path []EventTarget, scrollables map[EventTarget]Scrollable)
}

// Implementation defines the set of factory functions provided by the
// internal/event package.
type Implementation struct {
	NewDispatcher func() Dispatcher
	NewTarget     func() EventTarget
}

var impl Implementation

// RegisterImplementation registers the concrete implementation of events.
// This is called by internal/event's init() function.
func RegisterImplementation(i Implementation) {
	impl = i
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher() Dispatcher {
	if impl.NewDispatcher == nil {
		panic("event: implementation not registered. Did you forget to import internal/event?")
	}
	return impl.NewDispatcher()
}

// NewTarget creates a new EventTarget implementation.
func NewTarget() EventTarget {
	return impl.NewTarget()
}
