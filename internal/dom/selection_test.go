package dom

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
)

func TestRange_String(t *testing.T) {
	doc := NewDocument()
	div := doc.CreateElement("div", nil)
	doc.AppendChild(div)

	t1 := doc.CreateTextNode("Hello ", nil)
	t2 := doc.CreateTextNode("World", nil)
	div.AppendChild(t1)
	div.AppendChild(t2)

	r := &Range{doc: doc}

	// Case 1: Within same text node
	r.SetStart(t1, 0)
	r.SetEnd(t1, 5)
	if r.String() != "Hello" {
		t.Errorf("expected 'Hello', got %q", r.String())
	}

	// Case 2: Across sibling text nodes
	r.SetStart(t1, 0)
	r.SetEnd(t2, 5)
	if r.String() != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", r.String())
	}

	// Case 3: Partial across sibling text nodes
	r.SetStart(t1, 3)
	r.SetEnd(t2, 2)
	if r.String() != "lo Wo" {
		t.Errorf("expected 'lo Wo', got %q", r.String())
	}
}

func TestRange_String_Elements(t *testing.T) {
	doc := NewDocument()
	// <div>
	//   <span>A</span>
	//   <span>B</span>
	//   <span>C</span>
	// </div>
	div := doc.CreateElement("div", nil)
	s1 := doc.CreateElement("span", nil)
	s1.AppendChild(doc.CreateTextNode("A", nil))
	s2 := doc.CreateElement("span", nil)
	s2.AppendChild(doc.CreateTextNode("B", nil))
	s3 := doc.CreateElement("span", nil)
	s3.AppendChild(doc.CreateTextNode("C", nil))
	div.AppendChild(s1)
	div.AppendChild(s2)
	div.AppendChild(s3)
	doc.AppendChild(div)

	r := &Range{doc: doc}

	// Range from div start to child index 2 (before s3)
	r.SetStart(div, 0)
	r.SetEnd(div, 2)
	// Should include s1 ("A") and s2 ("B")
	if r.String() != "AB" {
		t.Errorf("expected 'AB', got %q", r.String())
	}
}

func TestSelection_Events(t *testing.T) {
	doc := NewDocument()
	fired := 0
	doc.AddEventListener(event.EventSelectionChange, func(e event.Event) {
		fired++
	})

	sel := doc.Selection()
	r := &Range{doc: doc}
	r.SetStart(doc, 0) // Should trigger change via notifyChange -> sel.changed

	if fired == 0 {
		t.Error("expected selectionchange event to fire on SetStart")
	}

	fired = 0
	sel.AddRange(r)
	if fired == 0 {
		t.Error("expected selectionchange event to fire on AddRange")
	}

	fired = 0
	sel.RemoveAllRanges()
	if fired == 0 {
		t.Error("expected selectionchange event to fire on RemoveAllRanges")
	}
}

