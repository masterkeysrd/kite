package element_test

// Tests for TSK-029: Unified Text Control Base (textControlBase).
//
// These tests verify:
//   - ScrollCursorIntoView correctly updates Y-scroll for multi-line controls.
//   - ScrollCursorIntoView ignores Y-scroll for single-line controls.
//   - Clicking inside a heavily scrolled textarea maps the click to the correct
//     buffer byte offset using the generic hit-testing logic.

import (
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/text"
	"github.com/masterkeysrd/kite/key"
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
	// The text.Buffer places the cursor at the end by default, i.e. line 9.
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

	// Verify it is NOT immediate.
	_, scrollYBefore := txa.Scroll()
	if scrollYBefore != 0 {
		t.Errorf("scroll.Y = %d immediately after SyncBuffer, want 0 (must be deferred)", scrollYBefore)
	}

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
	ev := event.NewMouseEvent(event.EventMouseDown, geom.Point{X: 0, Y: 0}, event.ButtonLeft, 0)
	ev.Local = geom.Point{X: 0, Y: 0}
	d := event.NewDispatcher()
	d.Dispatch(ev, []event.EventTarget{txa})

	// "line0\n" = 6, "line1\n" = 6, "line2\n" = 6 → line3 starts at offset 18.
	wantOffset := 18
	if got := txa.Buffer().ByteOffset(); got != wantOffset {
		t.Errorf("ByteOffset after scrolled click = %d, want %d", got, wantOffset)
	}
}

// TestTextArea_Panic_DocumentMismatch verifies that adopting a TextArea into a
// new document does not cause a panic during selection updates due to
// mismatched owner documents on UA nodes.
func TestTextArea_Panic_DocumentMismatch(t *testing.T) {
	doc1 := dom.NewDocument()
	txa := element.NewTextArea(doc1, "Hello")

	// Move to doc2.
	doc2 := dom.NewDocument()
	doc2.AppendChild(txa)

	// Mutate buffer to trigger rebuild of UA subtree in doc2.
	txa.Buffer().Insert(" World")
	txa.SyncBuffer()

	// Trigger selection update. If it uses nodes from doc1, it will panic.
	txa.SetSelectionRange(0, 5)

	// If we reached here without panic, the fix is verified.
}

// TestTextArea_Panic_InvalidBROffset verifies that mapping a selection to a
// <br> element does not use an invalid offset (1), which would exceed the
// child count (0) of the void <br> element and cause a panic.
func TestTextArea_Panic_InvalidBROffset(t *testing.T) {
	doc := dom.NewDocument()
	txa := element.NewTextArea(doc, "Line One\nLine Two")
	root := element.Box(txa)
	doc.AppendChild(root)

	// Layout is needed for resolveOffset to work (it iterates ChildNodes,
	// but textControlBase.resolveOffset uses the uaDiv's children which are
	// created during rebuildUASubtree).

	// Offset 8 is the '\n' character.
	// We want to ensure that setting selection at or after this '\n' works.

	// Test setting selection exactly at the \n.
	txa.SetSelectionRange(8, 9)

	// Test setting selection starting exactly at the \n.
	txa.SetSelectionRange(8, 10)

	// If no panic, we are good.
}

