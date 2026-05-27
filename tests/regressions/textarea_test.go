package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/internal/paint"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

func TestTextArea_Regression_Nav(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	// 10 cells wide to force wrapping
	txa := element.NewTextArea(e.Document(), "line1\nline2")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// Initial cursor is at end of "line1\nline2" (offset 11)
	testenv.Expect(t, txa).ToHaveCursorAt(5, 1)

	// Up to line 1
	e.DispatchKey(txa, key.Key{Code: key.KeyUp})
	e.RenderFrame()
	testenv.Expect(t, txa).ToHaveCursorAt(5, 0)

	// Down to line 2
	e.DispatchKey(txa, key.Key{Code: key.KeyDown})
	e.RenderFrame()
	testenv.Expect(t, txa).ToHaveCursorAt(5, 1)
}

func TestTextArea_Regression_SoftWrapNav(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "123456789012345")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(5, 1)

	// Up to line 0
	e.DispatchKey(txa, key.Key{Code: key.KeyUp})
	e.RenderFrame()
	testenv.Expect(t, txa).ToHaveCursorAt(5, 0)
}

func TestTextArea_Bug1_UpFromLastChar(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "abc\ndef")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	txa.Buffer().MoveToEnd()
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(3, 1)

	e.DispatchKey(txa, key.Key{Code: key.KeyUp})
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(3, 0)
}

func TestTextArea_Bug2_WithPadding(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "abc\ndef")
	txa.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Height:  style.Some(style.Cells(5)),
		Padding: style.Some(style.Edges(0, 1)),
		Border:  style.SingleBorder().Some(),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	txa.Buffer().SetOffset(5)
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(3, 2)

	e.DispatchKey(txa, key.Key{Code: key.KeyUp})
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(3, 1)
}

func TestTextArea_Bug3_StuckInThirdLine(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	initialText := "Welcome!\n\nThird line"
	txa := element.NewTextArea(e.Document(), initialText)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	txa.Buffer().SetOffset(10)
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(0, 2)

	e.DispatchKey(txa, key.Key{Code: key.KeyUp})
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(0, 1)
}

// TestTextArea_Bug4_DownFromLastRowStaysOnLastRow verifies that pressing Down
// while the cursor is anywhere on the last content row does not advance the
// cursor to a phantom row beyond the content (i.e. the trailing-<br> line).
//
// Regression for: pressing Down from mid-last-row moves buffer offset to the
// end of the last line (perceived as cursor jumping to last+1).
func TestTextArea_Bug4_DownFromLastRowStaysOnLastRow(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	// Two lines: move cursor to the START of the last line (not the end).
	txa := element.NewTextArea(e.Document(), "line1\nline2")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// Move cursor to the start of "line2" (offset 6).
	txa.Buffer().SetOffset(6)
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(0, 1)
	beforeOffset := txa.Buffer().ByteOffset()
	beforeY := txa.CursorState().Y

	// Press Down — cursor is on the last content row and must not move.
	e.DispatchKey(txa, key.Key{Code: key.KeyDown})
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(txa.CursorState().X, beforeY)
	// The buffer offset must not change either — pressing Down on the last line
	// must be a strict no-op, not a jump-to-end-of-line.
	if got := txa.Buffer().ByteOffset(); got != beforeOffset {
		t.Errorf("Down from last row: buffer offset changed from %d to %d, want no change", beforeOffset, got)
	}
}

// TestTextArea_Bug4_DownFromLastRowSoftWrap verifies the same invariant when
// the last line is produced by soft-wrap rather than a hard newline.
func TestTextArea_Bug4_DownFromLastRowSoftWrap(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	// Single long line that soft-wraps at column 10 → produces two visual rows.
	// Place cursor at start of the second (last) visual row.
	txa := element.NewTextArea(e.Document(), "1234567890abcde")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// Offset 10 = start of the second wrapped row ("abcde" starts at byte 10).
	txa.Buffer().SetOffset(10)
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(0, 1)
	beforeOffset := txa.Buffer().ByteOffset()
	beforeY := txa.CursorState().Y

	// Press Down — cursor is on the last visual row and must not move.
	e.DispatchKey(txa, key.Key{Code: key.KeyDown})
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(txa.CursorState().X, beforeY)
	if got := txa.Buffer().ByteOffset(); got != beforeOffset {
		t.Errorf("Down from last soft-wrap row: buffer offset changed from %d to %d, want no change", beforeOffset, got)
	}
}

