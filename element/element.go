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

	// SetStyle replaces the author-set style for this element and marks style dirty.
	SetStyle(s style.Style)

	// IsHidden reports whether the element is in the hidden state.
	IsHidden() bool

	// SetHidden sets or clears the element's hidden state and marks style dirty.
	SetHidden(v bool)

	// Listeners returns the set of event listeners registered on this element.
	Listeners() []PendingListener

	// DispatchEvent fires an event that propagates through the DOM tree.
	DispatchEvent(e event.Event)

	// SetCursorNavigable sets whether the element is cursor-navigable.
	SetCursorNavigable(bool)
	// IsCursorNavigable reports whether the element is cursor-navigable.
	IsCursorNavigable() bool
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
	// compound elements set this in their constructor. Default: empty style.S().
	// See ADR-010.
	intrinsicStyle style.Style

	// hidden is an intrinsic boolean state flag.
	hidden bool

	// listeners holds event registrations made via OnEvent.
	listeners []PendingListener

	cursorNavigable bool
	caret           *CaretController
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
// override them via Style(). Most elements return an empty style.S(). Replaced
// and compound elements return a sparse Style with UA-forced properties
// (e.g. Display:InlineBlock, OverflowX:Clip). See ADR-010.
func (b *elementBase[Self]) IntrinsicStyle() style.Style { return b.intrinsicStyle }

// IsDirtyStyle reports whether this element's style needs re-resolution.
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

// SetStyle replaces the element's style wholesale and marks style dirty.
func (b *elementBase[Self]) SetStyle(s style.Style) {
	b.Style(s)
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

// SetHidden sets or clears the hidden state and marks style dirty.
func (b *elementBase[Self]) SetHidden(v bool) {
	b.Hidden(v)
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
	for cur := any(b.self).(dom.Node); cur != nil; {
		path = append(path, cur)
		parent := cur.Parent()
		if parent == nil {
			// 1. Cross UA shadow boundary: jump from UARoot to host element.
			if et := cur.EventTarget(); et != nil {
				if host, ok := et.(dom.Node); ok && host != cur {
					cur = host
					continue
				}
			}
			// 2. Cross Overlay boundary: jump from Overlay to Anchor.
			if ov, ok := cur.(interface{ Anchor() dom.Element }); ok {
				if anchor := ov.Anchor(); anchor != nil {
					if aNode, ok := anchor.(dom.Node); ok {
						cur = aNode
						continue
					}
				}
			}
			// 3. Link to document.
			if doc := cur.OwnerDocument(); doc != nil && cur != doc {
				cur = doc
				continue
			}
			break
		}
		cur = parent
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
// event-target resolver during dispatch setup. Nil handlers are ignored.
func (b *elementBase[Self]) OnEvent(typ event.EventType, fn event.Listener) *Self {
	if fn == nil {
		return b.self
	}
	b.listeners = append(b.listeners, PendingListener{Typ: typ, Fn: fn})
	b.AddEventListener(typ, fn)
	return b.self
}

// OnClick registers a typed click listener and returns *Self.
func (b *elementBase[Self]) OnClick(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventClick, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnPress registers a zero-argument activation listener on click (including
// keyboard Enter/Space activation on buttons) and returns *Self.
func (b *elementBase[Self]) OnPress(fn func()) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventClick, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			if me.Button == event.ButtonLeft || me.Button == event.ButtonNone {
				fn()
			}
			return
		}
		fn()
	})
}

// OnMouseDown registers a typed mousedown listener and returns *Self.
func (b *elementBase[Self]) OnMouseDown(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventMouseDown, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnMouseUp registers a typed mouseup listener and returns *Self.
func (b *elementBase[Self]) OnMouseUp(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventMouseUp, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnMouseMove registers a typed mousemove listener and returns *Self.
func (b *elementBase[Self]) OnMouseMove(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventMouseMove, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnMouseEnter registers a typed mouseenter listener and returns *Self.
func (b *elementBase[Self]) OnMouseEnter(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventMouseEnter, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnMouseLeave registers a typed mouseleave listener and returns *Self.
func (b *elementBase[Self]) OnMouseLeave(fn func(*event.MouseEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventMouseLeave, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok {
			fn(me)
		}
	})
}

// OnKeyDown registers a typed keydown listener and returns *Self.
func (b *elementBase[Self]) OnKeyDown(fn func(*event.KeyEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			fn(ke)
		}
	})
}

// OnKeyUp registers a typed keyup listener and returns *Self.
func (b *elementBase[Self]) OnKeyUp(fn func(*event.KeyEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventKeyUp, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			fn(ke)
		}
	})
}

// OnKeyPress registers a typed keypress listener and returns *Self.
func (b *elementBase[Self]) OnKeyPress(fn func(*event.KeyEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventKeyPress, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			fn(ke)
		}
	})
}

// OnKey registers a keydown listener filtered by combo via [event.KeyEvent.MatchString]
// and returns *Self.
func (b *elementBase[Self]) OnKey(combo string, fn func(*event.KeyEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok && ke.MatchString(combo) {
			fn(ke)
		}
	})
}

// OnFocus registers a typed focus listener and returns *Self.
func (b *elementBase[Self]) OnFocus(fn func(*event.FocusEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventFocus, func(e event.Event) {
		if fe, ok := e.(*event.FocusEvent); ok {
			fn(fe)
		}
	})
}

// OnBlur registers a typed blur listener and returns *Self.
func (b *elementBase[Self]) OnBlur(fn func(*event.FocusEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventBlur, func(e event.Event) {
		if fe, ok := e.(*event.FocusEvent); ok {
			fn(fe)
		}
	})
}

// OnWheelEvent registers a typed wheel listener and returns *Self.
// Named OnWheelEvent to avoid colliding with [event.Scrollable]'s OnWheel method.
func (b *elementBase[Self]) OnWheelEvent(fn func(*event.WheelEvent)) *Self {
	if fn == nil {
		return b.self
	}
	return b.OnEvent(event.EventWheel, func(e event.Event) {
		if we, ok := e.(*event.WheelEvent); ok {
			fn(we)
		}
	})
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
// sets the UA-mandated forced style (ADR-010); pass style.S() or omit
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
