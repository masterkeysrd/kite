package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func TestPlaceholder(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("Input placeholder", func(t *testing.T) {
		inp := element.NewInput(doc, "").WithPlaceholder("Search...")

		// Verify placeholder exists in shadow DOM.
		uaRoot := dom.UARoot(inp)
		if uaRoot == nil {
			t.Fatal("expected UA root")
		}
		uaDiv := uaRoot.FirstChild()
		if uaDiv == nil {
			t.Fatal("expected UA div")
		}

		// Initial state: empty buffer, placeholder should be visible.
		placeholderNode := uaDiv.FirstChild()
		if placeholderNode == nil || placeholderNode.NodeName() != "ua-placeholder" {
			t.Fatalf("expected placeholder element, got %v", placeholderNode)
		}
		if placeholderNode.(dom.Element).TextContent() != "Search..." {
			t.Errorf("expected 'Search...', got %q", placeholderNode.(dom.Element).TextContent())
		}

		// Type something: placeholder should disappear.
		inp.SetValue("hello")
		placeholderNode = uaDiv.FirstChild()
		if placeholderNode != nil && placeholderNode.NodeName() == "ua-placeholder" {
			t.Error("placeholder should be removed when input is not empty")
		}

		// Clear input: placeholder should reappear.
		inp.SetValue("")
		placeholderNode = uaDiv.FirstChild()
		if placeholderNode == nil || placeholderNode.NodeName() != "ua-placeholder" {
			t.Error("placeholder should reappear when input is cleared")
		}

		// Verify placeholder is inline to avoid extra rows.
		disp, _ := placeholderNode.(element.Element).DefaultStyle().DisplayOpt().Get()
		if disp != style.DisplayInline {
			t.Errorf("placeholder must be DisplayInline, got %v", disp)
		}
	})

	t.Run("TextArea placeholder", func(t *testing.T) {
		txa := element.NewTextArea(doc, "").WithPlaceholder("Notes...")

		uaRoot := dom.UARoot(txa)
		uaDiv := uaRoot.FirstChild()

		// Initial state.
		placeholderNode := uaDiv.FirstChild()
		if placeholderNode == nil || placeholderNode.NodeName() != "ua-placeholder" {
			t.Fatalf("expected placeholder element, got %v", placeholderNode)
		}

		// Type something.
		txa.SetValue("line 1")
		placeholderNode = uaDiv.FirstChild()
		if placeholderNode != nil && placeholderNode.NodeName() == "ua-placeholder" {
			t.Error("placeholder should be removed when textarea is not empty")
		}

		// Clear.
		txa.SetValue("")
		placeholderNode = uaDiv.FirstChild()
		if placeholderNode == nil || placeholderNode.NodeName() != "ua-placeholder" {
			t.Error("placeholder should reappear when textarea is cleared")
		}

		// Verify placeholder is inline.
		disp, _ := placeholderNode.(element.Element).DefaultStyle().DisplayOpt().Get()
		if disp != style.DisplayInline {
			t.Errorf("placeholder must be DisplayInline, got %v", disp)
		}
	})
}

// TestPlaceholder_Update_NoDuplicateInDOM is a regression test for the bug
// where changing the placeholder text while the input is empty left the old
// placeholder element in the DOM and inserted a new one beside it, causing
// the placeholder text to appear duplicated/concatenated in the layout.
func TestPlaceholder_Update_NoDuplicateInDOM(t *testing.T) {
	doc := dom.NewDocument()
	inp := element.NewInput(doc, "").WithPlaceholder("First")

	uaRoot := dom.UARoot(inp)
	uaDiv := uaRoot.FirstChild()

	// Count ua-placeholder nodes in the UA div.
	countPlaceholders := func() int {
		n := 0
		for child := uaDiv.FirstChild(); child != nil; child = child.NextSibling() {
			if child.NodeName() == "ua-placeholder" {
				n++
			}
		}
		return n
	}

	if got := countPlaceholders(); got != 1 {
		t.Fatalf("initial: expected 1 placeholder, got %d", got)
	}

	// Change placeholder text while buffer is still empty.
	inp.WithPlaceholder("Second")

	if got := countPlaceholders(); got != 1 {
		t.Errorf("after update: expected 1 placeholder, got %d (duplication bug)", got)
	}

	// Verify it shows the new text.
	first := uaDiv.FirstChild()
	if first == nil || first.NodeName() != "ua-placeholder" {
		t.Fatal("expected a placeholder node after update")
	}
	if got := first.(dom.Element).TextContent(); got != "Second" {
		t.Errorf("expected placeholder text 'Second', got %q", got)
	}
}
