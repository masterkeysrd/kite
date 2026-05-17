package dom

import (
	"testing"
)

// lifecycleNode is a test-only Node that records OnConnected / OnDisconnected
// events. It embeds *element directly so the unexported linkable methods are
// promoted automatically, making it insertable into the live tree.
//
// This type lives in the internal test file so it can access *element.
type lifecycleNode struct {
	*element
	events *[]string
}

func (l *lifecycleNode) OnConnected() {
	*l.events = append(*l.events, "connected:"+l.tagName)
}

func (l *lifecycleNode) OnDisconnected() {
	*l.events = append(*l.events, "disconnected:"+l.tagName)
}

// coreEl satisfies the coreElement interface so that asElement() can unwrap
// this test wrapper during the attach/detach walks.
func (l *lifecycleNode) coreEl() *element { return l.element }

// newLifecycleNode creates a lifecycleNode with its own events slice.
func newLifecycleNode(doc Document, tag string) (*lifecycleNode, *[]string) {
	events := &[]string{}
	inner := newElement(tag, doc)
	ln := &lifecycleNode{element: inner, events: events}
	// Set the outer pointer to the wrapper so adoption works correctly.
	inner.outer = ln
	return ln, events
}

// --- adoption ---------------------------------------------------------------

func TestElement_AppendChild_AdoptsInnerToOuter(t *testing.T) {
	doc := NewDocument()
	lc, _ := newLifecycleNode(doc, "div")

	doc.AppendChild(lc)

	if !lc.IsConnected() {
		t.Error("outer wrapper must be connected after AppendChild")
	}
}

// --- connection predicate ---------------------------------------------------

func TestDocument_IsConnected(t *testing.T) {
	doc := NewDocument()
	if !doc.IsConnected() {
		t.Error("document must always be connected")
	}
}

func TestElement_AppendChild_SetsConnected(t *testing.T) {
	doc := NewDocument()
	el := newElement("div", doc)

	if el.IsConnected() {
		t.Error("detached element must not be connected")
	}
	doc.AppendChild(el)
	if !el.IsConnected() {
		t.Error("element must be connected after AppendChild to document")
	}
}

func TestElement_RemoveChild_ClearsConnected(t *testing.T) {
	doc := NewDocument()
	el := newElement("div", doc)
	doc.AppendChild(el)
	doc.RemoveChild(el)

	if el.IsConnected() {
		t.Error("element must not be connected after RemoveChild")
	}
}

func TestElement_AppendChild_DetachedTree_LeavesRegistryEmpty(t *testing.T) {
	// Build a subtree off-document; the ID should not appear in the registry
	// until the subtree is attached to a connected ancestor.
	doc := NewDocument()
	root := newElement("div", doc)
	child := newElement("span", doc)
	child.SetID("greeting")

	root.AppendChild(child)

	if doc.GetElementByID("greeting") != nil {
		t.Error("ID registry must be empty for a detached subtree")
	}

	doc.AppendChild(root)
	if got := doc.GetElementByID("greeting"); got != child {
		t.Errorf("GetElementByID after attach = %v, want %v", got, child)
	}
}

func TestElement_AppendChild_DetachedTree_StillAdopts(t *testing.T) {
	// Adoption (outer back-pointer) runs even when the parent is not connected.
	doc := NewDocument()
	root := newElement("div", doc)
	lc, events := newLifecycleNode(doc, "span")

	root.AppendChild(lc)

	if root.IsConnected() {
		t.Error("root must not be connected before it is appended to the document")
	}
	if lc.IsConnected() {
		t.Error("lc must not be connected while its parent is detached")
	}
	if len(*events) != 0 {
		t.Errorf("no lifecycle events expected for detached subtree, got %v", *events)
	}
}

func TestElement_RemoveChild_PreservesOuter(t *testing.T) {
	// Detaching a node must not reset its outer back-pointer.
	doc := NewDocument()
	lc, _ := newLifecycleNode(doc, "div")

	doc.AppendChild(lc)
	doc.RemoveChild(lc)

	// Re-attaching should work correctly.
	doc.AppendChild(lc)
	if !lc.IsConnected() {
		t.Error("element must be re-connected after second AppendChild")
	}
}

// --- SetID before/after attach ----------------------------------------------

func TestElement_SetID_BeforeAttach_RegistersOnAttach(t *testing.T) {
	doc := NewDocument()
	el := newElement("div", doc)
	el.SetID("hero") // disconnected → registry untouched

	if doc.GetElementByID("hero") != nil {
		t.Error("registry must be empty before attach")
	}
	doc.AppendChild(el)
	if got := doc.GetElementByID("hero"); got != el {
		t.Errorf("GetElementByID after attach = %v, want %v", got, el)
	}
}

func TestElement_SetID_AfterDetach_NoOpOnRegistry(t *testing.T) {
	doc := NewDocument()
	el := newElement("div", doc)
	el.SetID("target")
	doc.AppendChild(el)
	doc.RemoveChild(el)

	// Changing ID on a disconnected node must not touch the registry.
	el.SetID("other")
	if doc.GetElementByID("other") != nil {
		t.Error("SetID on disconnected node must not register in the ID map")
	}
	if doc.GetElementByID("target") != nil {
		t.Error("old ID must have been removed when the node was detached")
	}
}

func TestElement_RemoveChild_UnregistersSubtree(t *testing.T) {
	doc := NewDocument()
	parent := newElement("div", doc)
	child := newElement("span", doc)
	child.SetID("child-id")
	parent.AppendChild(child)
	doc.AppendChild(parent)

	if doc.GetElementByID("child-id") == nil {
		t.Fatal("child-id should be registered after subtree is connected")
	}
	doc.RemoveChild(parent)
	if doc.GetElementByID("child-id") != nil {
		t.Error("child-id must be unregistered when subtree is detached")
	}
}

