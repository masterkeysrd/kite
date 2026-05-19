package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
)

func TestInlineTreeConstruction(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Create a hierarchy: Box > Span > Text

	// Create a hierarchy: Box > Span > Text
	box := element.NewBox(doc)
	span := element.NewSpan(doc)
	text := element.NewText(doc, "Hello")

	box.AppendChild(span)
	span.AppendChild(text)
	doc.AppendChild(box)

	// Trigger sync phase
	eng.Frame()

	// Verify Render Objects are created
	boxRO := box.RenderObject()
	if boxRO == nil {
		t.Fatal("Box render object not created")
	}

	spanRO := span.RenderObject()
	if spanRO == nil {
		t.Fatal("Span render object not created")
	}

	textRO := text.RenderObject()
	if textRO == nil {
		t.Fatal("Text render object not created")
	}

	// Verify Render Tree Structure
	if spanRO.Parent() != boxRO {
		t.Errorf("Span RO parent should be Box RO")
	}
	if textRO.Parent() != spanRO {
		t.Errorf("Text RO parent should be Span RO")
	}
}

func TestMixedTreeConstruction(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// Box

	// Box
	//  - Span
	//  - Box (nested)
	box := element.NewBox(doc)
	span := element.NewSpan(doc)
	nestedBox := element.NewBox(doc)

	box.AppendChild(span)
	box.AppendChild(nestedBox)
	doc.AppendChild(box)

	eng.Frame()

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
