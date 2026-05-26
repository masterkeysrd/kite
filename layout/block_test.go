package layout

import (
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
func (m *mockNode) FirstLayoutChild() Node {
	return m.firstChild
}

func (m *mockNode) NextLayoutSibling(child Node) Node {
	type nextSib interface{ NextSibling() Node }
	return child.(nextSib).NextSibling()
}

func (m *mockNode) FirstChild() Node { return m.firstChild }

func (m *mockNode) LogicalNode() any         { return nil }
func (m *mockNode) IsDirtyLayout() bool      { return m.dirty }
func (m *mockNode) IsDirtyPaint() bool       { return m.dirty }
func (m *mockNode) HasChildNeedsPaint() bool { return m.dirty }
func (m *mockNode) ClearDirtyLayout()        { m.dirty = false }
func (m *mockNode) Fragment() *Fragment      { return m.cachedFragment }
func (m *mockNode) CachedLayout(space ConstraintSpace) *Fragment {
	if m.cachedFragment != nil && m.cachedSpace == space {
		return m.cachedFragment
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

func (m *mockNode) SetOffset(Point) {}

func (m *mockNode) IsAnonymous() bool { return false }

func (m *mockNode) NextSibling() Node { return m.nextSibling }

type mockTextNode struct {
	mockNode
	data string
}

func (m *mockTextNode) Data() string     { return m.data }
func (m *mockTextNode) LogicalNode() any { return m }

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

	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)
	parent.SetCachedLayout(space, frag)

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
			Border:  style.SingleBorder(),
		},
		firstChild: &mockNode{
			style: &style.Computed{
				Width:  style.Cells(10),
				Height: style.Cells(2),
			},
		},
	}

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()

	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)
	parent.SetCachedLayout(space, frag)

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

	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)
	parent.SetCachedLayout(space, frag)

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

func TestBlockLayout_FixedHeightRespected(t *testing.T) {
	child := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(10),
			Height: style.Cells(10),
		},
	}
	parent := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(20),
			Height: style.Cells(5),
		},
		firstChild: child,
	}

	space := NewConstraintSpaceBuilder(Size{100, 100}).ToConstraintSpace()

	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)

	// Parent height should be exactly 5, despite child wanting 10
	if frag.Size.Height != 5 {
		t.Errorf("expected height 5, got %d", frag.Size.Height)
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
	algo := GetAlgorithm(parent)

	// First layout should compute intrinsic size.
	algo.Layout(nil, parent, space)
	if !parent.minMaxValid {
		t.Error("expected intrinsic sizes to be cached after first layout")
	}
	cachedSizes := parent.cachedMinMax

	// Second layout with different available size but same node (not dirty)
	// should reuse cached intrinsic size.
	space2 := NewConstraintSpaceBuilder(Size{200, 100}).ToConstraintSpace()
	algo2 := GetAlgorithm(parent)

	algo2.Layout(nil, parent, space2)
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

		algo := GetAlgorithm(root)
		algo.Layout(nil, root, space)
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

	algo := GetAlgorithm(root)
	frag := algo.Layout(nil, root, space)

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

	algo := GetAlgorithm(root)
	frag := algo.Layout(nil, root, space)

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

	algo := GetAlgorithm(root)
	frag := algo.Layout(nil, root, space)

	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Height != 10 {
		t.Errorf("expected child height 10 (50%% of 20), got %d", childFrag.Size.Height)
	}
}

// TestBlockLayout_EmptyIFC_MinimumLineHeight verifies that a block element
// whose only child is an inline text node with empty data (no visible text)
// still occupies exactly 1 content row. This mirrors the browser rule that an
// IFC always reserves at least one line-box height, preventing the element
// from collapsing to zero content height.
func TestBlockLayout_EmptyIFC_MinimumLineHeight(t *testing.T) {
	// A text node that produces no clusters — simulates an empty <input> buffer.
	emptyText := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "",
	}
	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(20),
			Height:  style.Auto,
		},
		firstChild: emptyText,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)

	// The block must be exactly 1 row tall (content only, no border/padding).
	if frag.Size.Height != 1 {
		t.Errorf("empty IFC: height = %d, want 1 (minimum line-box reserve)", frag.Size.Height)
	}
	// A phantom child fragment IS added for the empty IFC for cursor tracking.
	if len(frag.Children) != 1 {
		t.Errorf("empty IFC: Children count = %d, want 1", len(frag.Children))
	}
}

// TestBlockLayout_EmptyIFC_WithBorder verifies the compound case that most
// directly caused the visual bug: a bordered single-line input widget with no
// content must produce height = border.Top(1) + content(1) + border.Bottom(1)
// = 3 rows, not 2. Without the minimum-line-box reserve the content row was
// missing, rendering two border lines back-to-back with no gap between them.
func TestBlockLayout_EmptyIFC_WithBorder(t *testing.T) {
	emptyText := &mockTextNode{
		mockNode: mockNode{
			style: &style.Computed{Display: style.DisplayInline},
		},
		data: "",
	}
	// Bordered inline-block, 20 cells wide — mirrors the InputElement default style.
	parent := &mockNode{
		style: &style.Computed{
			Display: style.DisplayInlineBlock,
			Width:   style.Cells(20),
			Height:  style.Auto,
			Border:  style.SingleBorder(),
		},
		firstChild: emptyText,
	}

	space := NewConstraintSpaceBuilder(Size{20, 10}).ToConstraintSpace()
	algo := GetAlgorithm(parent)
	frag := algo.Layout(nil, parent, space)

	// border.Top(1) + content(1) + border.Bottom(1) = 3.
	// Before the fix this was 2 (border.Top + border.Bottom only).
	const wantHeight = 3
	if frag.Size.Height != wantHeight {
		t.Errorf("empty IFC with border: height = %d, want %d (top-border + content + bottom-border)",
			frag.Size.Height, wantHeight)
	}
	if frag.Size.Width != 20 {
		t.Errorf("empty IFC with border: width = %d, want 20", frag.Size.Width)
	}
}
