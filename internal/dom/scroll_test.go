package dom

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
)

func TestElement_Scroll(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div", nil)

	// Initial state
	x, y := el.Scroll()
	if x != 0 || y != 0 {
		t.Errorf("Initial scroll: got (%d, %d), want (0, 0)", x, y)
	}

	// ScrollTo
	el.ScrollTo(10, 20)
	x, y = el.Scroll()
	if x != 10 || y != 20 {
		t.Errorf("After ScrollTo(10, 20): got (%d, %d), want (10, 20)", x, y)
	}

	// ScrollBy
	el.ScrollBy(5, -5)
	x, y = el.Scroll()
	if x != 15 || y != 15 {
		t.Errorf("After ScrollBy(5, -5): got (%d, %d), want (15, 15)", x, y)
	}
}

func TestElement_ScrollEvent(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div", nil)

	var received *event.ScrollEvent
	el.AddEventListener(event.EventScroll, func(e event.Event) {
		received = e.(*event.ScrollEvent)
	})

	el.ScrollTo(5, 10)

	if received == nil {
		t.Fatal("ScrollEvent not received")
	}
	if received.X != 5 || received.Y != 10 {
		t.Errorf("ScrollEvent offset: got (%d, %d), want (5, 10)", received.X, received.Y)
	}
	if received.DeltaX != 5 || received.DeltaY != 10 {
		t.Errorf("ScrollEvent delta: got (%d, %d), want (5, 10)", received.DeltaX, received.DeltaY)
	}

	el.ScrollBy(1, 2)
	if received.X != 6 || received.Y != 12 {
		t.Errorf("ScrollEvent offset after ScrollBy: got (%d, %d), want (6, 12)", received.X, received.Y)
	}
	if received.DeltaX != 1 || received.DeltaY != 2 {
		t.Errorf("ScrollEvent delta after ScrollBy: got (%d, %d), want (1, 2)", received.DeltaX, received.DeltaY)
	}
}

type mockScrollClampingView struct {
	mockView
	maxSX, maxSY int
}

func (m *mockScrollClampingView) GetMaxScroll(n dom.Node) (x, y int) {
	return m.maxSX, m.maxSY
}

func TestElement_ScrollClamping(t *testing.T) {
	doc := NewDocument()
	el := doc.CreateElement("div", nil)

	// Set a mock view on the document with max scroll limits
	view := &mockScrollClampingView{maxSX: 50, maxSY: 100}
	doc.SetDefaultView(view)

	// Programmatic ScrollTo exceeding boundaries
	el.ScrollTo(9999, 9999)
	x, y := el.Scroll()
	if x != 50 || y != 100 {
		t.Errorf("ScrollTo(9999, 9999) did not clamp to view bounds: got (%d, %d), want (50, 100)", x, y)
	}

	// Programmatic ScrollTo below zero
	el.ScrollTo(-10, -20)
	x, y = el.Scroll()
	if x != 0 || y != 0 {
		t.Errorf("ScrollTo(-10, -20) did not clamp to zero: got (%d, %d), want (0, 0)", x, y)
	}

	// Programmatic ScrollBy exceeding bounds
	el.ScrollTo(40, 90)
	el.ScrollBy(20, 20)
	x, y = el.Scroll()
	if x != 50 || y != 100 {
		t.Errorf("ScrollBy(20, 20) did not clamp: got (%d, %d), want (50, 100)", x, y)
	}

	// Programmatic ScrollBy below zero
	el.ScrollBy(-60, -110)
	x, y = el.Scroll()
	if x != 0 || y != 0 {
		t.Errorf("ScrollBy(-60, -110) did not clamp: got (%d, %d), want (0, 0)", x, y)
	}
}
