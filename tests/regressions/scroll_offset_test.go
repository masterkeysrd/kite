package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func TestScroll_PreservedAcrossMoves(t *testing.T) {
	doc := dom.NewDocument()
	parent1 := doc.CreateElement("div", nil)
	parent2 := doc.CreateElement("div", nil)
	el := doc.CreateElement("div", nil)

	parent1.AppendChild(el)
	el.ScrollTo(10, 20)

	// Move to another parent
	parent1.RemoveChild(el)
	parent2.AppendChild(el)

	x, y := el.Scroll()
	if x != 10 || y != 20 {
		t.Errorf("Scroll not preserved after move: got (%d, %d), want (10, 20)", x, y)
	}
}

func TestScroll_PreservedAcrossOverflowToggle(t *testing.T) {
	doc := dom.NewDocument()
	// Using element package for easier style setting
	el := element.Box().Style(style.Style{}.Overflow(style.OverflowScroll))
	doc.AppendChild(el.Unwrap().(dom.Element))

	el.Unwrap().(dom.Element).ScrollTo(10, 20)

	// Toggle overflow to visible (non-container)
	el.Style(style.Style{}.Overflow(style.OverflowVisible))

	x, y := el.Unwrap().(dom.Element).Scroll()
	if x != 10 || y != 20 {
		t.Errorf("Scroll not preserved after overflow toggle: got (%d, %d), want (10, 20)", x, y)
	}
}

func TestScroll_EventBubbling(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	child := doc.CreateElement("div", nil)
	parent.AppendChild(child)

	var received bool
	parent.AddEventListener(event.EventScroll, func(e event.Event) {
		received = true
	})

	child.ScrollTo(1, 1)

	if !received {
		t.Error("ScrollEvent did not bubble to parent")
	}
}
