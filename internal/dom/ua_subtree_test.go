package dom

// ua_subtree_test.go — unit tests for the UA Shadow Subtree primitive (ADR-009).
//
// This file uses the internal package (package dom) to access *element
// directly so tests can inspect uaRoot and self fields without a public API.

import (
	"github.com/masterkeysrd/kite/dom"
	"slices"
	"testing"
)

// ---- helpers ----------------------------------------------------------------

// collectNodes returns a slice of all nodes yielded by LayoutChildren(host).
func collectLayoutChildren(host dom.Node) []dom.Node {
	var out []dom.Node
	for n := range LayoutChildren(host) {
		out = append(out, n)
	}
	return out
}

// collectPublicChildren returns a slice of all nodes yielded by n.ChildNodes().
func collectPublicChildren(host dom.Node) []dom.Node {
	var out []dom.Node
	for n := range host.ChildNodes() {
		out = append(out, n)
	}
	return out
}

// ---- tests ------------------------------------------------------------------

// TestAttachUARoot_SetsOuterOnRoot verifies that AttachUARoot sets the outer
// back-pointer on the UA root itself.
func TestAttachUARoot_SetsOuterOnRoot(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil).(*Element)
	uaBox := doc.CreateElement("div", nil).(*Element)

	host.AttachUARoot(uaBox)

	if uaBox.outer != host {
		t.Errorf("uaRoot.self = %v, want host (%v)", uaBox.outer, host)
	}
}

// TestAttachUARoot_SetsOuterRecursively verifies that AttachUARoot sets the
// outer pointer on every descendant, including grandchildren.
func TestAttachUARoot_SetsOuterRecursively(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil).(*Element)
	uaBox := doc.CreateElement("div", nil).(*Element)
	inner := doc.CreateElement("span", nil).(*Element)
	text := newTextNode("hi", doc, nil)

	uaBox.AppendChild(inner)
	inner.AppendChild(text)

	host.AttachUARoot(uaBox)

	// uaBox, inner, and text must all have their outer == host.
	if uaBox.outer != host {
		t.Errorf("uaBox.self = %v, want host", uaBox.outer)
	}
	if inner.outer != host {
		t.Errorf("inner.self = %v, want host", inner.outer)
	}
	if asBase(text).outer != host {
		t.Errorf("text.self = %v, want host", asBase(text).outer)
	}
}

// TestAttachUARoot_ChildrenHidesUARoot verifies that host.ChildNodes() does
// NOT yield the UA root or its descendants.
func TestAttachUARoot_ChildrenHidesUARoot(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	pub1 := doc.CreateElement("label", nil)
	uaBox := doc.CreateElement("div", nil)
	inner := doc.CreateElement("span", nil)
	uaBox.AppendChild(inner)
	host.AppendChild(pub1)

	host.AttachUARoot(uaBox)

	got := collectPublicChildren(host)
	if len(got) != 1 || got[0] != pub1 {
		t.Errorf("ChildNodes after AttachUARoot: got %v, want [pub1]", got)
	}
}

// TestAttachUARoot_LayoutChildrenUnion verifies that LayoutChildren yields
// public children first, then UA root's children.
func TestAttachUARoot_LayoutChildrenUnion(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	pub1 := doc.CreateElement("label", nil)
	pub2 := doc.CreateElement("icon", nil)
	uaBox := doc.CreateElement("div", nil)
	uaChild1 := doc.CreateElement("span", nil)
	uaChild2 := doc.CreateElement("b", nil)
	uaBox.AppendChild(uaChild1)
	uaBox.AppendChild(uaChild2)

	host.AppendChild(pub1)
	host.AppendChild(pub2)
	host.AttachUARoot(uaBox)

	got := collectLayoutChildren(host)
	want := []dom.Node{pub1, pub2, uaChild1, uaChild2}
	if !slices.Equal(got, want) {
		t.Errorf("LayoutChildren = %v, want %v", got, want)
	}
}

// TestLayoutChildren_NoUASubtree verifies zero overhead path for nodes
// without a UA subtree — iterator must yield only public children.
func TestLayoutChildren_NoUASubtree(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div", nil)
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	parent.AppendChild(a)
	parent.AppendChild(b)

	got := collectLayoutChildren(parent)
	want := []dom.Node{a, b}
	if !slices.Equal(got, want) {
		t.Errorf("LayoutChildren (no UA) = %v, want %v", got, want)
	}
}

