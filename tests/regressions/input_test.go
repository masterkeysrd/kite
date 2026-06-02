// Package regressions – InputElement regression tests (TSK-024).
//
// These tests verify the refactored <input> implementation that uses the UA
// Shadow Subtree (ADR-009), the Intrinsic Style Layer (ADR-010), and
// cursor.FromTextFragment (TSK-023). They exercise the full engine pipeline
// (sync → style → layout → paint) to ensure the element renders correctly.
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

// --- helpers -----------------------------------------------------------------

// inputDispatch fires a synthetic keydown event at the input element.
func inputDispatch(inp *element.InputElement, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

// --- TSK-024: UA subtree invisibility ----------------------------------------

// TestInput_Regression_PublicChildrenEmpty verifies that ChildNodes on the
// host InputElement yields no public children (the UA text node is hidden).
func TestInput_Regression_PublicChildrenEmpty(t *testing.T) {
	inp := element.Input("hello")
	count := 0
	for range inp.ChildNodes() {
		count++
	}
	if count != 0 {
		t.Errorf("ChildNodes count = %d, want 0 (UA text node must be invisible)", count)
	}
}

// --- TSK-024: IntrinsicStyle wins over author overrides ----------------------

// TestInput_Regression_IntrinsicStyleWins verifies that after an engine frame
// the computed style for the input reflects the UA-forced properties even when
// the author sets conflicting values. Display is NOT intrinsic and can be overridden.
func TestInput_Regression_IntrinsicStyleWins(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("hi")
	// Author attempts Display:Block — should now win over default InlineBlock.
	inp.Style(style.S().Display(style.DisplayBlock).OverflowX(style.OverflowVisible))

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	ro := eng.RenderObject(inp)
	if ro == nil {
		t.Fatal("no render object on input after Frame")
	}
	cs := ro.ComputedStyle()
	if cs == nil {
		t.Fatal("computed style is nil")
	}
	if cs.Display != style.DisplayBlock {
		t.Errorf("Display = %v, want DisplayBlock", cs.Display)
	}
	if cs.OverflowX != style.OverflowClip {
		t.Errorf("OverflowX = %v, want OverflowClip (intrinsic must win)", cs.OverflowX)
	}
	if cs.WhiteSpace != style.WhiteSpacePre {
		t.Errorf("WhiteSpace = %v, want WhiteSpacePre (intrinsic must win)", cs.WhiteSpace)
	}
}

// --- TSK-024: Typing produces correct buffer state ---------------------------

// TestInput_Regression_TypingUpdatesBuffer verifies that typing characters
// through key events updates the input value.
func TestInput_Regression_TypingUpdatesBuffer(t *testing.T) {
	inp := element.Input("")

	for _, ch := range "hello" {
		inputDispatch(inp, key.Key{Code: ch, Text: string(ch)})
	}

	if got := inp.Value(); got != "hello" {
		t.Errorf("Value() = %q after typing, want %q", got, "hello")
	}
}

// TestInput_Regression_BackspaceDeletes verifies backspace removes the last
// character.
func TestInput_Regression_BackspaceDeletes(t *testing.T) {
	inp := element.Input("hello")
	inputDispatch(inp, key.Key{Code: key.KeyBackspace})
	if got := inp.Value(); got != "hell" {
		t.Errorf("Value() = %q after backspace, want %q", got, "hell")
	}
}

// --- TSK-024: Focus navigation lands on host, not UA node -------------------

// TestInput_Regression_FocusLandsOnHost verifies that focus navigation
// (focus.Manager.Next) focuses the InputElement host, not a UA child.
func TestInput_Regression_FocusLandsOnHost(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame() // build render objects so IsFocusable check passes

	fm := focus.NewManager(eng.Document(), event.NewDispatcher())
	if !fm.Next() {
		t.Fatal("focus.Next() returned false — no focusable element found")
	}
	focused := fm.Current()
	if focused == nil {
		t.Fatal("Current() is nil after Next()")
	}
	if focused != inp {
		t.Errorf("focused element = %T, want *element.InputElement", focused)
	}
}

// --- TSK-024: Engine frame renders without error ----------------------------

// TestInput_Regression_EngineFrameWithText verifies that an input with text
// renders a frame without panicking.
func TestInput_Regression_EngineFrameWithText(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("hello world")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	fr := b.LastFrame()
	if fr.Surface == nil {
		t.Fatal("no surface produced after Frame")
	}
}

// TestInput_Regression_EngineFrameAfterTyping verifies that the engine can
// produce a frame after keyboard input is applied.
func TestInput_Regression_EngineFrameAfterTyping(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame() // initial frame

	// Type a few characters.
	for _, ch := range "test" {
		inputDispatch(inp, key.Key{Code: ch, Text: string(ch)})
	}

	eng.Frame() // second frame — must not panic

	fr := b.LastFrame()
	if fr.Surface == nil {
		t.Fatal("no surface produced after second Frame")
	}
}

// --- TSK-024: CursorState follows buffer offset ------------------------------

// TestInput_Regression_CursorState_EmptyBuffer verifies CursorState on an
// empty buffer (start and end positions are both (0,0)).
func TestInput_Regression_CursorState_EmptyBuffer(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	var p cursor.Provider = inp
	cs := p.CursorState()
	if !cs.Visible {
		t.Error("cursor should be visible")
	}
	// Empty buffer: cursor is at (0,0).
	if cs.X != 0 || cs.Y != 0 {
		t.Errorf("CursorState = (%d,%d), want (0,0) for empty buffer", cs.X, cs.Y)
	}
}

// TestInput_Regression_CursorState_AtEnd verifies that after typing, the
// cursor X is positive (past the last character).
func TestInput_Regression_CursorState_AtEnd(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("abc")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	var p cursor.Provider = inp
	cs := p.CursorState()
	if !cs.Visible {
		t.Error("cursor should be visible")
	}
	// Buffer starts at end of "abc" (3 bytes). Cursor X should be 3 (3 ASCII chars).
	if cs.X != 3 {
		t.Errorf("CursorState.X = %d, want 3 (end of 'abc')", cs.X)
	}
}

// --- Empty-input height regression (block_test empty-IFC fix) ----------------

// TestInput_Regression_EmptyBorderedInput_Height verifies end-to-end that an
// InputElement with a border and an empty buffer produces a fragment that is
// exactly 3 rows tall: top-border + 1 content row + bottom-border.
//
// This test would have caught the bug fixed in layout/block.go where an IFC
// with no visible text emitted zero line-boxes, causing currentBlockOffset to
// never advance past border.Top, collapsing the two border rows together:
//
//	╌╌╌╌╌╌╌╌╌╌  (broken: 2 rows, content row missing)
//	╌╌╌╌╌╌╌╌╌╌
//
//	┌────────┐  (correct: 3 rows)
//	│          │
//	└────────┘
func TestInput_Regression_EmptyBorderedInput_Height(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.S().Width(style.Cells(20)).Border(style.SingleBorder()))

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	ro := eng.RenderObject(inp)
	if ro == nil {
		t.Fatal("input has no render object after Frame")
	}
	frag := ro.Fragment()
	if frag == nil {
		t.Fatal("input render object has no fragment after Frame")
	}

	// border.Top(1) + content(1) + border.Bottom(1) = 3.
	const wantH = 3
	if frag.Size.Height != wantH {
		t.Errorf("empty bordered input: fragment height = %d, want %d\n"+
			"(regression: empty IFC was collapsing border rows together)",
			frag.Size.Height, wantH)
	}
	if frag.Size.Width != 20 {
		t.Errorf("empty bordered input: fragment width = %d, want 20", frag.Size.Width)
	}
}

