package testenv

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

// ElementAssertion wraps a node and the testing context for fluent assertions.
type ElementAssertion struct {
	t    *testing.T
	node dom.Node
}

// NewElementAssertion creates a new assertion helper for the given node.
func NewElementAssertion(t *testing.T, node dom.Node) *ElementAssertion {
	return &ElementAssertion{t: t, node: node}
}

// Expect is a convenience alias for NewElementAssertion to provide a BDD-style
// entry point for test assertions.
func Expect(t *testing.T, node dom.Node) *ElementAssertion {
	return NewElementAssertion(t, node)
}

// ToHaveChildCount asserts that the node has exactly expected direct children.
func (a *ElementAssertion) ToHaveChildCount(expected int, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	count := 0
	for range a.node.ChildNodes() {
		count++
	}
	if count != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected %d children, got %d%s", expected, count, suffix)
	}
	return a
}

// ToHaveChildrenText asserts that the node's direct children have the given
// text contents in document order.
func (a *ElementAssertion) ToHaveChildrenText(expected []string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	got := make([]string, 0, len(expected))
	for child := range a.node.ChildNodes() {
		got = append(got, child.TextContent())
	}
	suffix := ""
	if len(msgs) > 0 {
		suffix = ": " + fmt.Sprint(msgs...)
	}
	if len(got) != len(expected) {
		a.t.Fatalf("expected %d children text entries, got %d%s", len(expected), len(got), suffix)
	}
	for i := range expected {
		if got[i] != expected[i] {
			a.t.Fatalf("child %d text = %q, want %q%s", i, got[i], expected[i], suffix)
		}
	}
	return a
}

// ToHaveCursorAt specifically tests components that implement a cursor state.
func (a *ElementAssertion) ToHaveCursorAt(x, y int, msgs ...any) *ElementAssertion {
	a.t.Helper()

	// Provider that exposes CursorState()
	type cursorProvider interface {
		CursorState() cursor.State
	}

	if a.node == nil {
		a.t.Fatalf("node is nil")
	}

	// Try direct assertion first, then try Unwrap() for wrapped nodes.
	var provider cursorProvider
	if p, ok := a.node.(cursorProvider); ok {
		provider = p
	} else if un := a.node.Unwrap(); un != nil {
		if p2, ok2 := un.(cursorProvider); ok2 {
			provider = p2
		}
	}

	if provider == nil {
		a.t.Fatalf("expected node to implement CursorState provider, but it does not")
	}

	cs := provider.CursorState()
	if cs.X != x || cs.Y != y {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Errorf("expected cursor at (%d, %d), got (%d, %d)%s", x, y, cs.X, cs.Y, suffix)
	}

	return a
}

type fragmentProvider interface {
	GetFragment(dom.Node) *layout.Fragment
}

func (a *ElementAssertion) getFragment() *layout.Fragment {
	if a.node == nil {
		return nil
	}
	doc := a.node.OwnerDocument()
	if doc == nil {
		return nil
	}
	view := doc.DefaultView()
	if view == nil {
		return nil
	}
	if fp, ok := view.(fragmentProvider); ok {
		return fp.GetFragment(a.node)
	}
	return nil
}

// ToHaveFragmentHeight asserts that the node's render fragment has the
// expected height in layout units (cells/rows). Useful for verifying
// that elements respect their configured height and do not overflow.
func (a *ElementAssertion) ToHaveFragmentHeight(expected int, msgs ...any) *ElementAssertion {
	a.t.Helper()
	frag := a.getFragment()
	if frag == nil {
		a.t.Fatalf("layout fragment is nil for node")
	}

	if frag.Size.Height != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("fragment height = %d, want %d%s", frag.Size.Height, expected, suffix)
	}

	return a
}

// ExpectHardwareCursorVisible asserts that the backend's hardware cursor is visible.
func (a *ElementAssertion) ExpectHardwareCursorVisible(env *Environment, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if env == nil || env.Backend == nil {
		a.t.Fatalf("environment or backend is nil")
	}
	if !env.Backend.Cursor.Visible {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Errorf("hardware cursor not visible%s", suffix)
	}
	return a
}

