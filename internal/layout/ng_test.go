package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

type mockCursorNode struct {
	mockNode
}

func (m *mockCursorNode) ProvidesCursor() bool                         { return true }
func (m *mockCursorNode) LogicalNode() dom.Node                        { return m }
func (m *mockCursorNode) TagName() string                              { return "input" }
func (m *mockCursorNode) ID() string                                   { return "" }
func (m *mockCursorNode) SetID(string)                                 {}
func (m *mockCursorNode) Class() string                                { return "" }
func (m *mockCursorNode) SetClass(string)                              {}
func (m *mockCursorNode) QuerySelector(string) dom.Element             { return nil }
func (m *mockCursorNode) ReplaceWith(...dom.Node) dom.Element          { return m }
func (m *mockCursorNode) RawStyle() style.Style                        { return style.S() }
func (m *mockCursorNode) DefaultStyle() style.Style                    { return style.S() }
func (m *mockCursorNode) IntrinsicStyle() style.Style                  { return style.S() }
func (m *mockCursorNode) AttachUARoot(dom.Node)                        {}
func (m *mockCursorNode) Scroll() (int, int)                           { return 0, 0 }
func (m *mockCursorNode) ScrollTo(int, int)                            {}
func (m *mockCursorNode) ScrollBy(int, int)                            {}
func (m *mockCursorNode) ScrollCursorIntoView()                        {}
func (m *mockCursorNode) GetBoundingClientRect() (geometry.Rect, bool) { return geometry.Rect{}, false }
func (m *mockCursorNode) TabIndex() int                                { return 0 }
func (m *mockCursorNode) SetTabIndex(int)                              {}
func (m *mockCursorNode) Focus()                                       {}
func (m *mockCursorNode) Blur()                                        {}
func (m *mockCursorNode) IsFocusable() bool                            { return true }

var _ dom.Element = (*mockCursorNode)(nil)

func TestMaxScroll_CursorProvider(t *testing.T) {
	// Viewport 10x1.
	// Content fits exactly (10x1).
	// Cursor provider should NOT get the extra 1-cell scroll if it fits exactly.
	// (Actually, the logic says 'extentW >= viewport.Width', so it WILL get it if it fits exactly).
	// Let's re-verify the intended behavior.
	// If content is 10 and viewport is 10, maxSX should be 1 so the cursor can sit at index 10.

	node := &mockCursorNode{
		mockNode: mockNode{
			style: &style.Computed{
				Width:  style.Cells(10),
				Height: style.Cells(1),
			},
		},
	}

	// Create a text fragment of width 10
	textFrag := &Fragment{
		Size: geometry.Size{Width: 10, Height: 1},
		Node: node,
	}

	frag := &Fragment{
		Size: geometry.Size{Width: 10, Height: 1},
		Node: node,
		Children: []FragmentLink{
			{Offset: geometry.Point{X: 0, Y: 0}, Fragment: textFrag},
		},
	}

	maxSX, _ := MaxScroll(frag)
	if maxSX != 1 {
		t.Errorf("expected maxSX 1 for cursor provider fitting exactly, got %d", maxSX)
	}

	// Content is smaller (5x1).
	// Should NOT get extra scroll.
	textFragSmall := &Fragment{
		Size: geometry.Size{Width: 5, Height: 1},
		Node: node,
	}
	fragSmall := &Fragment{
		Size: geometry.Size{Width: 10, Height: 1},
		Node: node,
		Children: []FragmentLink{
			{Offset: geometry.Point{X: 0, Y: 0}, Fragment: textFragSmall},
		},
	}

	maxSXSmall, _ := MaxScroll(fragSmall)
	if maxSXSmall != 0 {
		t.Errorf("expected maxSX 0 for cursor provider smaller than viewport, got %d", maxSXSmall)
	}
}

func TestResolveDecorations(t *testing.T) {
	node := &mockNode{
		style: &style.Computed{
			Border:  style.SingleBorder(),
			Padding: style.Edges(1),
		},
	}

	// 1. No scrollbars
	decor := ResolveDecorations(node, false, false)
	// Insets = Border(1) + Padding(1) = 2 all sides
	expected := style.EdgeValues[int]{Top: 2, Right: 2, Bottom: 2, Left: 2}
	if decor.Insets != expected {
		t.Errorf("expected insets %v, got %v", expected, decor.Insets)
	}

	// 2. Vertical scrollbar
	decorY := ResolveDecorations(node, false, true)
	// Insets.Right should increment by 1
	if decorY.Insets.Right != 3 {
		t.Errorf("expected right inset 3 with scrollbarY, got %d", decorY.Insets.Right)
	}

	// 3. Horizontal scrollbar
	decorX := ResolveDecorations(node, true, false)
	// Insets.Bottom should increment by 1
	if decorX.Insets.Bottom != 3 {
		t.Errorf("expected bottom inset 3 with scrollbarX, got %d", decorX.Insets.Bottom)
	}

	// 4. Both scrollbars
	decorXY := ResolveDecorations(node, true, true)
	if decorXY.Insets.Right != 3 || decorXY.Insets.Bottom != 3 {
		t.Errorf("expected insets {R:3, B:3} with both scrollbars, got %v", decorXY.Insets)
	}

	// Viewport size check
	outer := geometry.Size{Width: 20, Height: 10}
	// decorXY insets: T:2, R:3, B:3, L:2
	// width = 20 - 2 - 3 = 15
	// height = 10 - 2 - 3 = 5
	vp := decorXY.ViewportSize(outer)
	if vp.Width != 15 || vp.Height != 5 {
		t.Errorf("expected viewport 15x5, got %dx%d", vp.Width, vp.Height)
	}
}
