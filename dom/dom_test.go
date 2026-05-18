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
func (f *fakeRO) SetDisabled(bool)                 {}
func (f *fakeRO) SetFocusable(bool)                {}
func (f *fakeRO) Style() *style.Computed           { return nil }
func (f *fakeRO) ComputedStyle() *style.Computed   { return nil }
func (f *fakeRO) SetComputedStyle(*style.Computed) {}
func (f *fakeRO) Flags() render.DirtyFlag          { return 0 }
func (f *fakeRO) MarkDirty(render.DirtyFlag)       {}
func (f *fakeRO) ClearDirty(render.DirtyFlag)      {}
func (f *fakeRO) ClearDirtyRecursive(render.DirtyFlag) {}
func (f *fakeRO) IsDetached() bool                 { return false }
func (f *fakeRO) Node() dom.Node                   { return nil }
func (f *fakeRO) InsertChild(child, before render.Object) {}
func (f *fakeRO) RemoveChild(child render.Object)         {}

// StyleNode implementation
func (f *fakeRO) RawStyle() style.Style             { return style.Style{} }
func (f *fakeRO) SetRawStyle(style.Style)           {}
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
func (f *fakeRO) IsDirtyLayout() bool                                      { return false }
func (f *fakeRO) ClearDirtyLayout()                                        {}
func (f *fakeRO) Fragment() *layout.Fragment                               { return nil }
func (f *fakeRO) CachedLayout(layout.ConstraintSpace) *layout.Fragment     { return nil }
func (f *fakeRO) SetCachedLayout(layout.ConstraintSpace, *layout.Fragment) {}
func (f *fakeRO) CachedMinMaxSizes() (layout.MinMaxSizes, bool)            { return layout.MinMaxSizes{}, false }
func (f *fakeRO) SetCachedMinMaxSizes(layout.MinMaxSizes)                  {}
func (f *fakeRO) LogicalNode() any                                         { return nil }

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
	parent := doc.CreateElement("div", nil)
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	c := doc.CreateElement("c", nil)

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
		parent := doc.CreateElement("div", nil)
		a := doc.CreateElement("a", nil)
		b := doc.CreateElement("b", nil)
		x := doc.CreateElement("x", nil)

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
		parent := doc.CreateElement("div", nil)
		a := doc.CreateElement("a", nil)
		b := doc.CreateElement("b", nil)
		c := doc.CreateElement("c", nil)
		x := doc.CreateElement("x", nil)

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
	parent := doc.CreateElement("div", nil)
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	c := doc.CreateElement("c", nil)

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
	parent := doc.CreateElement("div", nil)
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	c := doc.CreateElement("c", nil)
	x := doc.CreateElement("x", nil) // replaces b

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
}

func TestElement_ChildNodes_Iterator(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	c := doc.CreateElement("c", nil)

	parent.AppendChild(a)
	parent.AppendChild(b)
	parent.AppendChild(c)

	want := []dom.Node{a, b, c}
	got := []dom.Node{}
	for child := range parent.ChildNodes() {
		got = append(got, child)
	}

	if len(got) != len(want) {
		t.Fatalf("ChildNodes: got %d nodes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ChildNodes[%d]: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestElement_TextContent(t *testing.T) {
	doc := dom.NewDocument()
	root := doc.CreateElement("div", nil)
	a := doc.CreateElement("span", nil)
	a.AppendChild(doc.CreateTextNode("Hello ", nil))
	b := doc.CreateElement("span", nil)
	b.AppendChild(doc.CreateTextNode("World!", nil))

	root.AppendChild(a)
	root.AppendChild(b)

	want := "Hello World!"
	if got := root.TextContent(); got != want {
		t.Errorf("TextContent: got %q, want %q", got, want)
	}
}

func TestElement_CloneNode(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	parent.SetID("root")
	child := doc.CreateElement("span", nil)
	child.AppendChild(doc.CreateTextNode("foo", nil))
	parent.AppendChild(child)

	t.Run("Shallow", func(t *testing.T) {
		clone := parent.CloneNode(false).(dom.Element)
		if clone.TagName() != "div" {
			t.Errorf("CloneTagName: got %q, want %q", clone.TagName(), "div")
		}
		if clone.ID() != "root" {
			t.Errorf("CloneID: got %q, want %q", clone.ID(), "root")
		}
		if clone.HasChildNodes() {
			t.Error("Shallow clone should have no children")
		}
	})

	t.Run("Deep", func(t *testing.T) {
		clone := parent.CloneNode(true).(dom.Element)
		if !clone.HasChildNodes() {
			t.Fatal("Deep clone should have children")
		}
		c := clone.FirstChild().(dom.Element)
		if c.TagName() != "span" {
			t.Errorf("ChildTagName: got %q, want %q", c.TagName(), "span")
		}
		if c.TextContent() != "foo" {
			t.Errorf("ChildText: got %q, want %q", c.TextContent(), "foo")
		}
	})
}

func TestElement_MarkChildrenDirty_OnMutation(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	ro := &fakeRO{}
	parent.SetRenderObject(ro)

	child := doc.CreateElement("span", nil)
	parent.AppendChild(child)

	if ro.calls == 0 {
		t.Error("MarkChildrenDirty should be called on AppendChild")
	}

	before := ro.calls
	parent.RemoveChild(child)
	if ro.calls <= before {
		t.Error("MarkChildrenDirty should be called on RemoveChild")
	}
}
