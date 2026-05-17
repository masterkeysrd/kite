package focus

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// Reason describes how focus was acquired on the current element. It is used
// by painters to decide whether to draw a visible focus indicator (ring).
//
// See ADR-0010 §Reason.
type Reason uint8

const (
	// ReasonProgrammatic means the application called Focus() directly.
	ReasonProgrammatic Reason = iota
	// ReasonPointer means the element was focused via mouse or touch.
	ReasonPointer
	// ReasonKeyboard means the element was focused via the keyboard (Tab,
	// arrow keys, keyboard shortcut). Painters should show a focus ring.
	ReasonKeyboard
	// ReasonRestore means focus was restored automatically when a Scope was
	// popped (e.g. a modal was closed).
	ReasonRestore
)

// Scope is a focus containment region. While a Scope is active, tab navigation
// and focusable-filter queries are restricted to the subtree rooted at Root.
//
// Lifecycle:
//   - PushScope captures the current focus into PreviousFocus.
//   - PopScope restores PreviousFocus with ReasonRestore, or blurs if the
//     previous node is no longer focusable.
//
// See ADR-0010 §Scope.
type Scope struct {
	// Root is the render object that acts as the boundary for tab navigation
	// and focus queries while this scope is active. Must not be nil.
	Root render.Object

	// Autofocus is the initial focus target when the scope is pushed.
	// If nil no autofocus is applied; focus stays on the previous element
	// until a navigation or programmatic Focus call.
	Autofocus render.Object

	// PreviousFocus is captured automatically on PushScope. It is restored
	// with ReasonRestore when PopScope is called. Do not set this field
	// manually; Manager writes it on PushScope.
	PreviousFocus render.Object
}

// Manager owns focus state, the scope stack, and tab-navigation for a single
// render tree. All methods must be called from the single main-loop goroutine.
//
// A Manager must be constructed with NewManager; the zero value is invalid.
//
// See ADR-0010 §Manager.
type Manager struct {
	current    render.Object
	reason     Reason
	scopes     []*Scope
	dispatcher *event.Dispatcher
	resolver   event.EventTargetResolver
}

// NewManager creates a Manager backed by dispatcher and resolver.
// A default root scope covering the entire tree (with root as Root) is
// established automatically via PushScope; this mirrors the engine startup
// contract described in ADR-0010.
//
// resolver maps a render.Object to its EventTarget (may return nil).
// dispatcher routes focus / blur event through the render tree.
func NewManager(
	root render.Object,
	dispatcher *event.Dispatcher,
	resolver event.EventTargetResolver,
) *Manager {
	m := &Manager{
		dispatcher: dispatcher,
		resolver:   resolver,
	}
	m.PushScope(&Scope{Root: root})
	return m
}

// --- Read accessors -----------------------------------------------------------

// Current returns the currently focused render object, or nil.
func (m *Manager) Current() render.Object { return m.current }

// Reason returns the reason the current focus was acquired.
func (m *Manager) Reason() Reason { return m.reason }

// IsFocused reports whether n is the currently focused object.
func (m *Manager) IsFocused(n render.Object) bool { return n != nil && m.current == n }

// IsFocusVisible reports whether n is focused AND focus was acquired via the
// keyboard. Painters consult this to decide whether to draw a focus ring,
// matching the web :focus-visible heuristic.
func (m *Manager) IsFocusVisible(n render.Object) bool {
	return m.IsFocused(n) && m.reason == ReasonKeyboard
}

// ActiveScope returns the top of the scope stack, or nil if the stack is
// empty (should not happen after NewManager).
func (m *Manager) ActiveScope() *Scope {
	if len(m.scopes) == 0 {
		return nil
	}
	return m.scopes[len(m.scopes)-1]
}

// --- Mutation -----------------------------------------------------------------

// Focus attempts to move keyboard focus to n with the supplied reason.
// It returns false (and is a no-op) if n does not pass the focusable filter:
//
//   - n.Focusable() must be true
//   - n.Disabled() must be false
//   - n.ComputedStyle().Display must not be DisplayNone
//   - n must be a descendant of (or equal to) the active scope's Root
//
// When the focus changes, blur/focusout are dispatched on the previous
// target and focus/focusin are dispatched on n. DirtyPaint is marked on
// both the old and new nodes so painters can update focus rings.
func (m *Manager) Focus(n render.Object, reason Reason) bool {
	if !IsFocusable(n, m.ActiveScope()) {
		return false
	}
	m.setFocus(n, reason)
	return true
}

// Blur clears the current focus without moving it to another element.
// blur/focusout event are dispatched on the previously focused node.
func (m *Manager) Blur() {
	if m.current == nil {
		return
	}
	m.setFocus(nil, ReasonProgrammatic)
}

// setFocus is the single point through which all focus state changes flow.
// It emits focus event and marks paint-dirty on the affected nodes.
func (m *Manager) setFocus(next render.Object, reason Reason) {
	if m.current == next {
		// Re-focusing the same element still updates the reason.
		m.reason = reason
		return
	}

	old := m.current
	m.current = next
	m.reason = reason

	// Dispatch lose-focus event on old node.
	if old != nil {
		old.MarkDirty(render.DirtyPaint)
		path := ancestorPath(old)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventBlur, next), path)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocusOut, next), path)
	}

	// Dispatch gain-focus event on new node.
	if next != nil {
		next.MarkDirty(render.DirtyPaint)
		path := ancestorPath(next)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocus, old), path)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocusIn, old), path)
	}
}

// --- Tab navigation -----------------------------------------------------------

