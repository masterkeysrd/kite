package element

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/render"
)

func TestInlineTreeConstruction(t *testing.T) {
	doc := dom.NewDocument()

	// Create a hierarchy: Box > Span > Text
	box := NewBox(doc)
	span := NewSpan(doc)
	text := NewText(doc, "Hello")

	box.AppendChild(span)
	span.AppendChild(text)

	// Manually trigger OnConnected (simulating the engine/document attach walk)
	// In a real app, doc.AppendChild(box) would trigger this.
	box.OnConnected()
	span.OnConnected()
	text.OnConnected()

	// Verify Render Objects are created
	boxRO := box.RenderObject()
	if boxRO == nil {
		t.Fatal("Box render object not created")
	}
	if _, ok := boxRO.(*render.Block); !ok {
		t.Errorf("expected *render.Block, got %T", boxRO)
	}

	spanRO := span.RenderObject()
	if spanRO == nil {
		t.Fatal("Span render object not created")
	}
	if _, ok := spanRO.(*render.Inline); !ok {
		t.Errorf("expected *render.Inline, got %T", spanRO)
	}

	textRO := text.RenderObject()
	if textRO == nil {
		t.Fatal("Text render object not created")
	}
	if _, ok := textRO.(*render.Text); !ok {
		t.Errorf("expected *render.Text, got %T", textRO)
	}

	// Verify Render Tree Structure
	if spanRO.Parent() != boxRO {
		t.Errorf("Span RO parent should be Box RO")
	}
	if textRO.Parent() != spanRO {
		t.Errorf("Text RO parent should be Span RO")
	}

	if boxRO.FirstChild() != spanRO {
		t.Errorf("Box RO first child should be Span RO")
	}
	if spanRO.FirstChild() != textRO {
		t.Errorf("Span RO first child should be Text RO")
	}
}

func TestMixedTreeConstruction(t *testing.T) {
	doc := dom.NewDocument()

	// Box
	//  - Span
	//  - Box (nested)
	box := NewBox(doc)
	span := NewSpan(doc)
	nestedBox := NewBox(doc)

	box.AppendChild(span)
	box.AppendChild(nestedBox)

	box.OnConnected()
	span.OnConnected()
	nestedBox.OnConnected()

	boxRO := box.RenderObject()
	spanRO := span.RenderObject()
	nestedBoxRO := nestedBox.RenderObject()

	if boxRO.FirstChild() != spanRO {
		t.Errorf("expected Span RO as first child")
	}
	if spanRO.NextSibling() != nestedBoxRO {
		t.Errorf("expected nested Box RO as next sibling of Span RO")
	}
}
