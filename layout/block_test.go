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
	cachedMinMax   MinMaxSizes
	minMaxValid    bool
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
		width := 0
		switch m.style.Width.Kind() {
		case style.KindCells:
			width = m.style.Width.CellsValue()
		case style.KindPercent:
			width = int(float32(space.PercentageResolutionSize.Width) * m.style.Width.PercentValue() / 100.0)
		case style.KindAuto:
			if space.IsFixedInlineSize {
				width = space.AvailableSize.Width
			}
		}

		height := 0
		switch m.style.Height.Kind() {
		case style.KindCells:
			height = m.style.Height.CellsValue()
		case style.KindPercent:
			height = int(float32(space.PercentageResolutionSize.Height) * m.style.Height.PercentValue() / 100.0)
		case style.KindAuto:
			if space.IsFixedBlockSize {
				height = space.AvailableSize.Height
			}
		}

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

func (m *mockNode) CachedMinMaxSizes() (MinMaxSizes, bool) {
	if m.dirty {
		return MinMaxSizes{}, false
	}
	return m.cachedMinMax, m.minMaxValid
}

func (m *mockNode) SetCachedMinMaxSizes(sizes MinMaxSizes) {
	m.cachedMinMax = sizes
	m.minMaxValid = true
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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Parent should be 100x6 (stretched to available width)
	if frag.Size.Width != 100 {
		t.Errorf("expected width 100, got %d", frag.Size.Width)
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
			Width:   style.Auto,
			Height:  style.Auto,
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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Total width: stretches to available width (100)
	if frag.Size.Width != 100 {
		t.Errorf("expected width 100, got %d", frag.Size.Width)
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

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()

	algo := &BlockAlgorithm{Node: parent, Space: space}
	frag := algo.Layout()

	// Parent width: stretches to available width (100)
	if frag.Size.Width != 100 {
		t.Errorf("expected width 100, got %d", frag.Size.Width)
	}
	if frag.Size.Height != 5 {
		t.Errorf("expected height 5, got %d", frag.Size.Height)
	}

	// Child position should be (3, 1)
	if frag.Children[0].Offset != (Point{3, 1}) {
		t.Errorf("expected child offset {3, 1}, got %v", frag.Children[0].Offset)
	}
}

func TestBlockLayout_IntrinsicSizeCaching(t *testing.T) {
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(10),
			Height: style.Cells(2),
		},
	}
	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Auto,
		},
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()
	algo := &BlockAlgorithm{Node: parent, Space: space}

	// First layout should compute intrinsic size.
	algo.Layout()
	if !parent.minMaxValid {
		t.Error("expected intrinsic sizes to be cached after first layout")
	}
	cachedSizes := parent.cachedMinMax

	// Second layout with different available size but same node (not dirty)
	// should reuse cached intrinsic size.
	space2 := NewConstraintSpaceBuilder(Size{200, 100}).ToConstraintSpace()
	algo2 := &BlockAlgorithm{Node: parent, Space: space2}

	algo2.Layout()
	if !parent.minMaxValid {
		t.Error("expected intrinsic sizes to remain valid")
	}
	if parent.cachedMinMax != cachedSizes {
		t.Error("intrinsic sizes should not have changed")
	}
}

func BenchmarkBlockLayout_DeepTree(b *testing.B) {
	childStyle := &style.Computed{
		Width:  style.Cells(10),
		Height: style.Cells(1),
	}

	// Create a tree of depth 100
	var root *mockNode
	current := &mockNode{style: childStyle, dirty: true}
	root = current
	for range 100 {
		next := &mockNode{style: childStyle, dirty: true}
		current.firstChild = next
		current = next
	}

	space := NewConstraintSpaceBuilder(Size{100, 1000}).ToConstraintSpace()

	for b.Loop() {
		// Mark everything dirty to force full re-layout
		curr := root
		for curr != nil {
			curr.dirty = true
			if curr.firstChild != nil {
				curr = curr.firstChild.(*mockNode)
			} else {
				curr = nil
			}
		}

		algo := &BlockAlgorithm{Node: root, Space: space}
		algo.Layout()
	}
}

func TestBlockLayout_PercentageResolution(t *testing.T) {
	// Root (80x24) -> Child (50%) -> Grandchild (50%)
	grandchild := &mockNode{
		style: &style.Computed{
			Width:  style.Percent(50),
			Height: style.Cells(2),
		},
	}
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Percent(50),
			Height: style.Auto,
		},
		firstChild: grandchild,
	}
	root := &mockNode{
		style: &style.Computed{
			Width:  style.Percent(100),
			Height: style.Percent(100),
		},
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(Size{80, 24}).
		SetIsFixedInlineSize(true).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	algo := &BlockAlgorithm{Node: root, Space: space}
	frag := algo.Layout()

	// Root should be 80x24
	if frag.Size.Width != 80 || frag.Size.Height != 24 {
		t.Errorf("expected root 80x24, got %dx%d", frag.Size.Width, frag.Size.Height)
	}

	// Child should be 50% of 80 = 40
	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 40 {
		t.Errorf("expected child width 40, got %d", childFrag.Size.Width)
	}

	// Grandchild should be 50% of 40 = 20
	grandchildFrag := childFrag.Children[0].Fragment
	if grandchildFrag.Size.Width != 20 {
		t.Errorf("expected grandchild width 20, got %d", grandchildFrag.Size.Width)
	}
}

func TestBlockLayout_AutoStretch(t *testing.T) {
	// Root (80x24) -> Child (Width: Auto)
	// Even if child has no content, it should stretch to 80
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Auto,
			Height: style.Cells(1),
		},
	}
	root := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(80),
			Height: style.Cells(24),
		},
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(Size{80, 24}).
		SetIsFixedInlineSize(true).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	algo := &BlockAlgorithm{Node: root, Space: space}
	frag := algo.Layout()

	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 80 {
		t.Errorf("expected auto child to stretch to 80, got %d", childFrag.Size.Width)
	}
}

func TestBlockLayout_FixedBlockSizePercentage(t *testing.T) {
	// If parent has fixed block size, children should resolve percentage height
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(10),
			Height: style.Percent(50),
		},
	}
	root := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(80),
			Height: style.Cells(20),
		},
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(Size{80, 20}).
		SetIsFixedInlineSize(true).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	algo := &BlockAlgorithm{Node: root, Space: space}
	frag := algo.Layout()

	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Height != 10 {
		t.Errorf("expected child height 10 (50%% of 20), got %d", childFrag.Size.Height)
	}
}
