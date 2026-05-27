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

	r := &rangeImpl{doc: doc}

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

	r := &rangeImpl{doc: doc}

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
	r := &rangeImpl{doc: doc}
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
