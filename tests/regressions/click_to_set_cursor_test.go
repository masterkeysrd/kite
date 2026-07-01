package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

func TestInput_ClickToSetCursor(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello world")
	eng.Document().AppendChild(inp)

	// Layout and Paint
	eng.Frame()

	// Click at the start (offset 0)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := inp.Buffer().ByteOffset(); off != 0 {
		t.Errorf("Click at (0,0) expected offset 0, got %d", off)
	}

	// Click on 'e' (offset 1)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 1, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := inp.Buffer().ByteOffset(); off != 1 {
		t.Errorf("Click at (1,0) expected offset 1, got %d", off)
	}

	// Click on ' ' (offset 5)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 5, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := inp.Buffer().ByteOffset(); off != 5 {
		t.Errorf("Click at (5,0) expected offset 5, got %d", off)
	}

	// Click past the end of text but inside element
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 15, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := inp.Buffer().ByteOffset(); off != 11 {
		t.Errorf("Click at (15,0) expected offset 11, got %d", off)
	}
}

func TestTextArea_ClickToSetCursor(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "line1\nline2")
	eng.Document().AppendChild(txa)

	// Layout and Paint
	eng.Frame()

	// Click on line 1, 'l' (offset 0)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := txa.Buffer().ByteOffset(); off != 0 {
		t.Errorf("Click at (0,0) expected offset 0, got %d", off)
	}

	// Click on line 1, 'i' (offset 1)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 1, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := txa.Buffer().ByteOffset(); off != 1 {
		t.Errorf("Click at (1,0) expected offset 1, got %d", off)
	}

	// Click on line 2, 'l' (offset 6)
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 0, Y: 1, Button: event.ButtonLeft, Up: false})
	if off := txa.Buffer().ByteOffset(); off != 6 {
		t.Errorf("Click at (0,1) expected offset 6, got %d", off)
	}

	// Click past the end of line 2
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 10, Y: 1, Button: event.ButtonLeft, Up: false})
	if off := txa.Buffer().ByteOffset(); off != 11 {
		t.Errorf("Click at (10,1) expected offset 11, got %d", off)
	}

	// Click on a third (empty) line - should be clamped to end of buffer
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 0, Y: 2, Button: event.ButtonLeft, Up: false})
	if off := txa.Buffer().ByteOffset(); off != 11 {
		t.Errorf("Click at (0,2) expected offset 11, got %d", off)
	}
}

func TestOverlay_ClickToSetCursor(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})
	defer eng.Stop()

	// 1. Create a main background input
	bgInp := element.NewInput(eng.Document(), "background text")
	bgInp.Style(style.S().
		Width(style.Cells(20)).
		Height(style.Cells(1)))
	eng.Document().AppendChild(bgInp)

	// 2. Create an overlay input and wrap it in Overlay
	overlayInp := element.NewInput(eng.Document(), "overlay text")
	overlayInp.Style(style.S().
		Width(style.Cells(20)).
		Height(style.Cells(1)))

	overlay := element.Overlay(
		overlayInp,
		element.OverlayConfig{
			Anchor:    bgInp,
			Placement: geom.PlacementBottom,
			ZIndex:    100,
		},
	)

	eng.Document().AppendChild(overlay)

	// Layout and Paint
	eng.Frame()

	// Get overlay bounds to ensure we click inside it
	overlayRect, _ := overlayInp.GetBoundingClientRect()
	ox := overlayRect.Origin.X
	oy := overlayRect.Origin.Y

	// Click at (0, 0) which is on the background input
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft, Up: false})
	if off := bgInp.Buffer().ByteOffset(); off != 0 {
		t.Errorf("Click on bg expected offset 0, got %d", off)
	}

	// Click at (ox + 1, oy) which is on the overlay input (offset 1 'v')
	eng.ProcessRawEvent(&backend.RawMouseEvent{X: ox + 1, Y: oy, Button: event.ButtonLeft, Up: false})

	if off := overlayInp.Buffer().ByteOffset(); off != 1 {
		t.Errorf("Click on overlay expected offset 1, got %d", off)
	}
}
