package dom_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func TestDocument_Overlays_Sorting(t *testing.T) {
	doc := dom.NewDocument()
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)
	c := doc.CreateElement("c", nil)
	d := doc.CreateElement("d", nil)

	// Add in non-sorted order.
	doc.ShowOverlay(b, 10)
	doc.ShowOverlay(a, 5)
	doc.ShowOverlay(d, 10)
	doc.ShowOverlay(c, 0)

	// Expected order (by z-index, then insertion):
	// c (0), a (5), b (10, first), d (10, second)
	expected := []dom.Element{c, a, b, d}
	i := 0
	for el := range doc.Overlays() {
		if i >= len(expected) {
			t.Errorf("too many overlays, got more than %d", len(expected))
			break
		}
		if el != expected[i] {
			t.Errorf("at index %d: got %s, want %s", i, el.TagName(), expected[i].TagName())
		}
		i++
	}
	if i != len(expected) {
		t.Errorf("too few overlays, got %d, want %d", i, len(expected))
	}
}

func TestDocument_HideOverlay(t *testing.T) {
	doc := dom.NewDocument()
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)

	doc.ShowOverlay(a, 1)
	doc.ShowOverlay(b, 2)

	doc.HideOverlay(a)

	count := 0
	for el := range doc.Overlays() {
		if el != b {
			t.Errorf("got %s, want b", el.TagName())
		}
		count++
	}
	if count != 1 {
		t.Errorf("got %d overlays, want 1", count)
	}
}

func TestDocument_UpdateOverlayZIndex(t *testing.T) {
	doc := dom.NewDocument()
	a := doc.CreateElement("a", nil)
	b := doc.CreateElement("b", nil)

	doc.ShowOverlay(a, 1)
	doc.ShowOverlay(b, 2)

	// Update a to be above b.
	doc.ShowOverlay(a, 3)

	expected := []dom.Element{b, a}
	i := 0
	for el := range doc.Overlays() {
		if el != expected[i] {
			t.Errorf("at index %d: got %s, want %s", i, el.TagName(), expected[i].TagName())
		}
		i++
	}
}
