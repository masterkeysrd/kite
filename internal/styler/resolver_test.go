package styler_test

import (
	"image/color"
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// fakeNode — minimal implementation for tests
// ---------------------------------------------------------------------------

type fakeNode struct {
	rawStyle            style.Style
	elementDefaultStyle style.Style // optional per-element-type default
	intrinsicStyle      style.Style // optional UA-forced intrinsic style
	kind                dom.Kind
	doc                 dom.Document
}

func (n *fakeNode) Kind() dom.Kind                           { return n.kind }
func (n *fakeNode) NodeName() string                         { return "fake" }
func (n *fakeNode) Parent() dom.Node                         { return nil }
func (n *fakeNode) ParentElement() dom.Element               { return nil }
func (n *fakeNode) NextSibling() dom.Node                    { return nil }
func (n *fakeNode) PreviousSibling() dom.Node                { return nil }
func (n *fakeNode) OwnerDocument() dom.Document              { return n.doc }
func (n *fakeNode) IsConnected() bool                        { return true }
func (n *fakeNode) AppendChild(dom.Node) dom.Node            { return nil }
func (n *fakeNode) InsertBefore(dom.Node, dom.Node) dom.Node { return nil }
func (n *fakeNode) RemoveChild(dom.Node) dom.Node            { return nil }
func (n *fakeNode) ReplaceChild(dom.Node, dom.Node) dom.Node { return nil }
func (n *fakeNode) FirstChild() dom.Node                     { return nil }
func (n *fakeNode) LastChild() dom.Node                      { return nil }
func (n *fakeNode) HasChildNodes() bool                      { return false }
func (n *fakeNode) Contains(dom.Node) bool                   { return false }
func (n *fakeNode) ChildNodes() iter.Seq[dom.Node]           { return nil }
func (n *fakeNode) Unwrap() dom.Node                         { return nil }
func (n *fakeNode) TextContent() string                      { return "" }
func (n *fakeNode) CloneNode(bool) dom.Node                  { return nil }
func (n *fakeNode) EventTarget() event.EventTarget           { return nil }
func (n *fakeNode) AddEventListener(event.EventType, event.Listener, ...event.Option) event.Subscription {
	return nil
}
func (n *fakeNode) DispatchTo(event.Event)       {}
func (n *fakeNode) DispatchToTarget(event.Event) {}
func (n *fakeNode) RemoveRegistration(uint64)    {}

func (n *fakeNode) TagName() string                          { return "fake" }
func (n *fakeNode) ID() string                               { return "" }
func (n *fakeNode) SetID(string)                             {}
func (n *fakeNode) Class() string                            { return "" }
func (n *fakeNode) SetClass(string)                          {}
func (n *fakeNode) QuerySelector(string) dom.Element         { return nil }
func (n *fakeNode) ReplaceWith(...dom.Node) dom.Element      { return nil }
func (n *fakeNode) AttachUARoot(dom.Node)                    {}
func (n *fakeNode) Scroll() (int, int)                       { return 0, 0 }
func (n *fakeNode) ScrollTo(int, int)                        {}
func (n *fakeNode) ScrollBy(int, int)                        {}
func (n *fakeNode) ScrollCursorIntoView()                    {}
func (n *fakeNode) ProvidesCursor() bool                     { return false }
func (n *fakeNode) GetBoundingClientRect() (geom.Rect, bool) { return geom.Rect{}, false }
func (n *fakeNode) TabIndex() int                            { return -1 }
func (n *fakeNode) SetTabIndex(int)                          {}
func (n *fakeNode) Focus()                                   {}
func (n *fakeNode) Blur()                                    {}
func (n *fakeNode) IsFocusable() bool                        { return false }

func (n *fakeNode) RawStyle() style.Style       { return n.rawStyle }
func (n *fakeNode) DefaultStyle() style.Style   { return n.elementDefaultStyle }
func (n *fakeNode) IntrinsicStyle() style.Style { return n.intrinsicStyle }

// ---------------------------------------------------------------------------
// TestResolver_DefaultsApplied
// ---------------------------------------------------------------------------

func TestResolver_DefaultsApplied(t *testing.T) {
	r := styler.NewResolver()
	n := &fakeNode{kind: dom.KindElement}
	ro := render.NewBox(n, nil)
	ro.MarkDirty(render.DirtyStyle)

	got := r.Resolve(ro, nil)
	want := style.DefaultStyle()

	if got.Display != want.Display {
		t.Errorf("Display = %v, want %v", got.Display, want.Display)
	}
	if got.Foreground != want.Foreground {
		t.Errorf("Foreground = %v, want %v", got.Foreground, want.Foreground)
	}
}

func TestResolver_InheritsColor(t *testing.T) {
	r := styler.NewResolver()

	parentFG := color.RGBA{R: 100, G: 200, B: 50, A: 255}
	parentComputed := &style.Computed{
		Foreground: parentFG,
	}

	childNode := &fakeNode{kind: dom.KindElement}
	childRO := render.NewBox(childNode, nil)
	childRO.MarkDirty(render.DirtyStyle)

	got := r.Resolve(childRO, parentComputed)

	if got.Foreground != parentFG {
		t.Errorf("child: Foreground = %v, want inherited %v", got.Foreground, parentFG)
	}
}

func TestResolver_InheritsTextAlign(t *testing.T) {
	r := styler.NewResolver()

	parentComputed := &style.Computed{
		TextAlign: style.TextAlignCenter,
	}

	childNode := &fakeNode{kind: dom.KindElement}
	childRO := render.NewBox(childNode, nil)
	childRO.MarkDirty(render.DirtyStyle)

	got := r.Resolve(childRO, parentComputed)

	if got.TextAlign != style.TextAlignCenter {
		t.Errorf("child: TextAlign = %v, want inherited TextAlignCenter", got.TextAlign)
	}
}

func TestResolver_FullTreeWalk(t *testing.T) {
	r := styler.NewResolver()

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	pNode := &fakeNode{
		kind:     dom.KindElement,
		rawStyle: style.S().Foreground(red),
	}
	parent := render.NewBlock(pNode, nil)

	cNode := &fakeNode{kind: dom.KindElement}
	child := render.NewBlock(cNode, nil)
	parent.InsertChild(child, nil)

	// Resolve the tree.
	r.ResolveTree(parent, nil, false)

	if child.ComputedStyle().Foreground != red {
		t.Errorf("child should have inherited red, got %v", child.ComputedStyle().Foreground)
	}
}

func TestResolver_DynamicStyleUpdate(t *testing.T) {
	r := styler.NewResolver()

	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	pNode := &fakeNode{
		kind:     dom.KindElement,
		rawStyle: style.S().Foreground(blue),
	}
	parent := render.NewBlock(pNode, nil)

	cNode := &fakeNode{kind: dom.KindElement}
	child := render.NewBlock(cNode, nil)
	parent.InsertChild(child, nil)

	// Frame 1
	r.ResolveTree(parent, nil, false)
	if parent.ComputedStyle().Foreground != blue {
		t.Fatalf("expected blue parent initially, got %v", parent.ComputedStyle().Foreground)
	}
	if child.ComputedStyle().Foreground != blue {
		t.Fatalf("expected blue child initially, got %v", child.ComputedStyle().Foreground)
	}

	// Frame 2: Update parent style to red
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	pNode.rawStyle = style.S().Foreground(red)

	// Simulating propagateStyleDirty marking it dirty
	parent.MarkDirty(render.DirtyStyle)

	r.ResolveTree(parent, nil, false)
	if parent.ComputedStyle().Foreground != red {
		t.Errorf("expected red parent after update, got %v", parent.ComputedStyle().Foreground)
	}
	if child.ComputedStyle().Foreground != red {
		t.Errorf("expected red child after inheritance update, got %v", child.ComputedStyle().Foreground)
	}
}

func TestResolver_ChildStyleUpdate(t *testing.T) {
	r := styler.NewResolver()

	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	pNode := &fakeNode{
		kind:     dom.KindElement,
		rawStyle: style.S().Foreground(blue),
	}
	parent := render.NewBlock(pNode, nil)

	cNode := &fakeNode{
		kind:     dom.KindElement,
		rawStyle: style.S(),
	}
	child := render.NewBlock(cNode, nil)
	parent.InsertChild(child, nil)

	// Frame 1
	r.ResolveTree(parent, nil, false)
	if child.ComputedStyle().Foreground != blue {
		t.Fatalf("expected blue child initially, got %v", child.ComputedStyle().Foreground)
	}

	// Frame 2: Update child style to green (override parent)
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	cNode.rawStyle = style.S().Foreground(green)

	// Simulating propagateStyleDirty marking child dirty.
	// Since child is dirty style, it propagates ChildNeedsStyle to parent.
	child.MarkDirty(render.DirtyStyle)

	r.ResolveTree(parent, nil, false)
	if child.ComputedStyle().Foreground != green {
		t.Errorf("expected green child after update, got %v", child.ComputedStyle().Foreground)
	}
}

type fakeDocument struct {
	dom.Document
	view dom.View
}

func (d *fakeDocument) DefaultView() dom.View { return d.view }

type fakeView struct {
	dom.View
	size geom.Size
}

func (v *fakeView) ViewportSize() geom.Size { return v.size }

func TestResolver_MediaQueries(t *testing.T) {
	r := styler.NewResolver()

	view := &fakeView{size: geom.Size{Width: 40, Height: 20}}
	doc := &fakeDocument{view: view}

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	s := style.S().
		Foreground(red).
		Media(style.Query().MinWidth(80), style.S().Foreground(blue))

	node := &fakeNode{
		kind:     dom.KindElement,
		rawStyle: s,
		doc:      doc,
	}
	ro := render.NewBlock(node, nil)

	// Resolve style when width = 40 (should remain red)
	r.ResolveTree(ro, nil, false)
	if ro.ComputedStyle().Foreground != red {
		t.Errorf("expected red foreground, got %v", ro.ComputedStyle().Foreground)
	}

	// Change viewport width to 100 (matches query)
	view.size = geom.Size{Width: 100, Height: 20}
	ro.MarkDirty(render.DirtyStyle)
	r.Invalidate(node) // clear cache

	// Re-resolve
	r.ResolveTree(ro, nil, false)
	if ro.ComputedStyle().Foreground != blue {
		t.Errorf("expected blue foreground after matching media query, got %v", ro.ComputedStyle().Foreground)
	}
}
