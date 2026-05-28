package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/style"
)

func TestRegression_DynamicListStyleUpdate(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	doc := eng.Document()

	ul := element.NewUnorderedList(doc)
	li := element.NewListItem(doc)
	ul.AppendChild(li)
	doc.AppendChild(ul)

	// First frame: should be Disc (default)
	eng.Frame()
	if eng.RenderObject(li).ComputedStyle().ListStyleType != style.ListStyleDisc {
		t.Fatalf("expected initial Disc, got %v", eng.RenderObject(li).ComputedStyle().ListStyleType)
	}

	// Update UL style
	ul.Style(style.Style{
		ListStyleType: style.Some(style.ListStyleSquare),
	})

	// Second frame: should resolve to Square
	eng.Frame()
	if eng.RenderObject(li).ComputedStyle().ListStyleType != style.ListStyleSquare {
		t.Fatalf("expected updated Square, got %v", eng.RenderObject(li).ComputedStyle().ListStyleType)
	}
}