// TestTextArea_Bug4_DownFromLastRowWhenOverflow verifies that pressing Down on
// the last row is a strict no-op even when the content overflows the viewport
// and a non-zero scroll offset is in effect. The mid-row variant checks that
// the buffer offset does not jump to the end of the line.
func TestTextArea_Bug4_DownFromLastRowWhenOverflow(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	// 12 lines in a 4-row viewport — forces scrolling.
	text := "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11"
	txa := element.NewTextArea(e.Document(), text)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(4)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// Navigate from start to the last line via Down presses.
	txa.Buffer().MoveToStart()
	txa.SyncBuffer()
	e.RenderFrame()

	prevY, prevOff := -1, -1
	for range 20 {
		cs := txa.CursorState()
		off := txa.Buffer().ByteOffset()
		if cs.Y == prevY && off == prevOff {
			break
		}
		prevY, prevOff = cs.Y, off
		e.DispatchKey(txa, key.Key{Code: key.KeyDown})
		e.RenderFrame()
	}

	// Now on the last line. Record state.
	lastY := txa.CursorState().Y
	lastOff := txa.Buffer().ByteOffset()

	// Three more Down presses must all be no-ops.
	for i := 1; i <= 3; i++ {
		e.DispatchKey(txa, key.Key{Code: key.KeyDown})
		e.RenderFrame()
		cs := txa.CursorState()
		if cs.Y != lastY || txa.Buffer().ByteOffset() != lastOff {
			t.Errorf("Down #%d from last row (overflow): Y %d→%d off %d→%d, want no change",
				i, lastY, cs.Y, lastOff, txa.Buffer().ByteOffset())
		}
	}

	// Also test from mid-line on the last row (not at end of line).
	// "line11" starts at offset 67; place cursor 2 chars in.
	txa.Buffer().SetOffset(69)
	txa.SyncBuffer()
	e.RenderFrame()

	midY := txa.CursorState().Y
	if midY != lastY {
		t.Fatalf("mid-line pre-condition: Y = %d, want %d", midY, lastY)
	}

	e.DispatchKey(txa, key.Key{Code: key.KeyDown})
	e.RenderFrame()

	if txa.CursorState().Y != midY || txa.Buffer().ByteOffset() != 69 {
		t.Errorf("Down from mid-last-row (overflow): Y %d→%d off %d→%d, want no change",
			midY, txa.CursorState().Y, 69, txa.Buffer().ByteOffset())
	}
}

// TestTextArea_Bug5_CursorStateStaleFragment verifies that CursorState() returns
// the last known good position — not the top-left corner — when called with a
// stale fragment (e.g. from a keydown listener right after a buffer mutation,
// before the next layout pass runs).
//
// Regression for: after pressing Enter at the end of the initial text the status
// bar incorrectly showed Pos:(insetLeft,insetTop) instead of the real cursor
// position, making subsequent navigation appear to jump.
func TestTextArea_Bug5_CursorStateStaleFragment(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "hello\nworld")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// After the first frame, buffer is at end (offset 11), cursor is at
	// uaDiv Y=1 (last line). CursorState() should be (5,1).
	testenv.Expect(t, txa).ToHaveCursorAt(5, 1)

	// Now simulate a keydown listener reading CursorState() immediately after
	// a buffer mutation, but BEFORE the next layout pass. We do this by:
	// 1. Inserting text directly into the buffer (bypassing syncCallback).
	// 2. Calling CursorState() — at this point the fragment is stale (it was
	//    computed for the old buffer value).
	//
	// Previously CursorState() would return (0,0) when FromTextFragment failed
	// (offset past end of stale fragment), causing the status bar to show
	// (insetLeft,insetTop) = (0,0) instead of the real last position.
	txa.Buffer().Insert("\n") // buffer now has "hello\nworld\n", offset=12
	// Do NOT call SyncBuffer / e.RenderFrame() — fragment is still for the old value.

	// With the fix: should return last known good position (5,1), not (0,0).
	testenv.Expect(t, txa).ToHaveCursorAt(5, 1)

	// After the next frame the fragment is refreshed; cursor is now at the
	// start of the new empty line (offset 12 → uaDiv Y=2, X=0).
	txa.SyncBuffer()
	e.RenderFrame()

	testenv.Expect(t, txa).ToHaveCursorAt(0, 2)
}

