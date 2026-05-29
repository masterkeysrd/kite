package focus

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/style"
)

// Scope is a type alias for dom.FocusScope to preserve backwards compatibility.
type Scope = dom.FocusScope

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

// Manager owns focus state, the scope stack, and tab-navigation for a single
// logical tree. All methods must be called from the single main-loop goroutine.
//
// A Manager must be constructed with NewManager; the zero value is invalid.
type Manager struct {
	doc              dom.Document
	current          dom.Element
	reason           Reason
	scopes           []*dom.FocusScope
	dispatcher       event.Dispatcher
	initialFocusDone bool
}

var _ dom.FocusHandle = (*Manager)(nil)

// NewManager creates a Manager backed by dispatcher.
// A default root scope covering the entire tree (with root as Root) is
// established automatically via PushScope; this mirrors the engine startup
// contract.
//
// dispatcher routes focus / blur event through the logical tree.
func NewManager(
	root dom.Document,
	dispatcher event.Dispatcher,
) *Manager {
	m := &Manager{
		doc:        root,
		dispatcher: dispatcher,
	}
	m.PushScope(&dom.FocusScope{Root: root})
	return m
}

// --- Read accessors -----------------------------------------------------------

// Current returns the currently focused logical element, or nil.
func (m *Manager) Current() dom.Element { return m.current }

// Document returns the logical document root.
func (m *Manager) Document() dom.Document { return m.doc }

// Reason returns the reason the current focus was acquired.
func (m *Manager) Reason() Reason { return m.reason }

// IsFocused reports whether el is the currently focused element.
func (m *Manager) IsFocused(el dom.Element) bool { return el != nil && m.current == el }

// IsFocusVisible reports whether el is focused AND focus was acquired via the
// keyboard. Painters consult this to decide whether to draw a focus ring,
// matching the web :focus-visible heuristic.
func (m *Manager) IsFocusVisible(el dom.Element) bool {
	return m.IsFocused(el) && m.reason == ReasonKeyboard
}

// ActiveScopeInternal returns the top of the scope stack, or nil if the stack is
// empty (should not happen after NewManager).
func (m *Manager) ActiveScopeInternal() *dom.FocusScope {
	if len(m.scopes) == 0 {
		return nil
	}
	return m.scopes[len(m.scopes)-1]
}

// ActiveScope implements dom.FocusHandle.
func (m *Manager) ActiveScope() *dom.FocusScope {
	return m.ActiveScopeInternal()
}

// --- Mutation -----------------------------------------------------------------

// SetFocus attempts to move keyboard focus to el with the supplied reason.
// It returns false (and is a no-op) if el does not pass the focusable filter:
//
//   - el's IsFocusable() must be true
//   - if el implements dom.Disableable, IsDisabled() must be false
//   - its render object's ComputedStyle().Display must not be DisplayNone
//   - el must be a descendant of (or equal to) the active scope's Root
//
// When the focus changes, blur/focusout are dispatched on the previous
// target and focus/focusin are dispatched on el. DirtyPaint is marked on
// both the old and new nodes so painters can update focus rings.
func (m *Manager) SetFocus(el dom.Element, reason Reason) bool {
	if !m.IsFocusable(el, m.ActiveScopeInternal()) {
		return false
	}
	m.setFocus(el, reason)
	return true
}