func TestTextArea_Repro_StepByStep(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	content := "ABC\nDEF\nGHI"
	txa := element.TextArea(content)
	env.Mount(txa)
	env.Flush()

	// 1. Move to start of Line 2 ('D', offset 4).
	txa.SetSelectionRange(4, 4)
	env.Flush()

	sel := env.Document().Selection()

	// 2. Shift + Right. Should be "D".
	env.KeyPress("right", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "D" {
		t.Errorf("Step 1 (Shift+Right): expected 'D', got %q", got)
	}

	// 3. Shift + Right again. Should be "DE".
	env.KeyPress("right", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "DE" {
		t.Errorf("Step 2 (Shift+Right): expected 'DE', got %q", got)
	}

	// 4. Shift + Left. Should be back to "D".
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "D" {
		t.Errorf("Step 3 (Shift+Left): expected 'D', got %q", got)
	}

	// 5. Shift + Left. Should be empty (collapsed at 4).
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "" {
		t.Errorf("Step 4 (Shift+Left): expected '', got %q", got)
	}

	// 6. Shift + Left. Should be "\n" (backward selection from 4 to 3).
	env.KeyPress("left", key.ModShift)
	env.Flush()
	if got := sel.String(); got != "\n" {
		t.Errorf("Step 5 (Shift+Left): expected '\\n', got %q", got)
	}
}

func TestInput_KeyboardSelection(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	inp := element.Input("Hello World").WithID("inp")
	env.Mount(inp)
	env.Flush()

	// Focus the input
	env.Click(0, 0)
	env.KeyPress("end")
	env.Flush()

	// Initial caret at end (offset 11)
	// Shift + Left 5 times (selects "World")
	for i := 0; i < 5; i++ {
		env.KeyPress("left", key.ModShift)
	}
	env.Flush()

	doc := env.Document()
	sel := doc.Selection()

	if sel.RangeCount() != 1 {
		t.Fatalf("expected 1 range, got %d", sel.RangeCount())
	}

	if sel.String() != "World" {
		t.Errorf("expected selection 'World', got %q", sel.String())
	}

	// Type "Kite" to replace selection
	env.SendKey(key.Key{Code: 'K', Text: "K"})
	env.Flush()
	env.KeyPress("i")
	env.Flush()
	env.KeyPress("t")
	env.Flush()
	env.KeyPress("e")
	env.Flush()

	if inp.Value() != "Hello Kite" {
		t.Errorf("expected value 'Hello Kite', got %q", inp.Value())
	}

	if sel.RangeCount() != 0 {
		t.Errorf("expected selection to be cleared after typing, got %d", sel.RangeCount())
	}
}

func TestInput_MouseSelection(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	inp := element.Input("Hello World").WithID("inp")
	env.Mount(inp)
	env.Flush()

	// MouseDown on 'e' (offset 1)
	// Local coordinates for "Hello World":
	// H: 0, e: 1, l: 2, l: 3, o: 4,  : 5, W: 6, o: 7, r: 8, l: 9, d: 10
	env.MouseDown(1, 0, event.ButtonLeft)
	env.Flush()

	// Drag to 'o' (offset 7, after 'W')
	env.MouseMove(7, 0)
	env.Flush()

	// Depending on hit-testing, this might be "ello W" or "ello Wo"
	// Current implementation seems to give offset 8 for cell 7?
	// Let's see what we get.

	env.MouseUp(7, 0, event.ButtonLeft)
	env.Flush()

	// Backspace to delete selection
	env.KeyPress("backspace")
	env.Flush()

	if inp.Value() != "Horld" && inp.Value() != "Hoorld" {
		t.Errorf("expected value 'Horld' or 'Hoorld', got %q", inp.Value())
	}
}

func TestTextArea_Selection(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	txa := element.TextArea("Line One\nLine Two")
	env.Mount(txa)
	env.Flush()

	// Focus
	env.Click(0, 0)
	env.Flush()

	// Caret at end (offset 17)
	// Select All (Ctrl+A)
	env.KeyPress("a", key.ModCtrl)
	env.Flush()

	sel := env.Document().Selection()
	if sel.String() != "Line One\nLine Two" {
		t.Errorf("expected full selection, got %q", sel.String())
	}

	// Shift+Left to deselect one char
	env.KeyPress("left", key.ModShift)
	env.Flush()

	if sel.String() != "Line One\nLine Tw" {
		t.Errorf("expected partial selection, got %q", sel.String())
	}
}

func TestTextArea_SelectAllBugReproduction(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Text with multiple lines.
	content := "Line One\nLine Two\nLine Three"
	txa := element.TextArea(content)
	env.Mount(txa)
	env.Flush()

	// 1. Move to start of line 2 ('L' of "Line Two", offset 9)
	txa.Buffer().SetOffset(9)
	txa.SyncBuffer()
	env.Flush()

	// 2. Press Shift + Right. Should select only 'L'.
	env.KeyPress("right", key.ModShift)
	env.Flush()

	sel := env.Document().Selection()
	if sel.String() != "L" {
		t.Errorf("expected selection 'L', got %q", sel.String())
	}

	// 3. Move back to offset 9 and try Shift + Left.
	txa.SetSelectionRange(9, 9)
	env.Flush()

	env.KeyPress("left", key.ModShift)
	env.Flush()

	// Should select the newline character before 'L'.
	if sel.String() != "\n" {
		t.Errorf("expected selection '\\n', got %q", sel.String())
	}
}

func BenchmarkTextArea_UpdateSelectionRange(b *testing.B) {
	doc := dom.NewDocument()
	// Large textarea content to make mapping expensive.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "This is line number "+strings.Repeat("x", 20))
	}
	content := strings.Join(lines, "\n")
	txa := element.NewTextArea(doc, content)
	doc.AppendChild(txa)

	// Ensure UA subtree is built.
	txa.SyncBuffer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set selection at the end.
		txa.SetSelectionRange(len(content)-10, len(content))
	}
}

func BenchmarkBuffer_DeleteRange(b *testing.B) {
	content := strings.Repeat("hello world ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := text.NewBuffer(content)
		buf.DeleteRange(100, 200)
	}
}
