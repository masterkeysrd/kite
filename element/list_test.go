package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestListComponents_Defaults(t *testing.T) {
	// Using declarative API
	ul := element.UL()
	if ul.TagName() != "ul" {
		t.Errorf("expected tag ul, got %s", ul.TagName())
	}
	ulDef := ul.DefaultStyle()
	if ulDef.DisplayOpt().Value() != style.DisplayBlock {
		t.Errorf("UL default display should be Block")
	}
	if ulDef.ListStyleTypeOpt().Value() != style.ListStyleDisc {
		t.Errorf("UL default ListStyleType should be Disc")
	}
	if ulDef.PaddingOpt().Value().Left != 2 {
		t.Errorf("UL default Padding.Left should be 2")
	}

	ol := element.OL()
	if ol.TagName() != "ol" {
		t.Errorf("expected tag ol, got %s", ol.TagName())
	}
	olDef := ol.DefaultStyle()
	if olDef.DisplayOpt().Value() != style.DisplayBlock {
		t.Errorf("OL default display should be Block")
	}
	if olDef.ListStyleTypeOpt().Value() != style.ListStyleDecimal {
		t.Errorf("OL default ListStyleType should be Decimal")
	}
	if olDef.PaddingOpt().Value().Left != 3 {
		t.Errorf("OL default Padding.Left should be 3")
	}

	li := element.LI()
	if li.TagName() != "li" {
		t.Errorf("expected tag li, got %s", li.TagName())
	}
	liDef := li.DefaultStyle()
	if liDef.DisplayOpt().Value() != style.DisplayListItem {
		t.Errorf("LI default display should be ListItem")
	}
}

func TestListComponents_Inheritance(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	li1 := element.LI()
	li2 := element.LI()
	ol := element.OL(li1, li2)
	eng.Mount(ol)

	// Resolve styles
	eng.Frame()

	// Verify li1 and li2 inherited Decimal from ol
	if eng.RenderObject(li1).ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("li1 should inherit Decimal, got %v", eng.RenderObject(li1).ComputedStyle().ListStyleType)
	}
	if eng.RenderObject(li2).ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("li2 should inherit Decimal, got %v", eng.RenderObject(li2).ComputedStyle().ListStyleType)
	}

	// Change ol to Square
	ol.Style(style.S().ListStyleType(style.ListStyleSquare))

	// Resolve styles again
	eng.Frame()

	if eng.RenderObject(li1).ComputedStyle().ListStyleType != style.ListStyleSquare {
		t.Errorf("li1 should now be Square, got %v", eng.RenderObject(li1).ComputedStyle().ListStyleType)
	}
}

func TestListComponents_NestedInheritance(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	innerLi := element.LI()
	li := element.LI(
		element.OL(innerLi),
	)
	ul := element.UL(li)
	eng.Mount(ul)

	eng.Frame()

	if eng.RenderObject(li).ComputedStyle().ListStyleType != style.ListStyleDisc {
		t.Errorf("li should be Disc")
	}
	if eng.RenderObject(innerLi).ComputedStyle().ListStyleType != style.ListStyleDecimal {
		t.Errorf("innerLi should be Decimal, overriding inherited Disc")
	}
}
