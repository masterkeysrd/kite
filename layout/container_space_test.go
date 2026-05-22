package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// BuildChildSpace unit tests (TSK-041)
// ---------------------------------------------------------------------------

// makeParentSpace is a helper that creates a parent ConstraintSpace with the
// given available/containing/container sizes.
func makeParentSpace(available, containing, container Size, fixedBlock bool) ConstraintSpace {
	b := NewConstraintSpaceBuilder(available)
	b.SetContainingSpace(containing)
	b.SetContainerSpace(container)
	b.SetIsFixedInlineSize(true)
	if fixedBlock {
		b.SetIsFixedBlockSize(true)
	}
	return b.ToConstraintSpace()
}

func TestBuildChildSpace_KindCells(t *testing.T) {
	child := &mockNode{style: &style.Computed{
		Width:  style.Cells(10),
		Height: style.Auto,
	}}

	parentSpace := makeParentSpace(Size{50, 50}, Size{50, 50}, Size{40, 40}, false)
	cs := BuildChildSpace(child, Size{40, 40}, Size{50, 50}, parentSpace)

	if !cs.IsFixedInlineSize {
		t.Error("expected IsFixedInlineSize=true for KindCells")
	}
	if cs.AvailableSize.Width != 10 {
		t.Errorf("expected AvailableSize.Width=10, got %d", cs.AvailableSize.Width)
	}
}

func TestBuildChildSpace_KindPercent(t *testing.T) {
	// 50% of a 100-wide containing space (border-box) must give 50.
	child := &mockNode{style: &style.Computed{
		Width:  style.Percent(50),
		Height: style.Auto,
	}}

	containingSpace := Size{Width: 100, Height: 40}
	containerSpace := Size{Width: 80, Height: 30} // content-box (smaller)
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, false)
	cs := BuildChildSpace(child, containerSpace, containingSpace, parentSpace)

	if !cs.IsFixedInlineSize {
		t.Error("expected IsFixedInlineSize=true for KindPercent")
	}
	if cs.AvailableSize.Width != 50 {
		t.Errorf("expected AvailableSize.Width=50 (50%% of border-box 100), got %d", cs.AvailableSize.Width)
	}
}

func TestBuildChildSpace_KindAuto(t *testing.T) {
	// auto width with margin should shrink available width.
	child := &mockNode{style: &style.Computed{
		Width:  style.Auto,
		Height: style.Auto,
		Margin: style.EdgeValues[int]{Left: 5, Right: 5},
	}}

	containerSpace := Size{Width: 80, Height: 40}
	containingSpace := Size{Width: 100, Height: 50}
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, false)
	cs := BuildChildSpace(child, containerSpace, containingSpace, parentSpace)

	if !cs.IsFixedInlineSize {
		t.Error("expected IsFixedInlineSize=true for KindAuto non-table")
	}
	// childAvailWidth = 80 - 5 - 5 = 70
	if cs.AvailableSize.Width != 70 {
		t.Errorf("expected AvailableSize.Width=70, got %d", cs.AvailableSize.Width)
	}
}

func TestBuildChildSpace_KindMaxContent(t *testing.T) {
	child := &mockNode{style: &style.Computed{
		Width:  style.MaxContent,
		Height: style.Auto,
	}}

	parentSpace := makeParentSpace(Size{80, 40}, Size{100, 50}, Size{80, 40}, false)
	cs := BuildChildSpace(child, Size{80, 40}, Size{100, 50}, parentSpace)

	if cs.IsFixedInlineSize {
		t.Error("expected IsFixedInlineSize=false for KindMaxContent")
	}
}

func TestBuildChildSpace_HeightPercent_FixedParent(t *testing.T) {
	child := &mockNode{style: &style.Computed{
		Width:  style.Auto,
		Height: style.Percent(50),
	}}

	containingSpace := Size{Width: 100, Height: 40}
	containerSpace := Size{Width: 80, Height: 30}
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, true) // fixed block

	cs := BuildChildSpace(child, containerSpace, containingSpace, parentSpace)

	if !cs.IsFixedBlockSize {
		t.Error("expected IsFixedBlockSize=true for percent height with fixed parent")
	}
	// 50% of containingSpace.Height=40 = 20
	if cs.AvailableSize.Height != 20 {
		t.Errorf("expected AvailableSize.Height=20, got %d", cs.AvailableSize.Height)
	}
}