// --- lifecycle callbacks ----------------------------------------------------

func TestElement_Lifecycle_OnConnected_FiresPreOrder(t *testing.T) {
	doc := NewDocument()
	events := &[]string{}

	// Share the events slice so we can observe all callbacks in order.
	makeLC := func(tag string) *lifecycleNode {
		inner := newElement(tag, doc)
		ln := &lifecycleNode{element: inner, events: events}
		inner.outer = ln
		return ln
	}

	parent := makeLC("parent")
	child := makeLC("child")
	grandchild := makeLC("grandchild")

	parent.AppendChild(child)
	child.AppendChild(grandchild)
	doc.AppendChild(parent)

	want := []string{
		"connected:parent",
		"connected:child",
		"connected:grandchild",
	}
	checkEvents(t, *events, want)
}

func TestElement_Lifecycle_OnDisconnected_FiresPostOrder(t *testing.T) {
	doc := NewDocument()
	events := &[]string{}

	makeLC := func(tag string) *lifecycleNode {
		inner := newElement(tag, doc)
		ln := &lifecycleNode{element: inner, events: events}
		inner.outer = ln
		return ln
	}

	parent := makeLC("parent")
	child := makeLC("child")
	grandchild := makeLC("grandchild")

	parent.AppendChild(child)
	child.AppendChild(grandchild)
	doc.AppendChild(parent)

	// Reset to capture only disconnect events.
	*events = (*events)[:0]
	doc.RemoveChild(parent)

	want := []string{
		"disconnected:grandchild",
		"disconnected:child",
		"disconnected:parent",
	}
	checkEvents(t, *events, want)
}

func TestElement_Lifecycle_SelfMutation_Allowed(t *testing.T) {
	// Appending a child to a connected node inside AppendChild (post-attach)
	// must connect the new child immediately.
	doc := NewDocument()
	parent := newElement("box", doc)
	doc.AppendChild(parent)

	extra := newElement("extra", doc)
	parent.AppendChild(extra)

	if !extra.IsConnected() {
		t.Error("child appended to a connected parent must itself be connected")
	}
}

// --- move (detach + re-attach) ----------------------------------------------

func TestElement_Move_ObservesDisconnectThenConnect(t *testing.T) {
	doc := NewDocument()
	events := &[]string{}

	srcParent := newElement("src", doc)
	dstParent := newElement("dst", doc)

	inner := newElement("mover", doc)
	lc := &lifecycleNode{element: inner, events: events}
	inner.outer = lc

	doc.AppendChild(srcParent)
	doc.AppendChild(dstParent)
	srcParent.AppendChild(lc)

	// Reset events captured during first attach.
	*events = (*events)[:0]

	// Moving: AppendChild detects existing parent → detachWalk then attachWalk.
	dstParent.AppendChild(lc)

	want := []string{"disconnected:mover", "connected:mover"}
	checkEvents(t, *events, want)
}

// --- cross-document append --------------------------------------------------

func TestElement_CrossDocumentAppend_PanicsInDev(t *testing.T) {
	doc1 := NewDocument()
	doc2 := NewDocument()
	foreign := doc2.CreateElement("div")

	defer func() {
		if r := recover(); r == nil {
			t.Error("cross-document append must panic")
		}
	}()
	doc1.AppendChild(foreign)
}

// --- anchor registry --------------------------------------------------------

func TestAnchor_Name_BeforeAttach_RegistersOnAttach(t *testing.T) {
	doc := NewDocument()
	el := newElement("anchor", doc)

	if doc.FindAnchor("nav") != nil {
		t.Error("anchor must not be in registry before attach")
	}
	doc.AppendChild(el)
	doc.RegisterAnchor("nav", el)
	if got := doc.FindAnchor("nav"); got != el {
		t.Errorf("FindAnchor after register = %v, want %v", got, el)
	}
	doc.RemoveChild(el)
	doc.UnregisterAnchor("nav")
	if doc.FindAnchor("nav") != nil {
		t.Error("anchor must not be in registry after unregister")
	}
}

// --- RegistersSubtreeIDs ---------------------------------------------------

func TestElement_AppendChild_RegistersSubtreeIDs(t *testing.T) {
	// Attaching a whole subtree at once must register all IDs in the walk.
	doc := NewDocument()
	root := newElement("div", doc)
	a := newElement("a", doc)
	b := newElement("b", doc)
	a.SetID("node-a")
	b.SetID("node-b")
	root.AppendChild(a)
	root.AppendChild(b)

	// IDs not registered yet (detached).
	if doc.GetElementByID("node-a") != nil || doc.GetElementByID("node-b") != nil {
		t.Error("IDs must not be in registry while subtree is detached")
	}
	doc.AppendChild(root)
	if got := doc.GetElementByID("node-a"); got != a {
		t.Errorf("node-a: got %v, want %v", got, a)
	}
	if got := doc.GetElementByID("node-b"); got != b {
		t.Errorf("node-b: got %v, want %v", got, b)
	}
}

// --- helpers ----------------------------------------------------------------

func checkEvents(t *testing.T, got, want []string) {
	t.Helper()
	for i, ev := range got {
		if i >= len(want) {
			t.Errorf("extra event[%d] = %q", i, ev)
			continue
		}
		if ev != want[i] {
			t.Errorf("event[%d] = %q, want %q", i, ev, want[i])
		}
	}
	if len(got) != len(want) {
		t.Errorf("event count: got %d, want %d; full: %v", len(got), len(want), got)
	}
}