// ExpectHardwareCursorY asserts the Y position of the backend cursor.
func (a *ElementAssertion) ExpectHardwareCursorY(env *Environment, want int, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if env == nil || env.Backend == nil {
		a.t.Fatalf("environment or backend is nil")
	}
	got := env.Backend.Cursor.Y
	if got != want {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Errorf("hardware cursor Y = %d, want %d%s", got, want, suffix)
	}
	return a
}

// ToHaveFocus asserts that this node is the currently focused node in env.
func (a *ElementAssertion) ToHaveFocus(env *Environment, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if env == nil {
		a.t.Fatalf("environment is nil")
	}
	if env.CurrentFocus() != a.node {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected node to be focused, got %v%s", env.CurrentFocus(), suffix)
	}
	return a
}

// ToNotHaveFocus asserts that this node is not the currently focused node.
func (a *ElementAssertion) ToNotHaveFocus(env *Environment, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if env == nil {
		a.t.Fatalf("environment is nil")
	}
	if env.CurrentFocus() == a.node {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected node to not be focused%s", suffix)
	}
	return a
}

// ToHaveTableStructure verifies that the table's section fragments appear in
// the specified order (e.g. "thead", "tbody", "tfoot"). It returns the
// same ElementAssertion to allow chaining further table-related checks.
func (a *ElementAssertion) ToHaveTableStructure(expected []string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	frag := a.getFragment()
	if frag == nil {
		a.t.Fatalf("layout fragment is nil for node")
	}

	// Map section names to style.Display values
	nameToDisplay := map[string]style.Display{
		"thead": style.DisplayTableHeaderGroup,
		"tbody": style.DisplayTableRowGroup,
		"tfoot": style.DisplayTableFooterGroup,
	}

	if len(frag.Children) < len(expected) {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected at least %d section fragments, got %d%s", len(expected), len(frag.Children), suffix)
	}

	for i, name := range expected {
		wantDisp, ok := nameToDisplay[name]
		if !ok {
			a.t.Fatalf("unknown section name %q", name)
		}
		child := frag.Children[i].Fragment
		if child == nil || child.Node == nil || child.Node.Style() == nil {
			a.t.Fatalf("invalid child fragment at index %d", i)
		}
		if child.Node.Style().Display != wantDisp {
			suffix := ""
			if len(msgs) > 0 {
				suffix = ": " + fmt.Sprint(msgs...)
			}
			a.t.Fatalf("section %d = %v, want %v%s", i, child.Node.Style().Display, wantDisp, suffix)
		}
	}

	return a
}

// ColumnAssertion provides fluent assertions for a single table column.
type ColumnAssertion struct {
	t      *testing.T
	widths []int
}

// CellsInColumn collects the physical widths of cells in the given column
// index across all rows in the table's fragment and returns a ColumnAssertion
// for further checks (e.g. ToHaveEqualWidth).
func (a *ElementAssertion) CellsInColumn(col int) *ColumnAssertion {
	a.t.Helper()
	var widths []int
	frag := a.getFragment()
	if frag == nil {
		a.t.Fatalf("layout fragment is nil for node")
	}

	// Iterate over section fragments -> row fragments -> collect cell at index col
	for _, secLink := range frag.Children {
		sec := secLink.Fragment
		if sec == nil {
			continue
		}
		for _, rowLink := range sec.Children {
			row := rowLink.Fragment
			if row == nil {
				continue
			}
			if col < 0 || col >= len(row.Children) {
				// skip rows that don't have this column
				continue
			}
			cellFrag := row.Children[col].Fragment
			if cellFrag == nil {
				continue
			}
			widths = append(widths, cellFrag.Size.Width)
		}
	}

	return &ColumnAssertion{t: a.t, widths: widths}
}

// ToHaveEqualWidth asserts that all collected cell widths are equal.
func (c *ColumnAssertion) ToHaveEqualWidth(msgs ...any) *ColumnAssertion {
	c.t.Helper()
	if len(c.widths) == 0 {
		c.t.Fatalf("no cells collected for column")
	}
	want := c.widths[0]
	for i, w := range c.widths[1:] {
		if w != want {
			suffix := ""
			if len(msgs) > 0 {
				suffix = ": " + fmt.Sprint(msgs...)
			}
			c.t.Fatalf("column widths differ at index %d: %d vs %d%s", i+1, w, want, suffix)
		}
	}
	return c
}

