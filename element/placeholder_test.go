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
