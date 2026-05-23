package engine

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func TestEngine_EventCoalescing_MouseMove(t *testing.T) {
	b := mock.New(80, 24)
	e := New(b, Options{})
	defer e.Stop()

	count := 0
	div := element.NewBox(e.Document()).
		Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Height: style.Some(style.Percent(100)),
		})
	div.AddEventListener(event.EventMouseMove, func(ev event.Event) {
		count++
	})
	e.Document().AppendChild(div)

	// We need to trigger layout so hit testing works.
	e.Frame()

	// Push multiple mouse moves into the buffer.
	e.eventBuffer = append(e.eventBuffer,
		&event.RawMouseEvent{X: 1, Y: 1, Move: true},
		&event.RawMouseEvent{X: 2, Y: 2, Move: true},
		&event.RawMouseEvent{X: 3, Y: 3, Move: true},
	)

	e.drainEvents()

	if count != 1 {
		t.Errorf("expected 1 mousemove event, got %d", count)
	}
}

func TestEngine_EventCoalescing_WheelEvents(t *testing.T) {
	b := mock.New(80, 24)
	e := New(b, Options{})
	defer e.Stop()

	var lastDeltaY int
	count := 0
	div := element.NewBox(e.Document()).
		Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Height: style.Some(style.Percent(100)),
		})
	div.AddEventListener(event.EventWheel, func(ev event.Event) {
		we := ev.(*event.WheelEvent)
		lastDeltaY = we.DeltaY
		count++
	})
	e.Document().AppendChild(div)

	// Layout.
	e.Frame()

	// Push multiple wheel events.
	e.eventBuffer = append(e.eventBuffer,
		&event.RawMouseEvent{X: 1, Y: 1, DeltaY: 1},
		&event.RawMouseEvent{X: 1, Y: 1, DeltaY: 2},
		&event.RawMouseEvent{X: 1, Y: 1, DeltaY: 3},
	)

	e.drainEvents()

	if count != 1 {
		t.Errorf("expected 1 wheel event, got %d", count)
	}
	if lastDeltaY != 6 {
		t.Errorf("expected deltaY 6, got %d", lastDeltaY)
	}
}