func TestBuildChildSpace_HeightPercent_AutoParent(t *testing.T) {
	child := &mockNode{style: &style.Computed{
		Width:  style.Auto,
		Height: style.Percent(50),
	}}

	containingSpace := Size{Width: 100, Height: 40}
	containerSpace := Size{Width: 80, Height: 30}
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, false) // NOT fixed block

	cs := BuildChildSpace(child, containerSpace, containingSpace, parentSpace)

	if cs.IsFixedBlockSize {
		t.Error("expected IsFixedBlockSize=false when parent block size is not fixed")
	}
}

func TestBuildChildSpace_PassthroughFields(t *testing.T) {
	child := &mockNode{style: &style.Computed{
		Width:  style.Auto,
		Height: style.Auto,
	}}

	containingSpace := Size{Width: 80, Height: 60}
	containerSpace := Size{Width: 60, Height: 40}
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, false)

	cs := BuildChildSpace(child, containerSpace, containingSpace, parentSpace)

	if cs.ContainingSpace != containingSpace {
		t.Errorf("expected ContainingSpace=%v, got %v", containingSpace, cs.ContainingSpace)
	}
	if cs.ContainerSpace != containerSpace {
		t.Errorf("expected ContainerSpace=%v, got %v", containerSpace, cs.ContainerSpace)
	}
}

func TestBuildChildSpace_DisplayTable_Auto(t *testing.T) {
	// Tables with auto width should NOT get IsFixedInlineSize (they shrink-wrap).
	child := &mockNode{style: &style.Computed{
		Display: style.DisplayTable,
		Width:   style.Auto,
		Height:  style.Auto,
	}}

	parentSpace := makeParentSpace(Size{80, 40}, Size{100, 50}, Size{80, 40}, false)
	cs := BuildChildSpace(child, Size{80, 40}, Size{100, 50}, parentSpace)

	if cs.IsFixedInlineSize {
		t.Error("expected IsFixedInlineSize=false for table with auto width")
	}
}

// ---------------------------------------------------------------------------
// Integration tests — full layout pass
// ---------------------------------------------------------------------------

func TestPercentResolvesAgainstBorderBox(t *testing.T) {
	// Parent: width=20 border=1 each side padding=2 each side → content=14.
	// Child: width=50% → must resolve against parent's border-box (20), giving 10.
	// Old code wrongly resolved against content-box (14), giving 7.
	parentStyle := &style.Computed{
		Width:   style.Cells(20),
		Height:  style.Auto,
		Border:  style.SingleBorder(),
		Padding: style.EdgeValues[int]{Left: 2, Right: 2, Top: 2, Bottom: 2},
	}
	childStyle := &style.Computed{
		Width:  style.Percent(50),
		Height: style.Cells(1),
	}

	child := &mockNode{style: childStyle}
	parent := &mockNode{style: parentStyle, firstChild: child}

	// Do NOT fix the inline size so the parent resolves its own KindCells width=20.
	space := NewConstraintSpaceBuilder(Size{Width: 100, Height: 100}).
		SetContainingSpace(Size{Width: 100, Height: 100}).
		SetContainerSpace(Size{Width: 100, Height: 100}).
		ToConstraintSpace()

	frag := (&BlockAlgorithm{Node: parent, Space: space}).Layout()

	// Parent is width=20 (KindCells). Child should be 50% of 20 = 10.
	if len(frag.Children) == 0 {
		t.Fatal("expected at least one child fragment")
	}
	childFrag := frag.Children[0].Fragment
	if childFrag.Size.Width != 10 {
		t.Errorf("expected child width=10 (50%% of border-box 20), got %d", childFrag.Size.Width)
	}
}

func TestContainerSpaceFlowsToGrandchild(t *testing.T) {
	// Three-level tree: viewport(W:100) → parent(W:50, P:5each) → child(W:50%).
	// Child's 50% must resolve against parent's border-box (50), not viewport (100).
	parentStyle := &style.Computed{
		Width:   style.Cells(50),
		Height:  style.Auto,
		Padding: style.EdgeValues[int]{Left: 5, Right: 5, Top: 0, Bottom: 0},
	}
	grandchildStyle := &style.Computed{
		Width:  style.Percent(50),
		Height: style.Cells(1),
	}
	grandchild := &mockNode{style: grandchildStyle}
	parent := &mockNode{style: parentStyle, firstChild: grandchild}
	root := &mockNode{
		style: &style.Computed{Width: style.Cells(100), Height: style.Auto},
		firstChild: parent,
	}

	// Do NOT fix the inline size so parent resolves its own KindCells width=50.
	space := NewConstraintSpaceBuilder(Size{100, 100}).
		SetContainingSpace(Size{100, 100}).
		SetContainerSpace(Size{100, 100}).
		ToConstraintSpace()

	rootFrag := (&BlockAlgorithm{Node: root, Space: space}).Layout()

	// Navigate: root → parent → grandchild
	if len(rootFrag.Children) == 0 {
		t.Fatal("root has no children")
	}
	parentFrag := rootFrag.Children[0].Fragment
	if len(parentFrag.Children) == 0 {
		t.Fatal("parent has no children")
	}
	gcFrag := parentFrag.Children[0].Fragment

	// grandchild width = 50% of parent border-box (50) = 25
	if gcFrag.Size.Width != 25 {
		t.Errorf("expected grandchild width=25 (50%% of parent border-box 50), got %d", gcFrag.Size.Width)
	}
}

