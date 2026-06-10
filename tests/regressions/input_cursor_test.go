package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

// TestRegression_InputCursorPosition verifies that the hardware cursor is placed
// correctly inside the input field's content box, accounting for border and padding.
func TestRegression_InputCursorPosition(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello")
	inp.Style(style.S().Width(style.Cells(20)).Border(style.SingleBorder()).Padding(0, 2))

	eng.Mount(inp)
	eng.Frame()

	// Initial focus (autofocus) should land on 'inp'.
	// Origin should be (0,0) as it's the root.
	// Border: Top=1, Left=1. Padding: Left=2.
	// Content start X = 1 (border) + 2 (padding) = 3.
	// Content start Y = 1 (border) + 0 (padding) = 1.

	// Ensure we are at the start of the buffer.
	inp.Buffer().MoveToStart()
	inp.SyncBuffer()
	eng.Frame()

	// "hello" at offset 0 should be at (3, 1).
	if b.Cursor.X != 3 || b.Cursor.Y != 1 {
		t.Errorf("cursor position with 'hello' at offset 0: got (%d,%d), want (3,1)", b.Cursor.X, b.Cursor.Y)
	}

	// "hello" at offset 5 should be at (3+5, 1).
	inp.Buffer().MoveToEnd()
	inp.SyncBuffer()
	eng.Frame()

	if b.Cursor.X != 8 || b.Cursor.Y != 1 { // 3 + 5 = 8
		t.Errorf("cursor position with 'hello' at offset 5: got (%d,%d), want (8,1)", b.Cursor.X, b.Cursor.Y)
	}
}

// TestRegression_EmptyInputCursorPosition verifies that the cursor is placed
// inside the content box even when the input is empty.
func TestRegression_EmptyInputCursorPosition(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	inp.Style(style.S().Width(style.Cells(20)).Border(style.SingleBorder()).Padding(0, 1))

	eng.Mount(inp)
	eng.Frame()

	// Content start X = 1 (border) + 1 (padding) = 2.
	// Content start Y = 1 (border) + 0 (padding) = 1.
	if b.Cursor.X != 2 || b.Cursor.Y != 1 {
		t.Errorf("empty input cursor position: got (%d,%d), want (2,1)", b.Cursor.X, b.Cursor.Y)
	}
}

// TestRegression_InputDoubleSpaceCursorNotLocked verifies that typing two
// consecutive spaces does not cause the cursor to jump to position (0,0) or
// become locked (unable to move left/right).
//
// Root cause: CSS whitespace collapsing in collectText would collapse the
// second space into nothing, producing a text fragment with fewer bytes than
// the buffer's byteOffset. cursor.FromTextFragment then returned (0,0,false)
// because byteOffset exceeded the total fragment byte count.
func TestRegression_InputDoubleSpaceCursorNotLocked(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	eng.Mount(inp)
	eng.Frame()

	// Type "ab  " (a, b, space, space).
	for _, ch := range "ab" {
		inputDispatch(inp, key.Key{Code: ch, Text: string(ch)})
	}
	inputDispatch(inp, key.Key{Code: ' ', Text: " "})
	inputDispatch(inp, key.Key{Code: ' ', Text: " "})

	eng.Frame()

	// Buffer: "ab  ", byteOffset=4. Cursor must NOT be at (0,0).
	var p cursor.Provider = inp
	cs := p.CursorState()
	if cs.X == 0 && cs.Y == 0 {
		t.Errorf("cursor jumped to (0,0) after double space — cursor is locked")
	}
	// Cursor should be at X=4 (2 chars + 2 spaces, each 1 cell wide).
	if cs.X != 4 {
		t.Errorf("cursor X after \"ab  \": got %d, want 4", cs.X)
	}

	// Pressing left should move the cursor (not stay locked).
	inputDispatch(inp, key.Key{Code: key.KeyLeft})
	eng.Frame()
	cs2 := p.CursorState()
	if cs2.X != 3 {
		t.Errorf("cursor X after left from \"ab  \": got %d, want 3", cs2.X)
	}
}

