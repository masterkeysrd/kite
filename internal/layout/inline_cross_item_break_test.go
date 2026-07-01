package layout

// Unit tests for the cross-item BreakSoft injection in InlineItemsBuilder.collectText.
//
// Background: the text shaper (text.Shape) operates per-text-node and cannot see
// across node boundaries.  A word boundary formed by a space at the end of one text
// node and a non-space at the start of the next is therefore never classified as
// BreakSoft by the shaper alone.  collectText must detect this situation from the
// builder's lastWasSpace flag and promote the first eligible cluster to BreakSoft.
//
// These tests exercise:
//  1. InlineItemsBuilder – verifying the BreakClass emitted on items produced by
//     Build, both within a single node and across node boundaries.
//  2. Full IFC layout (BlockAlgorithm + multiple mockTextNodes as children) –
//     verifying that the line breaker wraps at the correct width when the only
//     break opportunity exists between two adjacent text nodes.
//  3. Edge and boundary conditions: no trailing space, pre/pre-wrap whitespace,
//     empty middle nodes, leading-space collapsing on the second node, three-way
//     chaining, and the "fits on one line" non-regression.

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout/text"
	"github.com/masterkeysrd/kite/style"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// inlineTextStyle returns a minimal DisplayInline style.
func inlineTextStyle() *style.Computed {
	return &style.Computed{Display: style.DisplayInline}
}

// blockStyleW returns a DisplayBlock style with the given cell width.
func blockStyleW(w int) *style.Computed {
	return &style.Computed{Display: style.DisplayBlock, Width: style.Cells(w)}
}

// buildItemsFromBlock runs InlineItemsBuilder.Build on each direct inline child
// of blockNode (as block.go does) and returns the concatenated flat item list.
// This mimics the way processInlines works in block.go.
func buildItemsFromBlock(blockNode Node) []InlineItem {
	b := NewInlineItemsBuilder(text.NewShaper(0), blockNode)
	for child := blockNode.FirstLayoutChild(); child != nil; child = blockNode.NextLayoutSibling(child) {
		b.collect(child)
	}
	return b.items
}

// textItemsOnly filters an item list to InlineText entries.
func textItemsOnly(items []InlineItem) []InlineItem {
	var out []InlineItem
	for _, it := range items {
		if it.Type == InlineText {
			out = append(out, it)
		}
	}
	return out
}

// chainTextNodes links a sequence of mockTextNode values into a sibling list
// (each node's mockNode.nextSibling points to the next) and returns the first
// node as the head Node.
func chainTextNodes(nodes ...*mockTextNode) Node {
	for i := 0; i < len(nodes)-1; i++ {
		nodes[i].mockNode.nextSibling = nodes[i+1]
	}
	return nodes[0]
}

// runIFCLayout builds a block parent with the given inline children (connected via
// chainTextNodes) and runs the full block layout algorithm at the given width.
func runIFCLayout(t *testing.T, width int, nodes ...*mockTextNode) *Fragment {
	t.Helper()
	head := chainTextNodes(nodes...)
	parent := &mockNode{
		style:      blockStyleW(width),
		firstChild: head,
	}
	space := NewConstraintSpaceBuilder(geometry.Size{Width: width, Height: 1000}).
		SetIsFixedInlineSize(true).
		ToConstraintSpace()
	return GetAlgorithm(parent).Layout(nil, parent, space)
}

// newTextNode creates a mockTextNode with DisplayInline style and the given text.
func newTextNode(data string) *mockTextNode {
	return &mockTextNode{
		mockNode: mockNode{style: inlineTextStyle()},
		data:     data,
	}
}