// ToHaveCellContentInFrame scans the last painted framebuffer for a cell whose
// content equals the expected string. Useful for asserting border junctions
// and other painted characters.
func (a *ElementAssertion) ToHaveCellContentInFrame(env *Environment, expected string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if env == nil || env.Backend == nil {
		a.t.Fatalf("environment or backend is nil")
	}
	fr := env.Backend.LastFrame()
	if fr.Surface == nil {
		a.t.Fatalf("no frame produced by backend")
	}
	fb := fr.Surface
	bounds := fb.Bounds()
	found := false
	for y := bounds.Origin.Y; y < bounds.Origin.Y+bounds.Size.Height; y++ {
		for x := bounds.Origin.X; x < bounds.Origin.X+bounds.Size.Width; x++ {
			if fb.CellAt(x, y).Content == expected {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected to find %q in last frame%s", expected, suffix)
	}
	return a
}

// ToHaveScroll asserts that the element reports the given scroll offsets.
func (a *ElementAssertion) ToHaveScroll(x, y int, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}

	// Prefer the node itself, but fall back to Unwrap() if it's a wrapper.
	var el dom.Element
	if e, ok := a.node.(dom.Element); ok {
		el = e
	} else if un := a.node.Unwrap(); un != nil {
		if e2, ok2 := un.(dom.Element); ok2 {
			el = e2
		}
	}

	if el == nil {
		a.t.Fatalf("node does not implement dom.Element (cannot check scroll)")
	}

	cx, cy := el.Scroll()
	if cx != x || cy != y {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected scroll (%d, %d), got (%d, %d)%s", x, y, cx, cy, suffix)
	}
	return a
}

// EventAssertion provides helpers for asserting that events fire on targets.
type EventAssertion struct {
	t         *testing.T
	target    event.EventTarget
	eventType event.EventType
}

// ExpectEvent creates an assertion for event listeners.
func ExpectEvent(t *testing.T, target event.EventTarget, eventType event.EventType) *EventAssertion {
	return &EventAssertion{t: t, target: target, eventType: eventType}
}

// ToFireWhen executes the action and asserts the event was received.
func (ea *EventAssertion) ToFireWhen(action func()) {
	ea.t.Helper()
	fired := false
	sub := ea.target.AddEventListener(ea.eventType, func(e event.Event) {
		fired = true
	}, event.Once())

	action()

	if !fired {
		ea.t.Errorf("expected event %q to fire, but it did not", ea.eventType)
	}
	if sub != nil {
		sub.Cancel()
	}
}

// ToHaveTextContent asserts that the node's TextContent() equals the expected string.
func (a *ElementAssertion) ToHaveTextContent(expected string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	got := a.node.TextContent()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected text content %q, got %q%s", expected, got, suffix)
	}
	return a
}

// ToBeChecked asserts that the node (or its underlying unwrapped node) is a Checkbox or Radio element and matches the expected checked state.
func (a *ElementAssertion) ToBeChecked(expected bool, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	type checkedProvider interface {
		Checked() bool
	}
	var provider checkedProvider
	if p, ok := a.node.(checkedProvider); ok {
		provider = p
	} else if un := a.node.Unwrap(); un != nil {
		if p2, ok2 := un.(checkedProvider); ok2 {
			provider = p2
		}
	}
	if provider == nil {
		a.t.Fatalf("expected node to support Checked state, but it does not")
	}
	got := provider.Checked()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected checked = %t, got %t%s", expected, got, suffix)
	}
	return a
}

// ToBeDisabled asserts that the node (or its underlying unwrapped node) is a form control or interactive element and matches the expected disabled state.
func (a *ElementAssertion) ToBeDisabled(expected bool, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	type disabledProvider interface {
		IsDisabled() bool
	}
	var provider disabledProvider
	if p, ok := a.node.(disabledProvider); ok {
		provider = p
	} else if un := a.node.Unwrap(); un != nil {
		if p2, ok2 := un.(disabledProvider); ok2 {
			provider = p2
		}
	}
	if provider == nil {
		a.t.Fatalf("expected node to support IsDisabled check, but it does not")
	}
	got := provider.IsDisabled()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected disabled = %t, got %t%s", expected, got, suffix)
	}
	return a
}