func TestTextArea_CrashOverflow(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "line1")
	txa.Style(style.Style{
		Width:      style.Some(style.Cells(20)),
		Height:     style.Some(style.Cells(3)),
		Background: style.Some[color.Color](color.White),
	})
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	for range 10 {
		e.DispatchKey(txa, key.Key{Code: key.KeyEnter})
		e.RenderFrame()
	}

	testenv.Expect(t, txa).ToHaveFragmentHeight(3)

	fb := paint.NewFrameBuffer(0, 0, 80, 20)
	e.Engine.PaintEngine().Paint(nil, e.Engine.RenderView().Fragment(), fb)
}

func TestTextArea_IsFocusable(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "")
	if !txa.IsFocusable() {
		t.Errorf("TextArea should be focusable")
	}
}

func TestTextArea_AutoFocus_And_Sync(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	txa := element.NewTextArea(e.Document(), "")
	root := element.Box(txa)
	e.Mount(root)
	e.RenderFrame()

	// 1. Verify Auto-focus works on first key press
	e.SendKey(key.Key{Code: 'x', Text: "x"})
	e.RenderFrame()

	if !e.HasFocus(txa) {
		t.Errorf("TextArea should be auto-focused after key press, got %v", e.Engine.Document().CurrentFocus())
	}

	if txa.Value() != "x" {
		t.Errorf("Value = %q, want \"x\"", txa.Value())
	}

	// 2. Verify that Sync flags propagate from UA subtree (repainting fix)
	// We check if the host element is marked for sync.
	txa.Buffer().Insert("y")
	txa.SyncBuffer() // This calls rebuildUASubtree and MarkNeedsSync on a UA node

	if !txa.NeedsSync() {
		t.Error("Host TextArea should need sync after UA subtree modification")
	}
}

// TestTextArea_Regression_ScrollCursorPos verifies that the hardware cursor
// stays aligned with the text content when the textarea is scrolled.
// This handles the regression where content was panned incorrectly relative
// to the cursor due to border/padding insets in the paint engine's clamping logic.
func TestTextArea_Regression_ScrollCursorPos(t *testing.T) {
	e := testenv.Default(80, 26)
	defer e.Close()

	// 5 lines of text
	initialText := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	txa := element.NewTextArea(e.Document(), initialText)
	txa.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Height:  style.Some(style.Cells(7)), // 3 lines visible + 4 cells inset (border=1, padding=1)
		Padding: style.Some(style.Edges(1, 1)),
		Border:  style.SingleBorder().Some(),
	})

	root := element.Box(txa)
	e.Mount(root)

	// Focus the textarea so the engine tracks its cursor.
	e.Engine.Document().Focus(txa)
	e.RenderFrame()

	// Place cursor at "Line 3" (offset 14)
	txa.Buffer().SetOffset(14)
	txa.SyncBuffer()
	e.RenderFrame()

	// Line 3 is at local Y=2.
	// state.Y = insetTop + localY = 2 + 2 = 4.
	// Initial scroll is (0,0) because Line 3 fits in contentH=3.

	testenv.Expect(t, txa).
		ExpectHardwareCursorVisible(e).
		ExpectHardwareCursorY(e, 4)

	// Scroll down by 1 line.
	txa.ScrollTo(0, 1)
	e.RenderFrame()
	testenv.Expect(t, txa).ExpectHardwareCursorY(e, 3)

	// Scroll down by 2 lines.
	txa.ScrollTo(0, 2)
	e.RenderFrame()
	testenv.Expect(t, txa).ExpectHardwareCursorY(e, 2)
}