func TestBlockChildUsesContainerSpace(t *testing.T) {
	// Parent: W=40, border=1 each side, padding=2 each side.
	// Child: W=auto, margin=3 each side.
	// childAvailWidth = 40 - 2 (border) - 4 (padding) - 6 (margin) = 28.
	parentStyle := &style.Computed{
		Width:   style.Cells(40),
		Height:  style.Auto,
		Border:  style.SingleBorder(),
		Padding: style.EdgeValues[int]{Left: 2, Right: 2, Top: 0, Bottom: 0},
	}
	childStyle := &style.Computed{
		Width:  style.Auto,
		Height: style.Cells(1),
		Margin: style.EdgeValues[int]{Left: 3, Right: 3},
	}
	child := &mockNode{style: childStyle}
	parent := &mockNode{style: parentStyle, firstChild: child}

	// Do NOT fix inline size so parent resolves its own KindCells width=40.
	space := NewConstraintSpaceBuilder(Size{100, 100}).
		SetContainingSpace(Size{100, 100}).
		SetContainerSpace(Size{100, 100}).
		ToConstraintSpace()

	frag := (&BlockAlgorithm{Node: parent, Space: space}).Layout()

	if len(frag.Children) == 0 {
		t.Fatal("expected child fragment")
	}
	childFrag := frag.Children[0].Fragment

	// 40 - 1(L) - 1(R) - 2(PL) - 2(PR) - 3(ML) - 3(MR) = 28
	if childFrag.Size.Width != 28 {
		t.Errorf("expected child width=28, got %d", childFrag.Size.Width)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks (TSK-041)
// ---------------------------------------------------------------------------

func BenchmarkBuildChildSpace(b *testing.B) {
	child := &mockNode{style: &style.Computed{
		Width:  style.Auto,
		Height: style.Auto,
		Margin: style.EdgeValues[int]{Left: 2, Right: 2, Top: 1, Bottom: 1},
	}}
	containerSpace := Size{Width: 80, Height: 40}
	containingSpace := Size{Width: 100, Height: 50}
	parentSpace := makeParentSpace(containerSpace, containingSpace, containerSpace, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildChildSpace(child, containerSpace, containingSpace, parentSpace)
	}
}

func BenchmarkLayoutWithContainerSpace(b *testing.B) {
	// Build a 50-node tree with mixed block and inline children.
	makeChild := func(w, h int) *mockNode {
		n := &mockNode{style: &style.Computed{
			Width:  style.Cells(w),
			Height: style.Cells(h),
		}}
		return n
	}

	// Link 50 siblings.
	first := makeChild(10, 1)
	prev := first
	for i := 1; i < 50; i++ {
		n := makeChild(10, 1)
		prev.nextSibling = n
		prev = n
	}

	root := &mockNode{
		style: &style.Computed{
			Display: style.DisplayBlock,
			Width:   style.Cells(100),
			Height:  style.Auto,
		},
		firstChild: first,
	}

	space := NewConstraintSpaceBuilder(Size{100, 1000}).
		SetContainingSpace(Size{100, 1000}).
		SetContainerSpace(Size{100, 1000}).
		SetIsFixedInlineSize(true).
		ToConstraintSpace()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.cachedFragment = nil
		// Invalidate children caches too.
		for n := root.firstChild; n != nil; {
			type sibling interface{ NextSibling() Node }
			mn := n.(*mockNode)
			mn.cachedFragment = nil
			n = n.(sibling).NextSibling()
		}
		algo := &BlockAlgorithm{Node: root, Space: space}
		algo.Layout()
	}
}
