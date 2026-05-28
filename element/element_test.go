package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
)

func TestDeclarativeAPI(t *testing.T) {
	// Test case 1: Box("Hello", Span("World"))
	box := element.Box("Hello", element.Span("World"))

	if box.NodeName() != "box" {
		t.Errorf("expected box, got %s", box.NodeName())
	}

	// First child should be text "Hello"
	child1 := box.FirstChild()
	if child1 == nil || child1.NodeName() != "#text" || child1.TextContent() != "Hello" {
		t.Errorf("expected first child to be text 'Hello', got %v", child1)
	}

	// Second child should be span "World"
	child2 := child1.NextSibling()
	if child2 == nil || child2.NodeName() != "span" {
		t.Errorf("expected second child to be span, got %v", child2)
	}
	if child2.TextContent() != "World" {
		t.Errorf("expected span text to be 'World', got %q", child2.TextContent())
	}

	// Test case 2: Flattening slices
	box2 := element.Box([]any{"a", "b"})
	if box2.FirstChild().TextContent() != "a" {
		t.Errorf("expected 'a'")
	}
	if box2.FirstChild().NextSibling().TextContent() != "b" {
		t.Errorf("expected 'b'")
	}
}

func TestInlineTreeConstruction(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Using declarative API
	box := element.Box(
		element.Span(
			element.Text("Hello"),
		),
	)
	eng.Mount(box)

	// Trigger sync phase
	eng.Frame()

	// Verify Render Objects are created
	boxRO := eng.RenderObject(box)
	if boxRO == nil {
		t.Fatal("Box render object not created")
	}

	span := box.FirstChild()
	spanRO := eng.RenderObject(span)
	if spanRO == nil {
		t.Fatal("Span render object not created")
	}

	text := span.FirstChild()
	textRO := eng.RenderObject(text)
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

	box := element.Box(
		element.Span(),
		element.Box(),
	)
	eng.Mount(box)

	eng.Frame()

	boxRO := eng.RenderObject(box)
	span := box.FirstChild()
	spanRO := eng.RenderObject(span)
	nestedBox := span.NextSibling()
	nestedBoxRO := eng.RenderObject(nestedBox)

	if boxRO.FirstChild() != spanRO {
		t.Errorf("expected Span RO as first child")
	}
	if spanRO.NextSibling() != nestedBoxRO {
		t.Errorf("expected nested Box RO as next sibling of Span RO")
	}
}
