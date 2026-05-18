package layout

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

type mockNode struct {
	style          *style.Computed
	firstChild     Node
	nextSibling    Node
	dirty          bool
	cachedSpace    ConstraintSpace
	cachedFragment *Fragment
}

func (m *mockNode) Style() *style.Computed { return m.style }
func (m *mockNode) LayoutChildren() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for n := m.firstChild; n != nil; {
			next := n.(*mockNode).NextSibling()
			if !yield(n) {
				return
			}
			n = next
		}
	}
}
func (m *mockNode) LogicalNode() any    { return nil }
func (m *mockNode) IsDirtyLayout() bool { return m.dirty }
func (m *mockNode) ClearDirtyLayout()   { m.dirty = false }
func (m *mockNode) Fragment() *Fragment { return m.cachedFragment }
func (m *mockNode) CachedLayout(space ConstraintSpace) *Fragment {
	if m.cachedFragment != nil {
		return m.cachedFragment
	}
	// For leaf nodes, return a fragment based on Style if requested
	if m.firstChild == nil {
		width := m.style.Width.CellsValue()
		height := m.style.Height.CellsValue()
		return &Fragment{
			Size: Size{Width: width, Height: height},
			Node: m,
		}
	}
	return nil
}
func (m *mockNode) SetCachedLayout(space ConstraintSpace, frag *Fragment) {
	m.cachedSpace = space
	m.cachedFragment = frag
	m.dirty = false
}
func (m *mockNode) NextSibling() Node { return m.nextSibling }

func TestBlockLayout_VerticalStacking(t *testing.T) {
	// Create a parent with 3 children, each 10x2
	childStyle := &style.Computed{
		Width:  style.Cells(10),
		Height: style.Cells(2),
	}
	c3 := &mockNode{style: childStyle}
	c2 := &mockNode{style: childStyle, nextSibling: c3}
	c1 := &mockNode{style: childStyle, nextSibling: c2}

	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Auto,
		},
		firstChild: c1,
	}

	space := ConstraintSpace{
		Constraints: Constraints{
			Min: Size{0, 0},
			Max: Size{100, 100},
		},
	}

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Parent should be 10x6
	if frag.Size.Width != 10 {
		t.Errorf("expected width 10, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 6 {
		t.Errorf("expected height 6, got %d", frag.Size.Height)
	}

	// Check child positions
	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(frag.Children))
	}

	expectedOffsets := []Point{
		{0, 0},
		{0, 2},
		{0, 4},
	}

	for i, child := range frag.Children {
		if child.Offset != expectedOffsets[i] {
			t.Errorf("child %d: expected offset %v, got %v", i, expectedOffsets[i], child.Offset)
		}
	}
}

func TestBlockLayout_PaddingAndBorder(t *testing.T) {
	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Auto,
			Padding: style.EdgeValues[int]{Top: 1, Bottom: 1, Left: 2, Right: 2},
			Border: style.Border{
				Width: style.EdgeValues[int]{Top: 1, Bottom: 1, Left: 1, Right: 1},
			},
		},
		firstChild: &mockNode{
			style: &style.Computed{
				Width:  style.Cells(10),
				Height: style.Cells(2),
			},
		},
	}

	space := ConstraintSpace{
		Constraints: Constraints{
			Min: Size{0, 0},
			Max: Size{100, 100},
		},
	}

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Total width: 1 (border-left) + 2 (padding-left) + 10 (child) + 2 (padding-right) + 1 (border-right) = 16
	// Total height: 1 (border-top) + 1 (padding-top) + 2 (child) + 1 (padding-bottom) + 1 (border-bottom) = 6
	if frag.Size.Width != 16 {
		t.Errorf("expected width 16, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 6 {
		t.Errorf("expected height 6, got %d", frag.Size.Height)
	}

	// Child position should be (3, 2)
	if frag.Children[0].Offset != (Point{3, 2}) {
		t.Errorf("expected child offset {3, 2}, got %v", frag.Children[0].Offset)
	}
}

func TestBlockLayout_Margins(t *testing.T) {
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(10),
			Height: style.Cells(2),
			Margin: style.EdgeValues[int]{Top: 1, Bottom: 2, Left: 3, Right: 4},
		},
	}
	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Auto,
		},
		firstChild: child,
	}

	space := ConstraintSpace{
		Constraints: Constraints{
			Min: Size{0, 0},
			Max: Size{100, 100},
		},
	}

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Parent width: 3 (margin-left) + 10 (child width) + 4 (margin-right) = 17
	// Parent height: 1 (margin-top) + 2 (child height) + 2 (margin-bottom) = 5
	if frag.Size.Width != 17 {
		t.Errorf("expected width 17, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 5 {
		t.Errorf("expected height 5, got %d", frag.Size.Height)
	}

	// Child position should be (3, 1)
	if frag.Children[0].Offset != (Point{3, 1}) {
		t.Errorf("expected child offset {3, 1}, got %v", frag.Children[0].Offset)
	}
}

func BenchmarkBlockLayout_100Children(b *testing.B) {
	childStyle := &style.Computed{
		Width:  style.Cells(10),
		Height: style.Cells(1),
	}
	var firstChild, prev *mockNode
	for i := 0; i < 100; i++ {
		curr := &mockNode{style: childStyle, dirty: true}
		if firstChild == nil {
			firstChild = curr
		}
		if prev != nil {
			prev.nextSibling = curr
		}
		prev = curr
	}
	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Auto,
		},
		firstChild: firstChild,
		dirty:      true,
	}

	space := ConstraintSpace{
		Constraints: Constraints{
			Min: Size{0, 0},
			Max: Size{100, 1000},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mark dirty to force re-layout
		parent.dirty = true
		for n := firstChild; n != nil; {
			n.dirty = true
			if next := n.nextSibling; next != nil {
				n = next.(*mockNode)
			} else {
				n = nil
			}
		}
		algo := &BlockAlgorithm{Node: parent, Space: space}
		algo.Layout()
	}
}
