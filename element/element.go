package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	_ "github.com/masterkeysrd/kite/internal/event"
	"github.com/masterkeysrd/kite/style"
)

// Element is the interface satisfied by every native element type. It extends
// [dom.Element] so wrapper elements can participate directly in logical-tree
// APIs while still exposing builder-oriented style and state helpers.
type Element interface {
	dom.Element

	// RawStyle returns the author-set style for this element.
	RawStyle() style.Style

	// IsHidden reports whether the element is in the hidden state.
	IsHidden() bool

	// Listeners returns the set of event listeners registered on this element.
	Listeners() []PendingListener

	// DispatchEvent fires an event that propagates through the DOM tree.
	DispatchEvent(e event.Event)
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
	// Style();
	rawStyle style.Style

	// defaultStyle holds the element-type default style.
	defaultStyle style.Style

	// intrinsicStyle holds the UA-mandated forced style for this element.
	// Properties set here cannot be overridden by the author. Replaced and
	// compound elements set this in their constructor. Default: empty Style{}.
	// See ADR-010.
	intrinsicStyle style.Style

	// hidden is an intrinsic boolean state flag.
	hidden bool

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

// Unwrap returns the underlying [dom.Node].
func (b *elementBase[Self]) Unwrap() dom.Node { return b.Element }

// RawStyle returns the author-set style for this element.
func (b *elementBase[Self]) RawStyle() style.Style { return b.rawStyle }

// DefaultStyle returns the element-type default style.
func (b *elementBase[Self]) DefaultStyle() style.Style { return b.defaultStyle }

// IntrinsicStyle returns the UA-mandated forced style for this element.
// Properties set here have the highest cascade precedence; authors cannot
// override them via Style(). Most elements return an empty Style{}. Replaced
// and compound elements return a sparse Style with UA-forced properties
// (e.g. Display:InlineBlock, OverflowX:Clip). See ADR-010.
func (b *elementBase[Self]) IntrinsicStyle() style.Style { return b.intrinsicStyle }

// IsDirtyStyle implements style.StyleNode by delegating to the internal flag.
func (b *elementBase[Self]) IsDirtyStyle() bool {
	if d := internaldom.AsDirtyElement(b.Element); d != nil {
		return d.IsDirtyStyle()
	}
	return false
}

// IsHidden reports whether the element is hidden.
func (b *elementBase[Self]) IsHidden() bool { return b.hidden }

// --- Fluent modifier methods (return *Self for type-precise chaining) --------

// Style replaces the element's style wholesale and returns *Self.
// Calling Style twice discards the first call; style composition belongs at
// the [style.Style] value level via Style.Merge.
func (b *elementBase[Self]) Style(s style.Style) *Self {
	b.rawStyle = s
	if d := internaldom.AsDirtyElement(b.Element); d != nil {
		d.MarkStyleDirty()
	}
	return b.self
}

// WithClass sets the string classification tag and returns *Self.
func (b *elementBase[Self]) WithClass(class string) *Self {
	b.SetClass(class)
	return b.self
}

// WithID sets the element's identifier and returns *Self.
func (b *elementBase[Self]) WithID(id string) *Self {
	b.SetID(id)
	return b.self
}

// WithTabIndex sets the tab index on the element and returns *Self.
func (b *elementBase[Self]) WithTabIndex(index int) *Self {
	b.SetTabIndex(index)
	return b.self
}

// Hidden sets or clears the hidden state and returns *Self.
func (b *elementBase[Self]) Hidden(v bool) *Self {
	b.hidden = v
	if d := internaldom.AsDirtyElement(b.Element); d != nil {
		d.MarkStyleDirty()
	}
	return b.self
}

// ScrollbarX enables or disables the horizontal scrollbar and returns *Self.
func (b *elementBase[Self]) ScrollbarX(v bool) *Self {
	b.rawStyle = b.rawStyle.ScrollbarX(v)
	if d := internaldom.AsDirtyElement(b.Element); d != nil {
		d.MarkStyleDirty()
	}
	return b.self
}

// ScrollbarY enables or disables the vertical scrollbar and returns *Self.
func (b *elementBase[Self]) ScrollbarY(v bool) *Self {
	b.rawStyle = b.rawStyle.ScrollbarY(v)
	if d := internaldom.AsDirtyElement(b.Element); d != nil {
		d.MarkStyleDirty()
	}
	return b.self
}

// Listeners returns the set of event listeners registered on this element.
func (b *elementBase[Self]) Listeners() []PendingListener {
	return b.listeners
}

// DispatchEvent fires an event that propagates through the DOM tree.
func (b *elementBase[Self]) DispatchEvent(e event.Event) {
	// Build the ancestor path for dispatch (root -> target).
	var path []event.EventTarget
	for p := dom.Node(b.Element); p != nil; p = p.Parent() {
		path = append(path, p.EventTarget())
	}
	// Reverse the path.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	dispatcher := event.NewDispatcher()
	dispatcher.Dispatch(e, path)
}

// OnEvent registers fn as a listener for event of type typ and returns *Self.
// Listeners are stored on the element and retrieved by the engine's
// event-target resolver during dispatch setup.
func (b *elementBase[Self]) OnEvent(typ event.EventType, fn event.Listener) *Self {
	b.listeners = append(b.listeners, PendingListener{Typ: typ, Fn: fn})
	b.AddEventListener(typ, fn)
	return b.self
}

// EventListeners returns the event listener registrations for this element.
// Used by the engine's event-target resolver to wire up dispatch.
func (b *elementBase[Self]) EventListeners() []PendingListener { return b.listeners }

// AddChild adds child as the last child of this element and returns the element itself
// for fluent chaining.
func (b *elementBase[Self]) AddChild(child dom.Node) *Self {
	b.AppendChild(child)
	return b.self
}

// --- Internal helpers --------------------------------------------------------

// initBase initialises b with the given DOM element, self-pointer and default style.
// Must be called exactly once, at the end of each element's constructor,
// before any modifier methods run. The optional intrinsicStyle parameter
// sets the UA-mandated forced style (ADR-010); pass style.Style{} or omit
// when no UA properties need to be forced.
func (b *elementBase[Self]) initBase(el dom.Element, self *Self, defaultStyle style.Style, intrinsicStyle ...style.Style) {
	b.Element = el
	b.self = self
	b.defaultStyle = defaultStyle
	if len(intrinsicStyle) > 0 {
		b.intrinsicStyle = intrinsicStyle[0]
	}
}

var orphanDocument = dom.NewDocument()

func processChildren(parent dom.Element, children []any) {
	for _, child := range children {
		if child == nil {
			continue
		}

		switch v := child.(type) {
		case string:
			parent.AppendChild(NewText(orphanDocument, v))
		case dom.Node:
			parent.AppendChild(v)
		case []any:
			processChildren(parent, v)
		case []dom.Node:
			for _, n := range v {
				parent.AppendChild(n)
			}
		case style.Style:
			// Special case: if parent is one of our elements, it has Style method.
			// Given the objective, let's focus on children first.
		}
	}
}