// ToHaveValue asserts that the node is a FormControl (or its underlying unwrapped node) and its Value() equals the expected value.
func (a *ElementAssertion) ToHaveValue(expected any, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	var control dom.FormControl
	if p, ok := a.node.(dom.FormControl); ok {
		control = p
	} else if un := a.node.Unwrap(); un != nil {
		if p2, ok2 := un.(dom.FormControl); ok2 {
			control = p2
		}
	}
	if control == nil {
		a.t.Fatalf("expected node to be a dom.FormControl, but it is not")
	}
	got := control.Value()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected value %v, got %v%s", expected, got, suffix)
	}
	return a
}

// ToHaveID asserts that the node is a dom.Element (or its underlying unwrapped node) and has the expected ID.
func (a *ElementAssertion) ToHaveID(expected string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	var el dom.Element
	if e, ok := a.node.(dom.Element); ok {
		el = e
	} else if un := a.node.Unwrap(); un != nil {
		if e2, ok2 := un.(dom.Element); ok2 {
			el = e2
		}
	}
	if el == nil {
		a.t.Fatalf("node is not an Element (cannot check ID)")
	}
	got := el.ID()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected ID %q, got %q%s", expected, got, suffix)
	}
	return a
}

// ToHaveClass asserts that the node is a dom.Element (or its underlying unwrapped node) and has the expected Class.
func (a *ElementAssertion) ToHaveClass(expected string, msgs ...any) *ElementAssertion {
	a.t.Helper()
	if a.node == nil {
		a.t.Fatalf("node is nil")
	}
	var el dom.Element
	if e, ok := a.node.(dom.Element); ok {
		el = e
	} else if un := a.node.Unwrap(); un != nil {
		if e2, ok2 := un.(dom.Element); ok2 {
			el = e2
		}
	}
	if el == nil {
		a.t.Fatalf("node is not an Element (cannot check Class)")
	}
	got := el.Class()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		a.t.Fatalf("expected Class %q, got %q%s", expected, got, suffix)
	}
	return a
}

// Eventually polls the given predicate function until it returns true or the timeout is reached.
// If the timeout is reached, the test fails with a description.
func Eventually(t *testing.T, predicate func() bool, timeout time.Duration, pollInterval ...time.Duration) {
	t.Helper()
	interval := 10 * time.Millisecond
	if len(pollInterval) > 0 {
		interval = pollInterval[0]
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(interval)
	}

	// Try one last time
	if predicate() {
		return
	}

	t.Fatalf("condition not met within %v", timeout)
}

// EventSpy records occurrences of a specific event type on a target.
type EventSpy struct {
	t      *testing.T
	events []event.Event
	mu     sync.Mutex
}

// SpyEvents attaches a listener to target for the given eventType and returns an EventSpy.
func SpyEvents(t *testing.T, target event.EventTarget, eventType event.EventType) *EventSpy {
	spy := &EventSpy{t: t}
	target.AddEventListener(eventType, func(e event.Event) {
		spy.mu.Lock()
		defer spy.mu.Unlock()
		spy.events = append(spy.events, e)
	})
	return spy
}

// Count returns the number of events captured.
func (s *EventSpy) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

// Events returns the captured events.
func (s *EventSpy) Events() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]event.Event, len(s.events))
	copy(copied, s.events)
	return copied
}

// AssertFired asserts that the event was fired at least once.
func (s *EventSpy) AssertFired(msgs ...any) *EventSpy {
	s.t.Helper()
	if s.Count() == 0 {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		s.t.Fatalf("expected event to fire, but it did not%s", suffix)
	}
	return s
}

// AssertFiredCount asserts that the event was fired exactly expected times.
func (s *EventSpy) AssertFiredCount(expected int, msgs ...any) *EventSpy {
	s.t.Helper()
	got := s.Count()
	if got != expected {
		suffix := ""
		if len(msgs) > 0 {
			suffix = ": " + fmt.Sprint(msgs...)
		}
		s.t.Fatalf("expected event to fire %d times, got %d%s", expected, got, suffix)
	}
	return s
}