// Next moves focus to the next focusable node in DOM tree order (depth-first,
// left-to-right) within the active scope, wrapping to the first focusable if
// at the end. Returns false if no focusable candidate exists.
func (m *Manager) Next() bool {
	scope := m.ActiveScope()
	if scope == nil {
		return false
	}
	candidates := collectFocusable(scope)
	if len(candidates) == 0 {
		return false
	}
	next := findNext(candidates, m.current)
	m.setFocus(next, ReasonKeyboard)
	return true
}

// Previous moves focus to the previous focusable node in DOM tree order within
// the active scope, wrapping to the last focusable if at the beginning. Returns
// false if no focusable candidate exists.
func (m *Manager) Previous() bool {
	scope := m.ActiveScope()
	if scope == nil {
		return false
	}
	candidates := collectFocusable(scope)
	if len(candidates) == 0 {
		return false
	}
	prev := findPrev(candidates, m.current)
	m.setFocus(prev, ReasonKeyboard)
	return true
}

// --- Scope stack -------------------------------------------------------------

// PushScope pushes s onto the scope stack. The current focus is captured into
// s.PreviousFocus. If s.Autofocus is set and passes the focusable filter,
// focus is moved to it with ReasonProgrammatic.
func (m *Manager) PushScope(s *Scope) {
	s.PreviousFocus = m.current
	m.scopes = append(m.scopes, s)
	if s.Autofocus != nil && IsFocusable(s.Autofocus, s) {
		m.setFocus(s.Autofocus, ReasonProgrammatic)
	}
}

// PopScope removes the top scope from the stack and returns it. If the
// captured PreviousFocus is still focusable (passes the new active scope's
// filter after the pop), focus is restored to it with ReasonRestore; otherwise
// Blur is called.
//
// PopScope is a no-op (returns nil) when only the root scope remains.
func (m *Manager) PopScope() *Scope {
	if len(m.scopes) <= 1 {
		// Never pop the root scope.
		return nil
	}
	top := m.scopes[len(m.scopes)-1]
	m.scopes = m.scopes[:len(m.scopes)-1]

	prev := top.PreviousFocus
	if prev != nil && IsFocusable(prev, m.ActiveScope()) {
		m.setFocus(prev, ReasonRestore)
	} else {
		m.setFocus(nil, ReasonProgrammatic)
	}
	return top
}

// --- Focusable filter --------------------------------------------------------

// IsFocusable reports whether n is a valid focus target within scope.
// A node is focusable iff all four conditions hold:
//
//  1. n.Focusable() == true
//  2. n.Disabled() == false
//  3. n.ComputedStyle() != nil && .Display != DisplayNone
//  4. n is a descendant of (or equal to) scope.Root
//
// IsFocusable is the single source of truth; all callers (Focus, Next,
// Previous, PushScope Autofocus) delegate to it.
func IsFocusable(n render.Object, scope *Scope) bool {
	if n == nil {
		return false
	}
	if !n.Focusable() {
		return false
	}
	if n.Disabled() {
		return false
	}
	cs := n.ComputedStyle()
	if cs == nil || cs.Display == style.DisplayNone {
		return false
	}
	if scope == nil {
		return true
	}
	return isDescendantOrEqual(n, scope.Root)
}

// --- Helpers -----------------------------------------------------------------

// isDescendantOrEqual reports whether n is scope.Root or a descendant of it.
func isDescendantOrEqual(n, root render.Object) bool {
	if root == nil {
		return true
	}
	for cur := n; cur != nil; cur = cur.Parent() {
		if cur == root {
			return true
		}
	}
	return false
}

// collectFocusable returns all focusable objects within scope in DOM tree order
// (depth-first, left-to-right pre-order walk of the scope's subtree).
func collectFocusable(scope *Scope) []render.Object {
	if scope == nil || scope.Root == nil {
		return nil
	}
	var out []render.Object
	walkPreOrder(scope.Root, func(n render.Object) {
		if IsFocusable(n, scope) {
			out = append(out, n)
		}
	})
	return out
}

// walkPreOrder performs a depth-first pre-order walk of the subtree rooted at
// root, calling fn on every node including root.
func walkPreOrder(root render.Object, fn func(render.Object)) {
	if root == nil {
		return
	}
	fn(root)
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		walkPreOrder(c, fn)
	}
}

// findNext returns the element that follows current in candidates (wrapping
// around). If current is nil or not in the list, it returns the first element.
func findNext(candidates []render.Object, current render.Object) render.Object {
	if current == nil {
		return candidates[0]
	}
	for i, c := range candidates {
		if c == current {
			return candidates[(i+1)%len(candidates)]
		}
	}
	// current not in candidates (moved outside scope) — start from beginning.
	return candidates[0]
}

// findPrev returns the element that precedes current in candidates (wrapping
// around). If current is nil or not in the list, it returns the last element.
func findPrev(candidates []render.Object, current render.Object) render.Object {
	if current == nil {
		return candidates[len(candidates)-1]
	}
	for i, c := range candidates {
		if c == current {
			idx := (i - 1 + len(candidates)) % len(candidates)
			return candidates[idx]
		}
	}
	// current not in candidates — start from the end.
	return candidates[len(candidates)-1]
}

// ancestorPath returns the ancestor chain from the tree root down to n
// (inclusive), ordered root → n. This is the path format required by
// event.Dispatcher.Dispatch.
func ancestorPath(n render.Object) []render.Object {
	var chain []render.Object
	for cur := n; cur != nil; cur = cur.Parent() {
		chain = append(chain, cur)
	}
	// Reverse to get root → n order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}
