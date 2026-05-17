package dom_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// fakeRO is a test-only render.Object that counts MarkChildrenDirty calls.
type fakeRO struct {
	calls int
}

func (f *fakeRO) MarkChildrenDirty()             { f.calls++ }
func (f *fakeRO) EventTarget() event.EventTarget { return nil }
func (f *fakeRO) Parent() render.Object          { return nil }
func (f *fakeRO) FirstChild() render.Object      { return nil }
func (f *fakeRO) LastChild() render.Object       { return nil }
func (f *fakeRO) NextSibling() render.Object     { return nil }
func (f *fakeRO) PreviousSibling() render.Object { return nil }
func (f *fakeRO) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {}
}
func (f *fakeRO) Focusable() bool                  { return false }
func (f *fakeRO) Disabled() bool                   { return false }
func (f *fakeRO) Style() *style.Computed           { return nil }
func (f *fakeRO) ComputedStyle() *style.Computed   { return nil }
func (f *fakeRO) SetComputedStyle(*style.Computed) {}
func (f *fakeRO) Flags() render.DirtyFlag          { return 0 }
func (f *fakeRO) MarkDirty(render.DirtyFlag)       {}
func (f *fakeRO) ClearDirty(render.DirtyFlag)      {}
func (f *fakeRO) IsDetached() bool                 { return false }

// StyleNode implementation
func (f *fakeRO) RawStyle() style.Style             { return style.Style{} }
func (f *fakeRO) ElementDefaultStyle() style.Style  { return style.Style{} }
func (f *fakeRO) IsDirtyStyle() bool                { return false }
func (f *fakeRO) HasDirtyStyleChild() bool          { return false }
func (f *fakeRO) ClearDirtyStyle()                  {}
func (f *fakeRO) ClearChildNeedsStyle()             {}
func (f *fakeRO) StyleParent() style.StyleNode      { return nil }
func (f *fakeRO) StyleFirstChild() style.StyleNode  { return nil }
func (f *fakeRO) StyleNextSibling() style.StyleNode { return nil }

// layout.Node implementation
func (f *fakeRO) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {}
}
func (f *fakeRO) IsDirtyLayout() bool { return false }
func (f *fakeRO) ClearDirtyLayout()   {}
func (f *fakeRO) Fragment() *layout.Fragment { return nil }
func (f *fakeRO) CachedLayout(layout.ConstraintSpace) *layout.Fragment { return nil }
func (f *fakeRO) SetCachedLayout(layout.ConstraintSpace, *layout.Fragment) {}
func (f *fakeRO) LogicalNode() any    { return nil }

var _ render.Object = (*fakeRO)(nil)

// requireNode fails the test if got != want.
func requireNode(t *testing.T, label string, got, want dom.Node) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

func TestElement_AppendChild_LinksSiblings(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div")
	a := doc.CreateElement("a")
	b := doc.CreateElement("b")
	c := doc.CreateElement("c")

	parent.AppendChild(a)
	parent.AppendChild(b)
	parent.AppendChild(c)

	requireNode(t, "FirstChild", parent.FirstChild(), a)
	requireNode(t, "LastChild", parent.LastChild(), c)

	// forward chain: a → b → c
	requireNode(t, "a.NextSibling", a.NextSibling(), b)
	requireNode(t, "b.NextSibling", b.NextSibling(), c)
	requireNode(t, "c.NextSibling", c.NextSibling(), nil)

	// backward chain: c → b → a
	requireNode(t, "c.PreviousSibling", c.PreviousSibling(), b)
	requireNode(t, "b.PreviousSibling", b.PreviousSibling(), a)
	requireNode(t, "a.PreviousSibling", a.PreviousSibling(), nil)

	// parent links
	if a.Parent() != parent {
		t.Error("a.Parent should be parent")
	}
	if b.Parent() != parent {
		t.Error("b.Parent should be parent")
	}
	if c.Parent() != parent {
		t.Error("c.Parent should be parent")
	}
}

func TestElement_InsertBefore_HeadAndMiddle(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("InsertAtHead", func(t *testing.T) {
		parent := doc.CreateElement("div")
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		x := doc.CreateElement("x")

		parent.AppendChild(a)
		parent.AppendChild(b)
		parent.InsertBefore(x, a) // x becomes first child

		requireNode(t, "FirstChild", parent.FirstChild(), x)
		requireNode(t, "x.NextSibling", x.NextSibling(), a)
		requireNode(t, "a.PreviousSibling", a.PreviousSibling(), x)
		requireNode(t, "a.NextSibling", a.NextSibling(), b)
		requireNode(t, "LastChild", parent.LastChild(), b)
	})

	t.Run("InsertInMiddle", func(t *testing.T) {
		parent := doc.CreateElement("div")
		a := doc.CreateElement("a")
		b := doc.CreateElement("b")
		c := doc.CreateElement("c")
		x := doc.CreateElement("x")

		parent.AppendChild(a)
		parent.AppendChild(b)
		parent.AppendChild(c)
		parent.InsertBefore(x, b) // a, x, b, c

		requireNode(t, "FirstChild", parent.FirstChild(), a)
		requireNode(t, "a.NextSibling", a.NextSibling(), x)
		requireNode(t, "x.PreviousSibling", x.PreviousSibling(), a)
		requireNode(t, "x.NextSibling", x.NextSibling(), b)
		requireNode(t, "b.PreviousSibling", b.PreviousSibling(), x)
		requireNode(t, "b.NextSibling", b.NextSibling(), c)
		requireNode(t, "LastChild", parent.LastChild(), c)
	})
}