// TestRegression_InputTrailingSpaceBackspace verifies that pressing backspace
// at the end of text ending in spaces can continue deleting into non-space
// characters without getting stuck.
//
// Root cause: same whitespace-collapsing byte mismatch as above. The fragment
// byte count was less than the buffer byteOffset, causing cursor.FromTextFragment
// to return (0,0,false) and subsequent backspace operations to be misrouted.
func TestRegression_InputTrailingSpaceBackspace(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello  ")
	eng.Mount(inp)
	eng.Frame()

	// Delete trailing spaces one by one, then continue into the word.
	inputDispatch(inp, key.Key{Code: key.KeyBackspace})
	if got := inp.Value(); got != "hello " {
		t.Errorf("after 1st backspace: value = %q, want \"hello \"", got)
	}
	eng.Frame()

	inputDispatch(inp, key.Key{Code: key.KeyBackspace})
	if got := inp.Value(); got != "hello" {
		t.Errorf("after 2nd backspace: value = %q, want \"hello\"", got)
	}
	eng.Frame()

	// Continue deleting into the word — must not be stuck.
	inputDispatch(inp, key.Key{Code: key.KeyBackspace})
	if got := inp.Value(); got != "hell" {
		t.Errorf("after 3rd backspace: value = %q, want \"hell\"", got)
	}

	_ = b // backend present to run engine frames above
}

// TestRegression_InputSpaceCursorAdvancesWithEchoLabel reproduces the shaper
// cache-poisoning bug: when a Normal-whitespace label (the echo span in the
// example app) shapes a string containing double spaces, it used to mutate the
// shared cached []Cluster by zeroing CellWidth on the second space. The same
// cached entry was then returned for the Pre-whitespace input text node,
// making subsequent spaces in the input appear invisible to
// cursor.FromTextFragment and causing the hardware cursor to stall.
func TestRegression_InputSpaceCursorAdvancesWithEchoLabel(t *testing.T) {
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// The echo label uses WhiteSpaceNormal (the default). It mirrors the input
	// value, so once the input has "f  " the label will contain "f  " too.
	// Under the old bug, shaping "  " in Normal mode zeroed the cached CellWidth
	// of the second space, poisoning every subsequent Shape(" ") call.
	echoLabel := element.Text("")

	inp := element.NewInput(eng.Document(), "")
	root := element.Box(
		inp,
		element.Span(echoLabel),
	)
	eng.Mount(root)
	eng.Frame()

	// Type "f" then two spaces — mirrors the user's reproduction.
	inputDispatch(inp, key.Key{Code: 'f', Text: "f"})
	inputDispatch(inp, key.Key{Code: ' ', Text: " "})
	inputDispatch(inp, key.Key{Code: ' ', Text: " "})

	// Update echo to trigger Normal-mode collapsing of "f  " in the same frame.
	echoLabel.SetData(inp.Value().(string))
	eng.Frame()

	// The input buffer is "f  " (3 bytes). The cursor must be at X=3 relative
	// to the ua-div (3 cells from left: 'f' at 0, space at 1, space at 2).
	var p cursor.Provider = inp
	cs := p.CursorState()
	// insetLeft = 0 (no border/padding on this input), so X == cx == 3.
	if cs.X != 3 {
		t.Errorf("cursor X after \"f  \": got %d, want 3 (cache poisoning caused second space to be invisible)", cs.X)
	}

	// Press left twice: cursor must reach X=1 (before first space), not X=2.
	inputDispatch(inp, key.Key{Code: key.KeyLeft})
	inputDispatch(inp, key.Key{Code: key.KeyLeft})
	eng.Frame()
	cs2 := p.CursorState()
	if cs2.X != 1 {
		t.Errorf("cursor X after 2 lefts from \"f  \": got %d, want 1", cs2.X)
	}
}
