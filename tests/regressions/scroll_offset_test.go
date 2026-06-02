package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
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

	testenv.Expect(t, el).ToHaveScroll(10, 20)
}

func TestScroll_PreservedAcrossOverflowToggle(t *testing.T) {
	doc := dom.NewDocument()
	// Using element package for easier style setting
	el := element.Box().Style(style.Style{}.Overflow(style.OverflowScroll))
	doc.AppendChild(el.Unwrap().(dom.Element))

	el.Unwrap().(dom.Element).ScrollTo(10, 20)

	// Toggle overflow to visible (non-container)
	el.Style(style.Style{}.Overflow(style.OverflowVisible))

	testenv.Expect(t, el.Unwrap().(dom.Element)).ToHaveScroll(10, 20)
}

func TestScroll_EventBubbling(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("div", nil)
	child := doc.CreateElement("div", nil)
	parent.AppendChild(child)

	testenv.ExpectEvent(t, parent, event.EventScroll).ToFireWhen(func() {
		child.ScrollTo(1, 1)
	})
}
