package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestListComponents_Defaults(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	// 1. UnorderedList
	ul := element.NewUnorderedList(doc)
	if ul.TagName() != "ul" {
		t.Errorf("expected tag ul, got %s", ul.TagName())
	}
	ulDef := ul.ElementDefaultStyle()
	if ulDef.Display.Value() != style.DisplayBlock {
		t.Errorf("UL default display should be Block")
	}
	if ulDef.ListStyleType.Value() != style.ListStyleDisc {
		t.Errorf("UL default ListStyleType should be Disc")
	}
	if ulDef.Padding.Value().Left != 2 {
		t.Errorf("UL default Padding.Left should be 2")
	}

	// 2. OrderedList
	ol := element.NewOrderedList(doc)
	if ol.TagName() != "ol" {
		t.Errorf("expected tag ol, got %s", ol.TagName())
	}
	olDef := ol.ElementDefaultStyle()
	if olDef.Display.Value() != style.DisplayBlock {
		t.Errorf("OL default display should be Block")
	}
	if olDef.ListStyleType.Value() != style.ListStyleDecimal {
		t.Errorf("OL default ListStyleType should be Decimal")
	}
	if olDef.Padding.Value().Left != 3 {
		t.Errorf("OL default Padding.Left should be 3")
	}

	// 3. ListItem
	li := element.NewListItem(doc)
	if li.TagName() != "li" {
		t.Errorf("expected tag li, got %s", li.TagName())
	}
	liDef := li.ElementDefaultStyle()
	if liDef.Display.Value() != style.DisplayListItem {
		t.Errorf("LI default display should be ListItem")
	}
}

func TestListComponents_Inheritance(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	ol := element.NewOrderedList(doc)
	li1 := element.NewListItem(doc)
	li2 := element.NewListItem(doc)

	ol.AppendChild(li1)
	ol.AppendChild(li2)
	doc.AppendChild(ol)

	// Resolve styles
	eng.Frame()

	// Verify li1 and li2 inherited Decimal from ol
	if li1.RenderObject().ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("li1 should inherit Decimal, got %v", li1.RenderObject().ComputedStyle().ListStyleType)
	}
	if li2.RenderObject().ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("li2 should inherit Decimal, got %v", li2.RenderObject().ComputedStyle().ListStyleType)
	}

	// Change ol to Square
	ol.Style(style.Style{
		ListStyleType: style.Some(style.ListStyleSquare),
	})

	// Resolve styles again
	eng.Frame()

	if li1.RenderObject().ComputedStyle().ListStyleType != style.ListStyleSquare {
		t.Errorf("li1 should now be Square, got %v", li1.RenderObject().ComputedStyle().ListStyleType)
	}
}

func TestListComponents_NestedInheritance(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	ul := element.NewUnorderedList(doc)
	li := element.NewListItem(doc)
	innerOl := element.NewOrderedList(doc)
	innerLi := element.NewListItem(doc)

	ul.AppendChild(li)
	li.AppendChild(innerOl)
	innerOl.AppendChild(innerLi)
	doc.AppendChild(ul)

	eng.Frame()

	if li.RenderObject().ComputedStyle().ListStyleType != style.ListStyleDisc {
		t.Errorf("li should be Disc")
	}
	if innerLi.RenderObject().ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("innerLi should be Decimal, overriding inherited Disc")
	}
}