// Focus implements dom.FocusHandle.
func (m *Manager) Focus(el dom.Element) {
	m.SetFocus(el, ReasonProgrammatic)
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
func (m *Manager) setFocus(next dom.Element, reason Reason) {
	if m.current == next {
		// Re-focusing the same element still updates the reason.
		m.reason = reason
		return
	}

	if next != nil {
		m.initialFocusDone = true
	}

	old := m.current
	m.current = next
	m.reason = reason

	// Dispatch lose-focus event on old node.
	if old != nil {
		if d := internaldom.AsDirty(old); d != nil {
			d.MarkNeedsSync()
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
		if d := internaldom.AsDirty(next); d != nil {
			d.MarkNeedsSync()
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
	scope := m.ActiveScopeInternal()
	if scope == nil {
		return false
	}
	candidates := m.collectFocusable(scope)
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
	scope := m.ActiveScopeInternal()
	if scope == nil {
		return false
	}
	candidates := m.collectFocusable(scope)
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
// PushScope pushes s onto the scope stack. The current focus is captured into
// s.PreviousFocus. If s.Autofocus is set and passes the focusable filter,
// focus is moved to it with ReasonProgrammatic.
func (m *Manager) PushScope(s *dom.FocusScope) {
	s.PreviousFocus = m.current
	m.scopes = append(m.scopes, s)
	if s.Autofocus != nil {
		if m.IsFocusable(s.Autofocus, s) {
			m.setFocus(s.Autofocus, ReasonProgrammatic)
		} else {
			// If the autofocus target itself is not focusable, walk its descendants
			// in pre-order to find the first focusable element.
			var firstFocusable dom.Element
			walkPreOrder(s.Autofocus, func(n dom.Node) {
				if firstFocusable != nil {
					return
				}
				if el, ok := n.(dom.Element); ok && el != s.Autofocus {
					if m.IsFocusable(el, s) {
						firstFocusable = el
					}
				}
			})
			if firstFocusable != nil {
				m.setFocus(firstFocusable, ReasonProgrammatic)
			}
		}
	}
}

// PopScope removes the top scope from the stack. If the captured
// PreviousFocus is still focusable (passes the new active scope's filter after
// the pop), focus is restored to it with ReasonRestore; otherwise Blur is
// called.
//
// PopScope is a no-op when only the root scope remains.
func (m *Manager) PopScope() {
	if len(m.scopes) <= 1 {
		// Never pop the root scope.
		return
	}
	top := m.scopes[len(m.scopes)-1]
	m.scopes = m.scopes[:len(m.scopes)-1]

	prev := top.PreviousFocus
	if prev != nil && m.IsFocusable(prev, m.ActiveScopeInternal()) {
		m.setFocus(prev, ReasonRestore)
	} else {
		m.setFocus(nil, ReasonProgrammatic)
	}
}

// ResetScope clears all scopes except the root, and restores focus to the root's PreviousFocus if possible. This is useful for recovering from unexpected states
// (e.g. focus trapped in a modal that failed to close).
func (m *Manager) ResetScope() {
	if len(m.scopes) == 0 {
		return
	}
	root := m.scopes[0]
	m.scopes = []*dom.FocusScope{root}
	m.setFocus(root.PreviousFocus, ReasonRestore)
	m.initialFocusDone = false
}

// SetInitialFocus establishes the initial focus on startup or mount.
// It is a no-op if initial focus has already been set since the last reset.
func (m *Manager) SetInitialFocus() {
	if m.initialFocusDone {
		return
	}
	m.initialFocusDone = true
	if m.current == nil {
		m.Next()
	}
}

// --- Focusable filter --------------------------------------------------------

// IsFocusable reports whether el is a valid focus target within scope.
// An element is focusable iff all four conditions hold:
//
//  1. el's IsFocusable() returns true.
//  2. If the element implements dom.Disableable, IsDisabled() must be false.
//  3. el's render object must exist and ComputedStyle().Display must not be DisplayNone.
//  4. el is a descendant of (or equal to) scope.Root
//
// IsFocusable is the single source of truth; all callers (Focus, Next,
// Previous, PushScope Autofocus) delegate to it.
func (m *Manager) IsFocusable(el dom.Element, scope *dom.FocusScope) bool {
	if el == nil {
		return false
	}

	// 1. Check IsFocusable()
	if !el.IsFocusable() {
		return false
	}

	// 2. Check Disableable interface
	if d, ok := el.(dom.Disableable); ok && d.IsDisabled() {
		return false
	}

	// 3. Check Render object and style if they exist.
	// If the render object hasn't been created yet (e.g. before the first
	// frame sync), we allow focus to be set based on logical state alone.
	// This ensures initial focus can be established at startup.
	if m.doc != nil && m.doc.DefaultView() != nil {
		cs := m.doc.DefaultView().GetComputedStyle(el)
		if cs != nil && cs.Display == style.DisplayNone {
			return false
		}
	}

	if scope == nil {
		return true
	}
	return isDescendantOrEqual(el, scope.Root)
}

// --- Helpers -----------------------------------------------------------------

// isDescendantOrEqual reports whether n is scope.Root or a descendant of it.
func isDescendantOrEqual(n, root dom.Node) bool {
	if root == nil {
		return true
	}

	rootBase := root
	for {
		if u := rootBase.Unwrap(); u != nil {
			rootBase = u
		} else {
			break
		}
	}

	for cur := n; cur != nil; cur = cur.Parent() {
		curBase := cur
		for {
			if u := curBase.Unwrap(); u != nil {
				curBase = u
			} else {
				break
			}
		}
		if curBase == rootBase {
			return true
		}
	}
	return false
}

// collectFocusable returns all focusable elements within scope in DOM tree order
// (depth-first, left-to-right pre-order walk of the scope's subtree).
func (m *Manager) collectFocusable(scope *dom.FocusScope) []dom.Element {
	if scope == nil || scope.Root == nil {
		return nil
	}
	var out []dom.Element
	walkPreOrder(scope.Root, func(n dom.Node) {
		if el, ok := n.(dom.Element); ok {
			if m.IsFocusable(el, scope) && el.TabIndex() >= 0 {
				out = append(out, el)
			}
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
func findNext(candidates []dom.Element, current dom.Element) dom.Element {
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
func findPrev(candidates []dom.Element, current dom.Element) dom.Element {
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