// TestInput_Regression_EmptyUnborderedInput_Height verifies the no-border
// variant: an input without a border and an empty buffer must be exactly 1 row
// tall (the single reserved content row).
func TestInput_Regression_EmptyUnborderedInput_Height(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.S().Width(style.Cells(20)))

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	ro := eng.RenderObject(inp)
	if ro == nil {
		t.Fatal("input has no render object after Frame")
	}
	frag := ro.Fragment()
	if frag == nil {
		t.Fatal("input render object has no fragment after Frame")
	}

	const wantH = 1
	if frag.Size.Height != wantH {
		t.Errorf("empty unbordered input: fragment height = %d, want %d",
			frag.Size.Height, wantH)
	}
}

// TestInput_Regression_NonEmptyBorderedInput_Height verifies that a bordered
// input with text also produces height = 3 (the content row is now occupied by
// the text rather than the implicit reserve, but the total stays the same).
func TestInput_Regression_NonEmptyBorderedInput_Height(t *testing.T) {
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello")
	inp.Style(style.S().Width(style.Cells(20)).Border(style.SingleBorder()))

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	ro := eng.RenderObject(inp)
	if ro == nil {
		t.Fatal("input has no render object after Frame")
	}
	frag := ro.Fragment()
	if frag == nil {
		t.Fatal("input render object has no fragment after Frame")
	}

	// Same 3-row structure: the content row now holds the text line-box.
	const wantH = 3
	if frag.Size.Height != wantH {
		t.Errorf("non-empty bordered input: fragment height = %d, want %d",
			frag.Size.Height, wantH)
	}
}

// --- TSK-024: Engine-Cursor integration --------------------------------------

// TestInput_Regression_EngineCursorIntegration verifies that the engine
// correctly picks up the cursor state from the focused InputElement and
// applies it to the backend.
func TestInput_Regression_EngineCursorIntegration(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	// Frame calls updateHardwareCursor.
	inp.Focus()
	eng.Frame()

	if !b.Cursor.Visible {
		t.Errorf("expected cursor to be visible when input is focused")
	}
	// cursor.FromTextFragment for "hello" at end (offset 5) should be X=5, Y=0.
	if b.Cursor.X != 5 {
		t.Errorf("expected cursor X=5, got %d", b.Cursor.X)
	}
}

// --- ADR-012: Scroll-back on backspace ---------------------------------------

// TestInput_Regression_ScrollBackOnBackspace verifies that when the cursor is
// at the end of a scrolled input, deleting characters "pulls" the text back so
// the cursor stays at the end of the content box (no empty space).
func TestInput_Regression_ScrollBackOnBackspace(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Create an input with a fixed width of 10 cells.
	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.S().Width(style.Cells(10)).Padding(style.EdgeValues[int]{}).Border(style.Border{}))

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	// Focus the input.
	eng.Document().Focus(inp)

	// Type 15 characters into a 10-cell box.
	// scrollX should end at 6.
	for _, ch := range "123456789012345" {
		inputDispatch(inp, key.Key{Code: ch, Text: string(ch)})
		eng.Frame()
	}

	if gotX, _ := inp.Scroll(); gotX != 6 {
		t.Errorf("After typing 15 chars, scrollX = %d, want 6", gotX)
	}

	// Backspace once. Value length becomes 14.
	// If we didn't fix the bug, scrollX would stay at 6, and the cursor would
	// be at position 14-6 = 8.
	// With the fix, scrollX should become 14-9 = 5, keeping the cursor at 9.
	inputDispatch(inp, key.Key{Code: key.KeyBackspace})
	eng.Frame()

	if gotX, _ := inp.Scroll(); gotX != 5 {
		t.Errorf("After backspace, scrollX = %d, want 5 (to keep cursor at the end)", gotX)
	}
}