func TestElement_RemoveChild_Unlinks(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div")
	a := doc.CreateElement("a")
	b := doc.CreateElement("b")
	c := doc.CreateElement("c")

	parent.AppendChild(a)
	parent.AppendChild(b)
	parent.AppendChild(c)

	removed := parent.RemoveChild(b)

	if removed != b {
		t.Error("RemoveChild should return the removed node")
	}

	requireNode(t, "FirstChild", parent.FirstChild(), a)
	requireNode(t, "LastChild", parent.LastChild(), c)
	requireNode(t, "a.NextSibling", a.NextSibling(), c)
	requireNode(t, "c.PreviousSibling", c.PreviousSibling(), a)

	// b should be fully unlinked
	if b.Parent() != nil {
		t.Error("removed node's Parent should be nil")
	}
	requireNode(t, "b.NextSibling", b.NextSibling(), nil)
	requireNode(t, "b.PreviousSibling", b.PreviousSibling(), nil)
}

func TestElement_ReplaceChild_PreservesSiblings(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div")
	a := doc.CreateElement("a")
	b := doc.CreateElement("b")
	c := doc.CreateElement("c")
	x := doc.CreateElement("x") // replaces b

	parent.AppendChild(a)
	parent.AppendChild(b)
	parent.AppendChild(c)

	removed := parent.ReplaceChild(x, b)

	if removed != b {
		t.Error("ReplaceChild should return the replaced node")
	}

	// tree should be: a <-> x <-> c
	requireNode(t, "FirstChild", parent.FirstChild(), a)
	requireNode(t, "LastChild", parent.LastChild(), c)
	requireNode(t, "a.NextSibling", a.NextSibling(), x)
	requireNode(t, "x.PreviousSibling", x.PreviousSibling(), a)
	requireNode(t, "x.NextSibling", x.NextSibling(), c)
	requireNode(t, "c.PreviousSibling", c.PreviousSibling(), x)
	if x.Parent() != parent {
		t.Error("x.Parent should be parent")
	}

	// b should be unlinked
	if b.Parent() != nil {
		t.Error("replaced node's Parent should be nil")
	}
	requireNode(t, "b.NextSibling", b.NextSibling(), nil)
	requireNode(t, "b.PreviousSibling", b.PreviousSibling(), nil)
}

func TestTextNode_SetData_NotifiesParent(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div")
	text := doc.CreateTextNode("hello")

	ro := &fakeRO{}
	parent.SetRenderObject(ro)

	parent.AppendChild(text)
	ro.calls = 0 // reset counter after append

	text.SetData("world")

	if ro.calls != 1 {
		t.Errorf("MarkChildrenDirty call count = %d, want 1", ro.calls)
	}
	if text.Data() != "world" {
		t.Error("Data() should return the updated value")
	}
}

func TestDocument_CreateElement_AssignsTagName(t *testing.T) {
	doc := dom.NewDocument()
	el := doc.CreateElement("section")

	if el.TagName() != "section" {
		t.Errorf("TagName = %q, want %q", el.TagName(), "section")
	}
	if el.OwnerDocument() != doc {
		t.Error("OwnerDocument should be the creating document")
	}
	if el.Parent() != nil {
		t.Error("newly created element should have no parent")
	}
}

// --- ID registry -----------------------------------------------------------

func TestDocument_GetElementByID_ReturnsElement(t *testing.T) {
	doc := dom.NewDocument()
	el := doc.CreateElement("div")
	el.SetID("hero")
	doc.AppendChild(el)

	got := doc.GetElementByID("hero")
	if got != el {
		t.Errorf("GetElementByID(%q) = %v, want %v", "hero", got, el)
	}
}

func TestDocument_GetElementByID_UpdatesOnSetID(t *testing.T) {
	doc := dom.NewDocument()
	el := doc.CreateElement("div")
	el.SetID("old")
	doc.AppendChild(el)

	el.SetID("new")

	if got := doc.GetElementByID("old"); got != nil {
		t.Error("GetElementByID(\"old\") should be nil after ID was changed")
	}
	if got := doc.GetElementByID("new"); got != el {
		t.Errorf("GetElementByID(%q) = %v, want %v", "new", got, el)
	}
}

func TestDocument_GetElementByID_RemovesOnDetach(t *testing.T) {
	doc := dom.NewDocument()
	el := doc.CreateElement("span")
	el.SetID("target")
	doc.AppendChild(el)

	doc.RemoveChild(el)

	if got := doc.GetElementByID("target"); got != nil {
		t.Error("GetElementByID should return nil after element is removed from tree")
	}
}

// --- Anchor registry -------------------------------------------------------

func TestDocument_FindAnchor_ScopedToAnchorRegistry(t *testing.T) {
	doc := dom.NewDocument()
	anchor := doc.CreateElement("a")

	// ID registry and anchor registry are independent.
	anchor.SetID("section-nav")
	doc.AppendChild(anchor)

	// Registering under the same string in the anchor registry should not
	// shadow the element in the ID registry, and vice-versa.
	doc.RegisterAnchor("section-nav", anchor)

	if got := doc.GetElementByID("section-nav"); got != anchor {
		t.Error("GetElementByID must still return the element even when an anchor shares the name")
	}
	if got := doc.FindAnchor("section-nav"); got != anchor {
		t.Errorf("FindAnchor(%q) = %v, want %v", "section-nav", got, anchor)
	}

	// Unregistering the anchor must not affect the ID registry.
	doc.UnregisterAnchor("section-nav")

	if got := doc.FindAnchor("section-nav"); got != nil {
		t.Error("FindAnchor should return nil after UnregisterAnchor")
	}
	if got := doc.GetElementByID("section-nav"); got != anchor {
		t.Error("GetElementByID must still work after anchor is unregistered")
	}
}
