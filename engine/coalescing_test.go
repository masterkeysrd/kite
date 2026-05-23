package engine

import (
	"strings"
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

func BenchmarkEngine_HitTest_TextArea(b *testing.B) {
	be := mock.New(80, 24)
	e := New(be, Options{})
	defer e.Stop()

	// 500 lines.
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString("This is a reasonably long line of text for testing hit testing performance\n")
	}
	txa := element.NewTextArea(e.Document(), sb.String())
	e.Document().AppendChild(txa)
	e.Frame() // Layout

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Hit test at the very last line to force walking all fragments if the implementation is linear.
		e.HitTest(10, 20)
	}
}

func BenchmarkEngine_ScrollBurst_Coalesced(b *testing.B) {
	be := mock.New(80, 24)
	e := New(be, Options{})
	defer e.Stop()

	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Line content\n")
	}
	txa := element.NewTextArea(e.Document(), sb.String())
	e.Document().AppendChild(txa)
	e.Frame() // Initial layout

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 500 wheel ticks arriving before the 8ms ticker fires.
		for j := 0; j < 500; j++ {
			e.eventBuffer = append(e.eventBuffer, &event.RawMouseEvent{
				X: 10, Y: 10, DeltaY: 1,
			})
		}
		e.drainEvents() // Coalesces 500 events into 1
		e.Frame()       // Process the 1 coalesced event
	}
}

func BenchmarkEngine_ScrollBurst_NonCoalesced(b *testing.B) {
	be := mock.New(80, 24)
	e := New(be, Options{})
	defer e.Stop()

	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Line content\n")
	}
	txa := element.NewTextArea(e.Document(), sb.String())
	e.Document().AppendChild(txa)
	e.Frame() // Initial layout

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate 500 wheel ticks, each triggering a frame (no coalescing).
		for j := 0; j < 500; j++ {
			e.eventBuffer = append(e.eventBuffer, &event.RawMouseEvent{
				X: 10, Y: 10, DeltaY: 1,
			})
			e.drainEvents()
			e.Frame()
		}
	}
}
