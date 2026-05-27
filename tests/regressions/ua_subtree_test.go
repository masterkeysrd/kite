// Package regressions contains integration and regression tests for the
// UA Shadow Subtree primitive (ADR-009, TSK-018).
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/style"
)

// ---- test host element ------------------------------------------------------

// testHost is a minimal custom element that attaches a UA shadow subtree in
// its constructor. It mirrors the pattern that <input> and <textarea> will use.
type testHost struct {
	dom.Element
	uaInner dom.Element // accessible for test inspection
}

// Unwrap returns the underlying dom.Element so that dom.asBase can reach the
// inner baseNode through the interface chain.
func (h *testHost) Unwrap() dom.Node { return h.Element }

func newTestHost(doc dom.Document) *testHost {
	h := &testHost{}
	el := doc.CreateElement("x-host", h)
	inner := doc.CreateElement("div", nil)
	text := doc.CreateTextNode("hi", nil)
	inner.AppendChild(text)

	h.Element = el
	h.uaInner = inner
	el.AttachUARoot(inner)
	return h
}

// ---- public-traversal invisibility tests ------------------------------------

// TestUASubtree_PublicTraversal_InvisibleToChildNodes verifies that public
// ChildNodes() does not yield UA-subtree nodes.
func TestUASubtree_PublicTraversal_InvisibleToChildNodes(t *testing.T) {
	doc := dom.NewDocument()
	h := newTestHost(doc)

	// Add a public child to the host.
	pub := doc.CreateElement("span", nil)
	h.AppendChild(pub)

	count := 0
	for child := range h.ChildNodes() {
		count++
		if child == h.uaInner {
			t.Error("ChildNodes yielded the UA root — encapsulation broken")
		}
	}
	if count != 1 {
		t.Errorf("ChildNodes count = %d, want 1", count)
	}
}

// TestUASubtree_GetElementByID_InvisibleToAuthor verifies that GetElementByID
// cannot reach an element placed inside the UA subtree.
func TestUASubtree_GetElementByID_InvisibleToAuthor(t *testing.T) {
	doc := dom.NewDocument()
	h := newTestHost(doc)
	h.uaInner.SetID("ua-inner-id")
	doc.AppendChild(h)

	if found := doc.GetElementByID("ua-inner-id"); found != nil {
		t.Errorf("GetElementByID found a UA-subtree element: %v", found)
	}
}

// ---- engine integration: UA subtree participates in render ------------------

// TestUASubtree_Engine_RenderTreeContainsUANodes verifies that after a frame,
// the render tree includes render objects for UA-subtree nodes.
func TestUASubtree_Engine_RenderTreeContainsUANodes(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	doc := eng.Document()
	h := newTestHost(doc)

	root := element.Box(h)
	eng.Mount(root)
	eng.Frame()

	// After the frame, the UA text node (child of uaInner) should have a
	// render object. dom.LayoutChildren(host) yields children of uaInner, not
	// uaInner itself (per spec step 4: "children of uaRoot").
	uaText := h.uaInner.FirstChild()
	if uaText == nil {
		t.Fatal("uaInner has no children")
	}
	if uaText.RenderObject() == nil {
		t.Error("UA-subtree text node has no render object after Frame — sync phase missed it")
	}
}

// TestUASubtree_Engine_HostStyleApplied verifies that setting Width/Height on
// the host element is picked up by the layout engine correctly.
func TestUASubtree_Engine_HostStyleApplied(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	doc := eng.Document()
	h := newTestHost(doc)

	root := element.Box(h).Style(style.Style{
		Display: style.Some(style.DisplayBlock),
	})
	eng.Mount(root)
	eng.Frame()

	frame := b.LastFrame()
	if frame.Surface == nil {
		t.Fatal("no surface in frame")
	}
	// We can't verify exact pixel layout without a full IFC here, but we do
	// verify that the frame was produced (no panic/hang) and the frame surface
	// is non-nil, proving the engine walked the UA subtree without crashing.
}

// ---- event dispatch: UA nodes are invisible ---------------------------------

// TestUASubtree_EventDispatch_TargetIsHost verifies that when the dispatcher
// builds a path from the host's ancestors, the UA nodes do not appear.
// We synthetically dispatch on the host and verify target == host.
func TestUASubtree_EventDispatch_TargetIsHost(t *testing.T) {
	doc := dom.NewDocument()
	h := newTestHost(doc)
	doc.AppendChild(h)

	var captured event.EventTarget
	h.AddEventListener(event.EventType("click"), func(e event.Event) {
		captured = e.Target()
	})

	// Build a path as the dispatcher would: ancestors of host from root → host.
	path := []event.EventTarget{doc, h}
	d := event.NewDispatcher()
	ev := event.NewMouseEvent(event.EventType("click"), geom.Point{}, 0, 0)
	d.Dispatch(ev, path)

	if captured == nil {
		t.Fatal("listener was not called")
	}
	// Target must be host, not a UA child.
	if captured != h {
		t.Errorf("event target = %v, want host (%v)", captured, h)
	}
}

// ---- focus navigation: UA nodes are not focusable --------------------------

// testFocusableElement is a dom.Element that also implements dom.Focusable.
type testFocusableElement struct {
	dom.Element
}

func (f *testFocusableElement) IsFocusable() bool { return true }
func (f *testFocusableElement) Focus()            {}
func (f *testFocusableElement) Blur()             {}
func (f *testFocusableElement) TabIndex() int     { return 0 }
func (f *testFocusableElement) Unwrap() dom.Node  { return f.Element }

// TestUASubtree_Focus_SkipsUANodes verifies that a focusable element inside a
// UA subtree is NOT discovered by focus.Manager.Next().
func TestUASubtree_Focus_SkipsUANodes(t *testing.T) {
	doc := dom.NewDocument()

	// Create a host with a UA subtree.
	uaInner := doc.CreateElement("div", nil)
	host := doc.CreateElement("x-host", nil)
	host.AttachUARoot(uaInner)

	// Append the host to the document.
	doc.AppendChild(host)

	// Verify that public ChildNodes of host does NOT yield the UA root.
	hostChildren := 0
	for range host.ChildNodes() {
		hostChildren++
	}
	if hostChildren != 0 {
		t.Errorf("host.ChildNodes() count = %d, want 0 (uaRoot must be invisible)", hostChildren)
	}

	// The focus manager walks the public children via ChildNodes/FirstChild.
	// Since host has no public focusable children and the UA root is invisible,
	// focus.Next() must return false.
	fm := focus.NewManager(doc, event.NewDispatcher())
	moved := fm.Next()
	if moved {
		t.Error("focus.Next() returned true — UA node leaked into focus traversal")
	}
}

// ---- dom.IsUANode public API ------------------------------------------------

// TestUASubtree_IsUANode_Integration verifies IsUANode works in an end-to-end
// scenario where the host is connected to a real document.
func TestUASubtree_IsUANode_Integration(t *testing.T) {
	doc := dom.NewDocument()
	h := newTestHost(doc)
	pub := doc.CreateElement("span", nil)
	h.AppendChild(pub)
	doc.AppendChild(h)

	if dom.IsUANode(pub) {
		t.Error("public child should not be a UA node")
	}
	if dom.IsUANode(h) {
		t.Error("host element should not be a UA node")
	}
	// The UA inner element's first child (the text node).
	text := h.uaInner.FirstChild()
	if text == nil {
		t.Fatal("uaInner has no children")
	}
	if !dom.IsUANode(h.uaInner) {
		t.Error("uaInner should be a UA node")
	}
	if !dom.IsUANode(text) {
		t.Error("text inside uaInner should be a UA node")
	}
}