// newTextNodeWS creates a mockTextNode with a specific WhiteSpace style.
func newTextNodeWS(data string, ws style.WhiteSpace) *mockTextNode {
	return &mockTextNode{
		mockNode: mockNode{style: &style.Computed{Display: style.DisplayInline, WhiteSpace: ws}},
		data:     data,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. InlineItemsBuilder – BreakClass on a single node (baseline)
// ─────────────────────────────────────────────────────────────────────────────

// TestInlineItemsBuilder_SingleNode_BreakSoftWithinWord verifies that the shaper
// correctly assigns BreakSoft to the non-space cluster that immediately follows a
// space within a single text node. This is pre-existing behaviour and serves as a
// baseline for the cross-item tests below.
func TestInlineItemsBuilder_SingleNode_BreakSoftWithinWord(t *testing.T) {
	// "hello world" → index 6 ('w') follows the space at index 5.
	node := newTextNode("hello world")
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: node,
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 1 {
		t.Fatalf("expected 1 InlineText item, got %d", len(items))
	}

	clusters := items[0].Text
	// h=0 e=1 l=2 l=3 o=4 ' '=5 w=6 …
	if clusters[6].BreakClass != text.BreakSoft {
		t.Errorf("cluster[6] ('w'): BreakClass = %v, want BreakSoft", clusters[6].BreakClass)
	}
	if clusters[0].BreakClass != text.BreakNone {
		t.Errorf("cluster[0] ('h'): BreakClass = %v, want BreakNone", clusters[0].BreakClass)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. InlineItemsBuilder – cross-item BreakSoft injection (the fix)
// ─────────────────────────────────────────────────────────────────────────────

// TestInlineItemsBuilder_CrossItem_BreakSoftAfterSpace is the core unit test for
// the fix. When node1 ends with a space and node2 starts with a non-space, the
// first cluster of node2 must be promoted to BreakSoft.
func TestInlineItemsBuilder_CrossItem_BreakSoftAfterSpace(t *testing.T) {
	node1 := newTextNode("foo ") // trailing space → lastWasSpace = true
	node2 := newTextNode("bar")  // 'b' must become BreakSoft
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	if items[1].Text[0].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[0] ('b'): BreakClass = %v, want BreakSoft (cross-item word boundary missing)", items[1].Text[0].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_NoBreakWhenNoTrailingSpace verifies that if
// node1 does NOT end with a space, no BreakSoft is injected onto node2.
func TestInlineItemsBuilder_CrossItem_NoBreakWhenNoTrailingSpace(t *testing.T) {
	node1 := newTextNode("foo") // no trailing space
	node2 := newTextNode("bar") // 'b' must remain BreakNone
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	if items[1].Text[0].BreakClass != text.BreakNone {
		t.Errorf("node2 cluster[0]: BreakClass = %v, want BreakNone (no trailing space on node1)", items[1].Text[0].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_NoBreakForPreNode verifies that the injection
// is suppressed when node2 has white-space:pre.  Pre nodes take a separate code
// path in collectText that bypasses the cross-item injection.
func TestInlineItemsBuilder_CrossItem_NoBreakForPreNode(t *testing.T) {
	node1 := newTextNode("foo ")
	node2 := newTextNodeWS("bar", style.WhiteSpacePre)
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	// The pre node is shaped verbatim; prevWasBreakableSpace=false for the shaper
	// so 'b' carries BreakNone from the shaper, and the builder injects nothing.
	if items[1].Text[0].BreakClass != text.BreakNone {
		t.Errorf("node2 (pre) cluster[0]: BreakClass = %v, want BreakNone (pre bypasses injection)", items[1].Text[0].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_ThreeNodes verifies that lastWasSpace is
// propagated correctly across three consecutive text nodes, each ending with a
// space, so that both node2[0] and node3[0] receive BreakSoft.
func TestInlineItemsBuilder_CrossItem_ThreeNodes(t *testing.T) {
	node1 := newTextNode("aa ")
	node2 := newTextNode("bb ")
	node3 := newTextNode("cc")
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2, node3),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 3 {
		t.Fatalf("expected 3 InlineText items, got %d", len(items))
	}

	if items[1].Text[0].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[0] ('b'): BreakClass = %v, want BreakSoft", items[1].Text[0].BreakClass)
	}
	if items[2].Text[0].BreakClass != text.BreakSoft {
		t.Errorf("node3 cluster[0] ('c'): BreakClass = %v, want BreakSoft", items[2].Text[0].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_EmptyMiddleNode verifies that an empty text
// node (which Shape returns nil for, causing collectText to return early) does not
// corrupt lastWasSpace — the BreakSoft is still injected on the next non-empty node.
func TestInlineItemsBuilder_CrossItem_EmptyMiddleNode(t *testing.T) {
	node1 := newTextNode("foo ")
	empty := newTextNode("") // Shape("") returns nil → collectText exits early
	node3 := newTextNode("bar")
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, empty, node3),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	// empty node produces no InlineText item.
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items (empty node contributes none), got %d", len(items))
	}

	if items[1].Text[0].BreakClass != text.BreakSoft {
		t.Errorf("node3 cluster[0] ('b'): BreakClass = %v, want BreakSoft (lastWasSpace survives empty middle node)", items[1].Text[0].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_LeadingSpaceOnSecondNode verifies that when
// node1 ends with a space and node2 starts with additional spaces, the leading
// spaces of node2 are collapsed (CellWidth=0) and the injection targets the first
// non-space cluster, not the collapsed spaces.
func TestInlineItemsBuilder_CrossItem_LeadingSpaceOnSecondNode(t *testing.T) {
	// node1: "foo " → node2: "  bar"
	// node2 leading spaces [0],[1] are collapsed; 'b' at [2] gets BreakSoft.
	node1 := newTextNode("foo ")
	node2 := newTextNode("  bar")
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	node2Clusters := items[1].Text
	// [0]=' ' [1]=' ' [2]='b' [3]='a' [4]='r'
	if node2Clusters[0].CellWidth != 0 {
		t.Errorf("node2 cluster[0] (1st space): CellWidth = %d, want 0 (collapsed)", node2Clusters[0].CellWidth)
	}
	if node2Clusters[1].CellWidth != 0 {
		t.Errorf("node2 cluster[1] (2nd space): CellWidth = %d, want 0 (collapsed)", node2Clusters[1].CellWidth)
	}
	// The 'b' at index 2 is the first non-space; it should be BreakSoft.
	if node2Clusters[2].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[2] ('b'): BreakClass = %v, want BreakSoft", node2Clusters[2].BreakClass)
	}
}

// TestInlineItemsBuilder_CrossItem_IntraNodeBreakUntouched verifies that
// BreakSoft classifications that were already assigned by the shaper within node2
// (e.g. the 'w' in "hello world") are preserved and not overwritten.
func TestInlineItemsBuilder_CrossItem_IntraNodeBreakUntouched(t *testing.T) {
	// node1: "x " → node2: "hello world"
	// node2[0] ('h') gets BreakSoft injected (cross-item).
	// node2[6] ('w') keeps BreakSoft from the shaper (intra-node).
	node1 := newTextNode("x ")
	node2 := newTextNode("hello world")
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	clusters := items[1].Text
	// Cross-item injection: 'h' at [0] must be BreakSoft.
	if clusters[0].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[0] ('h'): BreakClass = %v, want BreakSoft (cross-item)", clusters[0].BreakClass)
	}
	// Intra-node: 'w' at [6] must still be BreakSoft.
	if clusters[6].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[6] ('w'): BreakClass = %v, want BreakSoft (intra-node, must be preserved)", clusters[6].BreakClass)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. IFC layout – line-breaking across text node boundaries
// ─────────────────────────────────────────────────────────────────────────────

// TestIFC_CrossItemWrap_TwoNodes is the direct unit-level counterpart of the
// integration regression TestFlexWrapOn_PercentWidthWithTextItems.
//
//	node1: "AAAAAAAA " (9 cells, trailing space)
//	node2: "BBBBBBBB"  (8 cells)
//	width:  12
//
// "AAAAAAAA " fits (9≤12).  "AAAAAAAA BBBBBBBB" = 17 > 12 → wrap before node2.
// Expected: 2 line boxes; line0=9, line1=8.
func TestIFC_CrossItemWrap_TwoNodes(t *testing.T) {
	frag := runIFCLayout(t, 12,
		newTextNode("AAAAAAAA "),
		newTextNode("BBBBBBBB"),
	)

	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 line boxes (wrap at cross-item boundary), got %d", len(frag.Children))
	}
	if frag.Children[0].Fragment.Size.Width != 9 {
		t.Errorf("line 0 width = %d, want 9", frag.Children[0].Fragment.Size.Width)
	}
	if frag.Children[1].Fragment.Size.Width != 8 {
		t.Errorf("line 1 width = %d, want 8", frag.Children[1].Fragment.Size.Width)
	}
}

// TestIFC_CrossItemWrap_NoWrapWhenFits verifies that when the combined text of
// two adjacent nodes fits within the container, no spurious wrap is introduced by
// the cross-item BreakSoft injection.
func TestIFC_CrossItemWrap_NoWrapWhenFits(t *testing.T) {
	// "hello " (6) + "world" (5) = 11 < 20 → 1 line.
	frag := runIFCLayout(t, 20,
		newTextNode("hello "),
		newTextNode("world"),
	)

	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 line box (text fits in width), got %d", len(frag.Children))
	}
	if frag.Children[0].Fragment.Size.Width != 11 {
		t.Errorf("line 0 width = %d, want 11", frag.Children[0].Fragment.Size.Width)
	}
}

// TestIFC_CrossItemWrap_ThreeNodes verifies chained wrapping across three nodes.
//
//	node1: "AAAA "   (5)  node2: "BBBBBB " (7)  node3: "CCCC" (4)  width: 10
//
//	line 0: "AAAA "   = 5  (5+7=12 > 10 → wrap before node2)
//	line 1: "BBBBBB " = 7  (7+4=11 > 10 → wrap before node3)
//	line 2: "CCCC"    = 4
func TestIFC_CrossItemWrap_ThreeNodes(t *testing.T) {
	frag := runIFCLayout(t, 10,
		newTextNode("AAAA "),
		newTextNode("BBBBBB "),
		newTextNode("CCCC"),
	)

	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 line boxes, got %d", len(frag.Children))
	}
	wantWidths := []int{5, 7, 4}
	for i, want := range wantWidths {
		got := frag.Children[i].Fragment.Size.Width
		if got != want {
			t.Errorf("line %d width = %d, want %d", i, got, want)
		}
	}
}

// TestIFC_CrossItemWrap_SuppressedLeadingSpaceOnWrappedLine verifies that when a
// wrap occurs at a cross-item boundary, the leading space that triggered the break
// is visually suppressed on the new line (CSS §white-space normal behaviour).
//
//	node1: "AAAAAAAAAA " (11) → width=10, takes "AAAAAAAAAA " (11 > 10 but only
//	  break point is after the space at index 10; "AAAAAAAAAA" + space is taken
//	  and the trailing space is present on line 0).
//	node2: "BB" (2)
//
// Line 1 width must be 2, not 3 (no leading space rendered).
func TestIFC_CrossItemWrap_SuppressedLeadingSpaceOnWrappedLine(t *testing.T) {
	frag := runIFCLayout(t, 10,
		newTextNode("AAAAAAAAAA "),
		newTextNode("BB"),
	)

	if len(frag.Children) < 2 {
		t.Fatalf("expected ≥2 line boxes, got %d", len(frag.Children))
	}
	// Line 1 must be width 2 — "BB" with no leading space.
	if frag.Children[1].Fragment.Size.Width != 2 {
		t.Errorf("line 1 width = %d, want 2 (leading space on wrapped line must be suppressed)", frag.Children[1].Fragment.Size.Width)
	}
}

// TestIFC_CrossItemWrap_NoWrapWithoutTrailingSpace verifies that no spurious wrap
// is inserted when the first node ends without a space. The combined text overflows
// but there is no break opportunity → one overflowing line (OverflowWrapNormal).
func TestIFC_CrossItemWrap_NoWrapWithoutTrailingSpace(t *testing.T) {
	// "AAAAAAAAAA" (10, no space) + "BBBB" (4) = 14 > 10; no break → 1 line.
	frag := runIFCLayout(t, 10,
		newTextNode("AAAAAAAAAA"),
		newTextNode("BBBB"),
	)

	if len(frag.Children) != 1 {
		t.Fatalf("expected 1 line box (no break opportunity), got %d", len(frag.Children))
	}
	if frag.Children[0].Fragment.Size.Width != 14 {
		t.Errorf("line 0 width = %d, want 14 (both nodes on one overflowing line)", frag.Children[0].Fragment.Size.Width)
	}
}

// TestInlineItemsBuilder_CrossItem_PreTrailingSpace verifies that a pre/pre-wrap node
// ending with a space sets lastWasSpace = true, which correctly collapses leading spaces
// in a subsequent normal/nowrap node and promotes its first non-space cluster to BreakSoft.
func TestInlineItemsBuilder_CrossItem_PreTrailingSpace(t *testing.T) {
	node1 := newTextNodeWS("foo ", style.WhiteSpacePre)
	node2 := newTextNodeWS(" bar", style.WhiteSpaceNormal)
	parent := &mockNode{
		style:      blockStyleW(40),
		firstChild: chainTextNodes(node1, node2),
	}

	items := textItemsOnly(buildItemsFromBlock(parent))
	if len(items) != 2 {
		t.Fatalf("expected 2 InlineText items, got %d", len(items))
	}

	// node1 (pre) trailing space is visible
	if items[0].Text[3].CellWidth != 1 {
		t.Errorf("node1 (pre) trailing space CellWidth = %d, want 1", items[0].Text[3].CellWidth)
	}

	// node2 (normal) leading space should be collapsed to CellWidth = 0
	if items[1].Text[0].CellWidth != 0 {
		t.Errorf("node2 (normal) leading space CellWidth = %d, want 0 (collapsed)", items[1].Text[0].CellWidth)
	}

	// node2 first non-space cluster ('b' at index 1) should be BreakSoft
	if items[1].Text[1].BreakClass != text.BreakSoft {
		t.Errorf("node2 cluster[1] ('b') BreakClass = %v, want BreakSoft", items[1].Text[1].BreakClass)
	}
}
