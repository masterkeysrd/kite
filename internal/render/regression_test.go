package render_test

import (
	"image/color"
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

type stubNode struct {
	style style.Style
}

func (n *stubNode) Kind() dom.Kind                           { return dom.KindElement }
func (n *stubNode) NodeName() string                         { return "stub" }
func (n *stubNode) Parent() dom.Node                         { return nil }
func (n *stubNode) ParentElement() dom.Element               { return nil }
func (n *stubNode) NextSibling() dom.Node                    { return nil }
func (n *stubNode) PreviousSibling() dom.Node                { return nil }
func (n *stubNode) OwnerDocument() dom.Document              { return nil }
func (n *stubNode) IsConnected() bool                        { return true }
func (n *stubNode) AppendChild(dom.Node) dom.Node            { return nil }
func (n *stubNode) InsertBefore(dom.Node, dom.Node) dom.Node { return nil }
func (n *stubNode) RemoveChild(dom.Node) dom.Node            { return nil }
func (n *stubNode) ReplaceChild(dom.Node, dom.Node) dom.Node { return nil }
func (n *stubNode) FirstChild() dom.Node                     { return nil }
func (n *stubNode) LastChild() dom.Node                      { return nil }
func (n *stubNode) HasChildNodes() bool                      { return false }
func (n *stubNode) Contains(dom.Node) bool                   { return false }
func (n *stubNode) ChildNodes() iter.Seq[dom.Node]           { return nil }
func (n *stubNode) Unwrap() dom.Node                         { return nil }
func (n *stubNode) TextContent() string                      { return "" }

func (n *stubNode) CloneNode(bool) dom.Node        { return nil }
func (n *stubNode) EventTarget() event.EventTarget { return nil }
func (n *stubNode) AddEventListener(event.EventType, event.Listener, ...event.Option) event.Subscription {
	return nil
}
func (n *stubNode) DispatchTo(event.Event)       {}
func (n *stubNode) DispatchToTarget(event.Event) {}
func (n *stubNode) RemoveRegistration(uint64)    {}

func (n *stubNode) TagName() string                          { return "stub" }
func (n *stubNode) ID() string                               { return "" }
func (n *stubNode) SetID(string)                             {}
func (n *stubNode) Class() string                            { return "" }
func (n *stubNode) SetClass(string)                          {}
func (n *stubNode) Attribute(string) (string, bool)          { return "", false }
func (n *stubNode) SetAttribute(string, string)              {}
func (n *stubNode) RemoveAttribute(string)                   {}
func (n *stubNode) HasAttribute(string) bool                 { return false }
func (n *stubNode) EachAttribute(func(string, string) bool)  {}
func (n *stubNode) QuerySelector(string) dom.Element         { return nil }
func (n *stubNode) ReplaceWith(...dom.Node) dom.Element      { return n }
func (n *stubNode) AttachUARoot(dom.Node)                    {}
func (n *stubNode) Scroll() (int, int)                       { return 0, 0 }
func (n *stubNode) ScrollTo(int, int)                        {}
func (n *stubNode) ScrollBy(int, int)                        {}
func (n *stubNode) ScrollCursorIntoView()                    {}
func (n *stubNode) ProvidesCursor() bool                     { return false }
func (n *stubNode) GetBoundingClientRect() (geom.Rect, bool) { return geom.Rect{}, false }
func (n *stubNode) TabIndex() int                            { return -1 }
func (n *stubNode) SetTabIndex(int)                          {}
func (n *stubNode) Focus()                                   {}
func (n *stubNode) Blur()                                    {}
func (n *stubNode) IsFocusable() bool                        { return false }

func (n *stubNode) RawStyle() style.Style       { return n.style }
func (n *stubNode) DefaultStyle() style.Style   { return style.S() }
func (n *stubNode) IntrinsicStyle() style.Style { return style.S() }

func TestRegression_InheritancePropagation(t *testing.T) {
	view := render.NewRenderView()
	view.SetViewportSize(geom.Size{Width: 80, Height: 24})

	pNode := &stubNode{style: style.S().Foreground(color.White)}
	parent := render.NewBlock(pNode, nil)
	view.InsertChild(parent, nil)

	cNode := &stubNode{style: style.S()}
	child := render.NewBlock(cNode, nil)
	// child does not set foreground, should inherit
	parent.InsertChild(child, nil)

	// Resolve the tree.
	resolver := styler.NewResolver()
	resolver.ResolveTree(view, nil, false)

	if child.ComputedStyle().Foreground != color.White {
		t.Fatalf("Child should inherit white, got %v", child.ComputedStyle().Foreground)
	}

	// Change parent foreground
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	pNode.style = style.S().Foreground(red)
	parent.MarkDirty(render.DirtyStyle)

	// Resolve again
	resolver.ResolveTree(view, nil, false)

	if child.ComputedStyle().Foreground != red {
		t.Fatalf("Child should inherit red, got %v", child.ComputedStyle().Foreground)
	}
}

func TestRegression_FlexLayoutAfterDRY(t *testing.T) {
	view := render.NewRenderView()
	view.SetViewportSize(geom.Size{Width: 80, Height: 24})

	fNode := &stubNode{style: style.S().Display(style.DisplayFlex).Width(style.Percent(100)).Height(style.Percent(100))}
	flex := render.NewBox(fNode, nil)
	view.InsertChild(flex, nil)

	c1Node := &stubNode{style: style.S().Flex(1, 1, style.Auto)}
	child1 := render.NewBlock(c1Node, nil)
	flex.InsertChild(child1, nil)

	c2Node := &stubNode{style: style.S().Flex(1, 1, style.Auto)}
	child2 := render.NewBlock(c2Node, nil)
	flex.InsertChild(child2, nil)

	// Resolve the tree.
	resolver := styler.NewResolver()
	resolver.ResolveTree(view, nil, false)

	viewport := view.ViewportSize()
	render.LayoutPhase(nil, view, viewport)

	if flex.Fragment() == nil {
		t.Fatal("Flex fragment is nil")
	}
	if len(flex.Fragment().Children) != 2 {
		t.Fatalf("Expected 2 children in flex fragment, got %d", len(flex.Fragment().Children))
	}

	// Verify they are positioned side-by-side (default Row)
	c1 := flex.Fragment().Children[0]
	c2 := flex.Fragment().Children[1]

	if c1.Offset.X == c2.Offset.X && c1.Fragment.Size.Width > 0 {
		t.Errorf("Flex children should not overlap horizontally in Row. c1.X=%d, c2.X=%d", c1.Offset.X, c2.Offset.X)
	}
}

func TestRegression_MultipleChildrenBlock(t *testing.T) {
	view := render.NewRenderView()
	view.SetViewportSize(geom.Size{Width: 80, Height: 24})

	bNode := &stubNode{style: style.S().Display(style.DisplayBlock)}
	block := render.NewBlock(bNode, nil)
	view.InsertChild(block, nil)

	c1Node := &stubNode{style: style.S().Height(style.Cells(1))}
	child1 := render.NewBlock(c1Node, nil)
	block.InsertChild(child1, nil)

	c2Node := &stubNode{style: style.S().Height(style.Cells(1))}
	child2 := render.NewBlock(c2Node, nil)
	block.InsertChild(child2, nil)

	resolver := styler.NewResolver()
	resolver.ResolveTree(view, nil, false)

	render.LayoutPhase(nil, view, view.ViewportSize())

	if len(block.Fragment().Children) != 2 {
		t.Fatalf("Expected 2 children in block, got %d", len(block.Fragment().Children))
	}

	c1 := block.Fragment().Children[0]
	c2 := block.Fragment().Children[1]

	if c1.Offset.Y == c2.Offset.Y {
		t.Errorf("Block children should not overlap vertically. c1.Y=%d, c2.Y=%d", c1.Offset.Y, c2.Offset.Y)
	}
}

func TestRegression_ListNoChildrenNoCrash(t *testing.T) {
	node := render.NewBox(nil, nil)
	node.SetComputedStyle(&style.Computed{
		Display:       style.DisplayListItem,
		ListStyleType: style.ListStyleDisc,
	})

	space := layout.NewConstraintSpaceBuilder(geom.Size{Width: 100, Height: 100}).ToConstraintSpace()
	algo := layout.GetAlgorithm(node)

	// Should not crash
	frag := algo.Layout(nil, node, space)

	if frag == nil {
		t.Fatal("Fragment is nil")
	}

	// Should have 1 child (the marker)
	if len(frag.Children) != 1 {
		t.Errorf("Expected 1 child (marker), got %d", len(frag.Children))
	}
}
