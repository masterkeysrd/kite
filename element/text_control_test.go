package element_test

// Tests for TSK-029: Unified Text Control Base (textControlBase).
//
// These tests verify:
//   - ScrollCursorIntoView correctly updates Y-scroll for multi-line controls.
//   - ScrollCursorIntoView ignores Y-scroll for single-line controls.
//   - Clicking inside a heavily scrolled textarea maps the click to the correct
//     buffer byte offset using the generic hit-testing logic.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// ScrollCursorIntoView: multiline (TextArea)
// ---------------------------------------------------------------------------

// TestTextControlBase_ScrollCursorIntoView_UpdatesYScroll verifies that when
// the cursor is below the visible viewport of a textarea, ScrollCursorIntoView
// updates the Y-scroll so the cursor becomes visible.
//
// Setup:
//  1. Create a textarea with 10 lines of text and a 3-row viewport.
//  2. Run a frame to produce the layout fragment.
//  3. Trigger needsScrollIntoView by calling SyncBuffer (cursor is at end).
//  4. Run another frame to invoke the engine's auto-scroll phase.
func TestTextControlBase_ScrollCursorIntoView_UpdatesYScroll(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// 10 lines of text in a 3-row-visible textarea.
	// The editor.Buffer places the cursor at the end by default, i.e. line 9.
	text := "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9"
	txa := element.NewTextArea(eng.Document(), text)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(3)), // only 3 rows visible
	})
	root := element.Box(txa)
	eng.Mount(root)

	// First frame: render tree is built; auto-focus selects txa.
	eng.Frame()

	// Trigger the needsScrollIntoView flag so the next frame's auto-scroll
	// phase has work to do. SyncBuffer mirrors the flag-setting contract used
	// by keyboard input handlers.
	txa.SyncBuffer()
	eng.Frame()

	_, scrollY := txa.Scroll()
	if scrollY <= 0 {
		t.Errorf("scroll.Y = %d after cursor at line 9 with 3-row viewport, want > 0", scrollY)
	}
}

// TestTextControlBase_ScrollCursorIntoView_ScrollsUpWhenCursorAboveView verifies
// that if the cursor moves above the current scroll offset, Y-scroll is reduced
// to bring the cursor back into view.
func TestTextControlBase_ScrollCursorIntoView_ScrollsUpWhenCursorAboveView(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	text := "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9"
	txa := element.NewTextArea(eng.Document(), text)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(3)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Trigger scroll to the end (cursor is already at line 9).
	txa.SyncBuffer()
	eng.Frame()

	// Pre-condition: after scrolling to end, scroll.Y should be > 0.
	_, scrollY := txa.Scroll()
	if scrollY <= 0 {
		t.Fatalf("pre-condition: scroll.Y = %d, want > 0 (cursor at line 9)", scrollY)
	}

	// Move cursor to line 0 and trigger scroll-into-view.
	txa.Buffer().SetOffset(0)
	txa.SyncBuffer()
	eng.Frame()

	_, scrollY = txa.Scroll()
	if scrollY != 0 {
		t.Errorf("scroll.Y = %d after cursor at line 0, want 0", scrollY)
	}
}

// ---------------------------------------------------------------------------
// ScrollCursorIntoView: single-line (Input) — Y-scroll must remain 0
// ---------------------------------------------------------------------------

// TestTextControlBase_ScrollCursorIntoView_InputIgnoresYScroll verifies that
// for a single-line Input, ScrollCursorIntoView never sets a non-zero Y-scroll
// even when text overflows horizontally.
func TestTextControlBase_ScrollCursorIntoView_InputIgnoresYScroll(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.Style{
		Width: style.Some(style.Cells(5)),
	})
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	// Type more characters than the visible width to trigger X-scrolling.
	for _, ch := range "123456789012345" {
		d := event.NewDispatcher()
		ev := event.NewKeyEvent(event.EventKeyDown, key.Key{Code: ch, Text: string(ch)})
		d.Dispatch(ev, []event.EventTarget{inp})
	}
	eng.Frame()

	_, scrollY := inp.Scroll()
	if scrollY != 0 {
		t.Errorf("Input scroll.Y = %d, want 0 (single-line must never Y-scroll)", scrollY)
	}

	// X should have scrolled to keep the cursor in view.
	scrollX, _ := inp.Scroll()
	if scrollX <= 0 {
		t.Errorf("Input scroll.X = %d, want > 0 (text overflow must X-scroll)", scrollX)
	}
}

// ---------------------------------------------------------------------------
// Integration: click-to-offset in a heavily scrolled textarea
// ---------------------------------------------------------------------------

// TestTextControlBase_MouseDown_HitTest_AfterScroll verifies that clicking
// inside a scrolled textarea correctly maps the screen coordinate to the
// expected buffer byte offset using the generic textControlBase hit-testing.
//
// Scenario:
//   - Textarea with 5 lines, visible height = 2 rows (3 lines hidden).
//   - Scroll down so that line 3 is at the top of the visible viewport.
//   - Click at local (0, 0) — which with scrollY=3 maps to buffer row 3,
//     column 0, i.e. the start of "line3" at byte offset 18.
func TestTextControlBase_MouseDown_HitTest_AfterScroll(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// "line0\nline1\nline2\nline3\nline4"
	// line0 → offset 0–4  (5 chars + \n = 6)
	// line1 → offset 6–10 (5 chars + \n = 6)
	// line2 → offset 12–16
	// line3 → offset 18–22
	// line4 → offset 24–28
	text := "line0\nline1\nline2\nline3\nline4"
	txa := element.NewTextArea(eng.Document(), text)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(2)), // only 2 rows visible
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Scroll to row 3 programmatically.
	txa.ScrollTo(0, 3)
	eng.Frame()

	// Click at local (0, 0) — which, after applying scrollY=3, maps to
	// buffer row 3, column 0, i.e. the start of "line3" at byte offset 18.
	ev := event.NewMouseEvent(event.EventMouseDown, layout.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	ev.Local = layout.Point{X: 0, Y: 0}
	d := event.NewDispatcher()
	d.Dispatch(ev, []event.EventTarget{txa})

	// "line0\n" = 6, "line1\n" = 6, "line2\n" = 6 → line3 starts at offset 18.
	wantOffset := 18
	if got := txa.Buffer().ByteOffset(); got != wantOffset {
		t.Errorf("ByteOffset after scrolled click = %d, want %d", got, wantOffset)
	}
}