func TestRange_String_AncestorStart(t *testing.T) {
	doc := NewDocument()
	div := doc.CreateElement("div", nil)
	t1 := doc.CreateTextNode("Line 1", nil)
	br := doc.CreateElement("br", nil)
	t2 := doc.CreateTextNode("Line 2", nil)
	div.AppendChild(t1)
	div.AppendChild(br)
	div.AppendChild(t2)
	doc.AppendChild(div)

	r := doc.CreateRange()
	// Start at div, offset 2 (which is t2)
	r.SetStart(div, 2)
	// End at t2, offset 1 (the 'L' of "Line 2")
	r.SetEnd(t2, 1)

	got := r.String()
	want := "L"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestComparePositions(t *testing.T) {
	doc := NewDocument()
	div := doc.CreateElement("div", nil)
	t1 := doc.CreateTextNode("Hello", nil)
	t2 := doc.CreateTextNode("World", nil)
	div.AppendChild(t1)
	div.AppendChild(t2)
	doc.AppendChild(div)

	// Test 1: Compare within same node
	if cmp := doc.comparePositions(t1, 0, t1, 3); cmp >= 0 {
		t.Errorf("expected negative comparison, got %d", cmp)
	}
	if cmp := doc.comparePositions(t1, 3, t1, 0); cmp <= 0 {
		t.Errorf("expected positive comparison, got %d", cmp)
	}

	// Test 2: Compare across different nodes (t1 before t2 in preorder)
	if cmp := doc.comparePositions(t1, 0, t2, 0); cmp >= 0 {
		t.Errorf("expected negative comparison for t1 before t2, got %d", cmp)
	}
	if cmp := doc.comparePositions(t2, 0, t1, 0); cmp <= 0 {
		t.Errorf("expected positive comparison for t2 after t1, got %d", cmp)
	}

	// Test 3: Element child comparisons (div before t2 in preorder)
	if cmp := doc.comparePositions(div, 0, t2, 0); cmp >= 0 {
		t.Errorf("expected negative comparison for parent div before child t2, got %d", cmp)
	}

	// Test 4: Invalidation test
	doc.InvalidateTextNodeCache()
	t3 := doc.CreateTextNode("!", nil)
	div.AppendChild(t3)

	if cmp := doc.comparePositions(t2, 0, t3, 0); cmp >= 0 {
		t.Errorf("expected negative comparison for t2 before new child t3, got %d", cmp)
	}

	// Test 5: Detached node fallback (a node not connected to the document)
	detached := doc.CreateTextNode("detached", nil)
	if cmp := doc.comparePositions(t1, 0, detached, 0); cmp >= 0 {
		// Should fall back gracefully and order them (usually detached is not found in walk, so it's placed after/before depending on search)
		t.Logf("Comparison with detached node: %d", cmp)
	}
}

func TestFindNodeAtByteOffset_Overlays(t *testing.T) {
	doc := NewDocument()
	body := doc.CreateElement("div", nil)
	t1 := doc.CreateTextNode("Hello", nil) // 5 bytes: 0-5
	body.AppendChild(t1)
	doc.AppendChild(body)

	overlay := doc.CreateElement("div", nil)
	t2 := doc.CreateTextNode("World", nil) // 5 bytes: 5-10
	overlay.AppendChild(t2)
	doc.ShowOverlay(overlay, 1)

	// Invalidate to make sure caches are clean
	doc.InvalidateTextNodeCache()

	// Find node at offset 2 (should be in t1, "Hello")
	node, offset := doc.FindNodeAtByteOffset(doc, 2)
	if node != t1 || offset != 2 {
		t.Errorf("expected t1 at offset 2, got %v with offset %d", node, offset)
	}

	// Find node at offset 7 (should be in t2, "World", since t1 is 5 bytes and t2 starts at 5)
	node, offset = doc.FindNodeAtByteOffset(doc, 7)
	if node != t2 || offset != 2 {
		t.Errorf("expected t2 at offset 2 (total offset 7), got %v with offset %d", node, offset)
	}

	// Compare t1 and t2 (t2 is in overlay, so should be after t1)
	if cmp := doc.comparePositions(t1, 0, t2, 0); cmp >= 0 {
		t.Errorf("expected t1 in body before t2 in overlay, got %d", cmp)
	}
}

func TestTextNodeCacheInvalidation(t *testing.T) {
	doc := NewDocument()
	body := doc.CreateElement("div", nil)
	doc.AppendChild(body)

	// Step 1: Initial state with one text node
	t1 := doc.CreateTextNode("Hello", nil) // 5 bytes: 0-5
	body.AppendChild(t1)

	// Trigger cache build
	node, offset := doc.FindNodeAtByteOffset(doc, 2)
	if node != t1 || offset != 2 {
		t.Errorf("expected t1 at offset 2, got %v with offset %d", node, offset)
	}

	// Step 2: Structural change - Insert a new text node BEFORE t1
	t0 := doc.CreateTextNode("AB", nil) // 2 bytes: 0-2
	body.InsertBefore(t0, t1)

	// Cache should be automatically invalidated by InsertBefore.
	// Now t0 is bytes 0-2 ("AB"), t1 is bytes 2-7 ("Hello").
	node, offset = doc.FindNodeAtByteOffset(doc, 1)
	if node != t0 || offset != 1 {
		t.Errorf("expected t0 at offset 1, got %v with offset %d", node, offset)
	}

	node, offset = doc.FindNodeAtByteOffset(doc, 4) // 4 is index 2 in "Hello"
	if node != t1 || offset != 2 {
		t.Errorf("expected t1 at offset 4, got %v with offset %d", node, offset)
	}

	// Step 3: Structural change - Remove t0
	body.RemoveChild(t0)

	// Cache should be invalidated by RemoveChild.
	// t1 is back to being bytes 0-5.
	node, offset = doc.FindNodeAtByteOffset(doc, 2)
	if node != t1 || offset != 2 {
		t.Errorf("expected t1 at offset 2 after removal, got %v with offset %d", node, offset)
	}

	// Step 4: Structural change - Overlay
	overlay := doc.CreateElement("div", nil)
	to := doc.CreateTextNode("World", nil) // 5 bytes
	overlay.AppendChild(to)
	doc.ShowOverlay(overlay, 1)

	// Cache should be invalidated by ShowOverlay.
	// t1 is bytes 0-5, to is bytes 5-10.
	node, offset = doc.FindNodeAtByteOffset(doc, 6)
	if node != to || offset != 1 {
		t.Errorf("expected overlay node at offset 6, got %v with offset %d", node, offset)
	}
}