// TestGetElementByID_DoesNotFindUANodes verifies that GetElementByID cannot
// find an element placed inside the UA subtree.
func TestGetElementByID_DoesNotFindUANodes(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	uaBox := doc.CreateElement("div", nil)
	uaInner := doc.CreateElement("span", nil)
	uaInner.SetID("ua-secret")
	uaBox.AppendChild(uaInner)

	host.AttachUARoot(uaBox)
	doc.AppendChild(host)

	if found := doc.GetElementByID("ua-secret"); found != nil {
		t.Errorf("GetElementByID found a UA-subtree node: %v", found)
	}
}

// TestIsUANode_TrueForUADescendant verifies IsUANode returns true for nodes
// inside a UA subtree.
func TestIsUANode_TrueForUADescendant(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	uaBox := doc.CreateElement("div", nil)
	uaInner := doc.CreateElement("span", nil)
	uaBox.AppendChild(uaInner)

	host.AttachUARoot(uaBox)

	// uaBox is the ua root itself — it is a child of uaRoot? Actually uaRoot
	// IS uaBox. IsUANode checks parent links, so uaBox's parent is nil
	// (it's not in the public tree). uaInner's parent is uaBox.
	// The function checks if any public-tree ancestor holds the node's
	// ancestor-or-self as its uaRoot.
	//
	// uaInner → parent = uaBox. uaBox has no public parent.
	// But uaBox IS uaRoot of host. We need to ensure IsUANode can detect
	// that uaInner is inside host's uaRoot subtree.
	//
	// IsUANode walks cur.Parent() for cur starting at n. For uaInner:
	//   cur=uaInner → parent=uaBox → pe is host's uaRoot==uaBox → isInSubtree(uaInner, uaBox)==true ✓
	if !IsUANode(uaInner) {
		t.Error("IsUANode should be true for UA-subtree node")
	}

	// uaBox itself: cur=uaBox → parent=nil. We need to check if parent of uaBox
	// is the host element. But uaBox isn't in the public tree, so parent is nil.
	// IsUANode would return false for the root itself since it has no parent.
	// This is acceptable — the root only has host as its "logical" parent.
}

// TestIsUANode_FalseForPublicDescendant verifies IsUANode returns false for
// regular public children.
func TestIsUANode_FalseForPublicDescendant(t *testing.T) {
	doc := NewDocument()
	parent := doc.CreateElement("div", nil)
	child := doc.CreateElement("span", nil)
	parent.AppendChild(child)

	if IsUANode(child) {
		t.Error("IsUANode should be false for a public child")
	}
}

// TestIsUANode_FalseForNil verifies IsUANode returns false for a nil node.
func TestIsUANode_FalseForNil(t *testing.T) {
	if IsUANode(nil) {
		t.Error("IsUANode(nil) should be false")
	}
}

// TestAttachUARoot_MarksNeedsSync verifies that AttachUARoot marks the host
// as NeedsSync so the engine picks up the new subtree.
func TestAttachUARoot_MarksNeedsSync(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	// Clear flags that may be set from document insertion.
	AsDirty(host).ClearSyncFlags()

	uaBox := doc.CreateElement("div", nil)
	host.AttachUARoot(uaBox)

	if !AsDirty(host).NeedsSync() {
		t.Error("host.NeedsSync() should be true after AttachUARoot")
	}
}

// TestAttachUARoot_PanicsOnDouble verifies that calling AttachUARoot twice panics.
func TestAttachUARoot_PanicsOnDouble(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	uaBox1 := doc.CreateElement("div", nil)
	uaBox2 := doc.CreateElement("div", nil)
	host.AttachUARoot(uaBox1)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on double AttachUARoot")
		}
	}()
	host.AttachUARoot(uaBox2)
}

// TestAttachUARoot_DetachPreservesUARoot verifies that detaching the host from
// its document does NOT nil uaRoot (identity remains stable).
func TestAttachUARoot_DetachPreservesUARoot(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	uaBox := doc.CreateElement("div", nil)
	host.AttachUARoot(uaBox)

	doc.AppendChild(host)
	doc.RemoveChild(host)

	// uaRoot must still be reachable via UARoot helper.
	if UARoot(host) == nil {
		t.Error("uaRoot must be preserved after detaching the host")
	}
}

// TestLayoutChildren_EmptyUA verifies that a host with an empty UA subtree
// (uaRoot with no children) still yields only public children.
func TestLayoutChildren_EmptyUA(t *testing.T) {
	doc := NewDocument()
	host := doc.CreateElement("input", nil)
	pub := doc.CreateElement("label", nil)
	uaBox := doc.CreateElement("div", nil) // empty UA subtree
	host.AppendChild(pub)
	host.AttachUARoot(uaBox)

	got := collectLayoutChildren(host)
	want := []dom.Node{pub}
	if !slices.Equal(got, want) {
		t.Errorf("LayoutChildren (empty UA subtree) = %v, want %v", got, want)
	}
}
