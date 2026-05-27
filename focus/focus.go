package focus

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

// Reason describes how focus was acquired on the current element. It is used
// by painters to decide whether to draw a visible focus indicator (ring).
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
type Scope struct {
	// Root is the logical node that acts as the boundary for tab navigation
	// and focus queries while this scope is active. Must not be nil.
	Root dom.Node

	// Autofocus is the initial focus target when the scope is pushed.
	// If nil no autofocus is applied; focus stays on the previous element
	// until a navigation or programmatic Focus call.
	Autofocus dom.Node

	// PreviousFocus is captured automatically on PushScope. It is restored
	// with ReasonRestore when PopScope is called. Do not set this field
	// manually; Manager writes it on PushScope.
	PreviousFocus dom.Node
}

// Manager owns focus state, the scope stack, and tab-navigation for a single
// logical tree. All methods must be called from the single main-loop goroutine.
//
// A Manager must be constructed with NewManager; the zero value is invalid.
type Manager struct {
	current    dom.Node
	reason     Reason
	scopes     []*Scope
	dispatcher *event.Dispatcher
}

// NewManager creates a Manager backed by dispatcher.
// A default root scope covering the entire tree (with root as Root) is
// established automatically via PushScope; this mirrors the engine startup
// contract.
//
// dispatcher routes focus / blur event through the logical tree.
func NewManager(
	root dom.Node,
	dispatcher *event.Dispatcher,
) *Manager {
	m := &Manager{
		dispatcher: dispatcher,
	}
	m.PushScope(&Scope{Root: root})
	return m
}

// --- Read accessors -----------------------------------------------------------

// Current returns the currently focused logical node, or nil.
func (m *Manager) Current() dom.Node { return m.current }

// Reason returns the reason the current focus was acquired.
func (m *Manager) Reason() Reason { return m.reason }

// IsFocused reports whether n is the currently focused node.
func (m *Manager) IsFocused(n dom.Node) bool { return n != nil && m.current == n }

// IsFocusVisible reports whether n is focused AND focus was acquired via the
// keyboard. Painters consult this to decide whether to draw a focus ring,
// matching the web :focus-visible heuristic.
func (m *Manager) IsFocusVisible(n dom.Node) bool {
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
//   - n must implement dom.Focusable and IsFocusable() must be true
//   - if n implements dom.Disableable, IsDisabled() must be false
//   - its render object's ComputedStyle().Display must not be DisplayNone
//   - n must be a descendant of (or equal to) the active scope's Root
//
// When the focus changes, blur/focusout are dispatched on the previous
// target and focus/focusin are dispatched on n. DirtyPaint is marked on
// both the old and new nodes so painters can update focus rings.
func (m *Manager) Focus(n dom.Node, reason Reason) bool {
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
func (m *Manager) setFocus(next dom.Node, reason Reason) {
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
		if ro := old.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyPaint)
		}
		path := ancestorPath(old)
		etNext := event.EventTarget(nil)
		if next != nil {
			etNext = next
		}
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventBlur, etNext), path)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocusOut, etNext), path)
	}

	// Dispatch gain-focus event on new node.
	if next != nil {
		if ro := next.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyPaint)
		}
		path := ancestorPath(next)
		etOld := event.EventTarget(nil)
		if old != nil {
			etOld = old
		}
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocus, etOld), path)
		m.dispatcher.Dispatch(event.NewFocusEvent(event.EventFocusIn, etOld), path)
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

// ResetScope clears all scopes except the root, and restores focus to the root's PreviousFocus if possible. This is useful for recovering from unexpected states
// (e.g. focus trapped in a modal that failed to close).
func (m *Manager) ResetScope() {
	if len(m.scopes) == 0 {
		return
	}
	root := m.scopes[0]
	m.scopes = []*Scope{root}
	m.setFocus(root.PreviousFocus, ReasonRestore)
}

// --- Focusable filter --------------------------------------------------------

// IsFocusable reports whether n is a valid focus target within scope.
// A node is focusable iff all four conditions hold:
//
//  1. The logical node implements dom.Focusable and IsFocusable() returns true.
//  2. If the logical node implements dom.Disableable, IsDisabled() must be false.
//  3. n's render object must exist and ComputedStyle().Display must not be DisplayNone.
//  4. n is a descendant of (or equal to) scope.Root
//
// IsFocusable is the single source of truth; all callers (Focus, Next,
// Previous, PushScope Autofocus) delegate to it.
func IsFocusable(n dom.Node, scope *Scope) bool {
	if n == nil {
		return false
	}

	// 1. Check Focusable interface
	f, ok := n.(dom.Focusable)
	if !ok || !f.IsFocusable() {
		return false
	}

	// 2. Check Disableable interface
	if d, ok := n.(dom.Disableable); ok && d.IsDisabled() {
		return false
	}

	// 3. Check Render object and style if they exist.
	// If the render object hasn't been created yet (e.g. before the first
	// frame sync), we allow focus to be set based on logical state alone.
	// This ensures initial focus can be established at startup.
	if ro := n.RenderObject(); ro != nil {
		cs := ro.ComputedStyle()
		if cs != nil && cs.Display == style.DisplayNone {
			return false
		}
	}

	if scope == nil {
		return true
	}
	return isDescendantOrEqual(n, scope.Root)
}

// --- Helpers -----------------------------------------------------------------

// isDescendantOrEqual reports whether n is scope.Root or a descendant of it.
func isDescendantOrEqual(n, root dom.Node) bool {
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

// collectFocusable returns all focusable nodes within scope in DOM tree order
// (depth-first, left-to-right pre-order walk of the scope's subtree).
func collectFocusable(scope *Scope) []dom.Node {
	if scope == nil || scope.Root == nil {
		return nil
	}
	var out []dom.Node
	walkPreOrder(scope.Root, func(n dom.Node) {
		if IsFocusable(n, scope) {
			out = append(out, n)
		}
	})
	return out
}

// walkPreOrder performs a depth-first pre-order walk of the subtree rooted at
// root, calling fn on every node including root.
func walkPreOrder(root dom.Node, fn func(dom.Node)) {
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
func findNext(candidates []dom.Node, current dom.Node) dom.Node {
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
func findPrev(candidates []dom.Node, current dom.Node) dom.Node {
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
func ancestorPath(n dom.Node) []event.EventTarget {
	var chain []event.EventTarget
	for cur := n; cur != nil; cur = cur.Parent() {
		chain = append(chain, cur)
	}
	// Reverse to get root → n order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}
