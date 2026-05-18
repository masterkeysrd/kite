package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Element is the interface satisfied by every native element type. It extends
// [dom.Element] so wrapper elements can participate directly in logical-tree
// APIs while still exposing builder-oriented style and state helpers.
type Element interface {
	dom.Element

	// GetStyle returns the author-set style for this element.
	GetStyle() style.Style

	// IsDisabled reports whether the element is in the disabled state.
	IsDisabled() bool

	// IsHidden reports whether the element is in the hidden state.
	IsHidden() bool

	// Listeners returns the set of event listeners registered on this element.
	Listeners() []PendingListener
}

// elementBase is the shared data and method set embedded by every concrete
// native element type. The type parameter Self is the concrete element type,
// which allows modifier methods to return *Self for type-precise chaining.
//
// Pattern: every concrete type T embeds elementBase[T] and sets self=self
// inside its constructor (see box.go for a worked example).
//
// The self-pointer is the only slightly clever part of this design; it is
// documented here so that contributors can extend elements without confusion.
type elementBase[Self any] struct {
	// self is the back-pointer to the concrete *Self value. Set once in
	// the constructor; must not be changed after construction.
	self *Self

	// el is the underlying DOM element. It is created once at construction
	// time by calling Document.CreateElement or, when no document is
	// available, by using the orphan document.
	dom.Element

	// rawStyle holds the author-set sparse style. Replaced wholesale by
	// Style(); see ADR-0012 §6.
	rawStyle style.Style

	// class is the string classification tag; no selector engine is implied.
	class string

	// disabled and hidden are intrinsic boolean state flags.
	disabled bool
	hidden   bool

	// listeners holds event registrations made via OnEvent.
	listeners []PendingListener
}

// PendingListener records an event registration from OnEvent().
// It is exported so that the engine's event-target resolver and tests can
// inspect registered listeners without accessing unexported fields.
type PendingListener struct {
	// Typ is the event type this listener is registered for.
	Typ event.EventType
	// Fn is the listener function.
	Fn event.Listener
}

// --- Element interface implementation ----------------------------------------

// DOMElement returns the underlying [dom.Element].
func (b *elementBase[Self]) DOMElement() dom.Element { return b.Element }

// GetStyle returns the author-set style for this element.
func (b *elementBase[Self]) GetStyle() style.Style { return b.rawStyle }

// IsDisabled reports whether the element is disabled.
func (b *elementBase[Self]) IsDisabled() bool { return b.disabled }

// IsHidden reports whether the element is hidden.
func (b *elementBase[Self]) IsHidden() bool { return b.hidden }

// --- Fluent modifier methods (return *Self for type-precise chaining) --------

// Style replaces the element's style wholesale and returns *Self.
// Calling Style twice discards the first call; style composition belongs at
// the [style.Style] value level via Style.Merge (see ADR-0012 §6).
func (b *elementBase[Self]) Style(s style.Style) *Self {
	b.rawStyle = s
	if ro := b.Element.RenderObject(); ro != nil {
		ro.SetRawStyle(s)
	}
	return b.self
}

// setIDModifier sets the element's identifier and returns *Self. Concrete
// types expose this as their public ID(string) *Self modifier, which shadows
// the dom.Element.ID() string getter on the concrete type level. Callers
// that need the getter use GetID().
func (b *elementBase[Self]) setIDModifier(id string) *Self {
	b.Element.SetID(id)
	return b.self
}

// WithClass sets the string classification tag and returns *Self.
func (b *elementBase[Self]) WithClass(class string) *Self {
	b.class = class
	return b.self
}

// Disabled sets or clears the disabled state and returns *Self.
func (b *elementBase[Self]) Disabled(v bool) *Self {
	b.disabled = v
	if ro := b.Element.RenderObject(); ro != nil {
		ro.SetDisabled(v)
	}
	return b.self
}

// Hidden sets or clears the hidden state and returns *Self.
func (b *elementBase[Self]) Hidden(v bool) *Self {
	b.hidden = v
	return b.self
}

// Listeners returns the set of event listeners registered on this element.
func (b *elementBase[Self]) Listeners() []PendingListener {
	return b.listeners
}

// OnEvent registers fn as a listener for event of type typ and returns *Self.
// Listeners are stored on the element and retrieved by the engine's
// event-target resolver during dispatch setup.
func (b *elementBase[Self]) OnEvent(typ event.EventType, fn event.Listener) *Self {
	b.listeners = append(b.listeners, PendingListener{Typ: typ, Fn: fn})
	return b.self
}

// EventListeners returns the event listener registrations for this element.
// Used by the engine's event-target resolver to wire up dispatch.
func (b *elementBase[Self]) EventListeners() []PendingListener { return b.listeners }

// --- Internal helpers --------------------------------------------------------

// initBase initialises b with the given DOM element and self-pointer.
// Must be called exactly once, at the end of each element's constructor,
// before any modifier methods run.
func (b *elementBase[Self]) initBase(el dom.Element, self *Self) {
	b.Element = el
	b.self = self
}
