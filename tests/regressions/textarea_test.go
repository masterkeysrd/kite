package regressions

import (
	"encoding/json"
	"image/color"
	"os"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/paint"
	"github.com/masterkeysrd/kite/style"
)

func TestTextArea_Regression_Nav(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// 10 cells wide to force wrapping
	txa := element.NewTextArea(eng.Document(), "line1\nline2")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Initial cursor is at end of "line1\nline2" (offset 11)
	cs := txa.CursorState()
	if cs.X != 5 || cs.Y != 1 {
		t.Errorf("initial cursor = (%d, %d), want (5, 1)", cs.X, cs.Y)
	}

	// Up to line 1
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()
	cs = txa.CursorState()
	if cs.X != 5 || cs.Y != 0 {
		t.Errorf("cursor after Up = (%d, %d), want (5, 0)", cs.X, cs.Y)
	}

	// Down to line 2
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
	eng.Frame()
	cs = txa.CursorState()
	if cs.X != 5 || cs.Y != 1 {
		t.Errorf("cursor after Down = (%d, %d), want (5, 1)", cs.X, cs.Y)
	}
}

func TestTextArea_Regression_SoftWrapNav(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "123456789012345")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 1 || cs.X != 5 {
		t.Errorf("soft wrap cursor = (%d, %d), want (5, 1)", cs.X, cs.Y)
	}

	// Up to line 0
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()
	cs = txa.CursorState()
	if cs.Y != 0 || cs.X != 5 {
		t.Errorf("soft wrap cursor after Up = (%d, %d), want (5, 0)", cs.X, cs.Y)
	}
}

func TestTextArea_Bug1_UpFromLastChar(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "abc\ndef")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	txa.Buffer().MoveToEnd()
	txa.SyncBuffer()
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 1 || cs.X != 3 {
		t.Fatalf("Initial cursor should be at (3, 1), got (%d, %d)", cs.X, cs.Y)
	}

	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()

	cs = txa.CursorState()
	if cs.Y != 0 || cs.X != 3 {
		t.Errorf("After Up from end: cursor = (%d, %d), want (3, 0)", cs.X, cs.Y)
	}
}

func TestTextArea_Bug2_WithPadding(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "abc\ndef")
	txa.Style(style.Style{
		Width:   style.Some(style.Cells(20)),
		Height:  style.Some(style.Cells(5)),
		Padding: style.Some(style.Edges(0, 1)),
		Border:  style.SingleBorder().Some(),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	txa.Buffer().SetOffset(5)
	txa.SyncBuffer()
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 2 || cs.X != 3 {
		t.Fatalf("Initial cursor at 'e' should be (3, 2), got (%d, %d)", cs.X, cs.Y)
	}

	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()

	cs = txa.CursorState()
	if cs.Y != 1 || cs.X != 3 {
		t.Errorf("After Up with padding: cursor = (%d, %d), want (3, 1)", cs.X, cs.Y)
	}
}

func TestTextArea_Bug3_StuckInThirdLine(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	initialText := "Welcome!\n\nThird line"
	txa := element.NewTextArea(eng.Document(), initialText)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	txa.Buffer().SetOffset(10)
	txa.SyncBuffer()
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 2 || cs.X != 0 {
		t.Fatalf("Initial cursor at 'Third line' should be (0, 2), got (%d, %d)", cs.X, cs.Y)
	}

	dispatchKeyToTarget(txa, key.Key{Code: key.KeyUp})
	eng.Frame()

	cs = txa.CursorState()
	if cs.Y != 1 {
		t.Errorf("After first Up: cursor Y = %d, want 1", cs.Y)
	}
}

func TestTextArea_DumpTool(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "Dump test")
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Bind Ctrl+P manually in the application/test layer
	root.AddEventListener(event.EventKeyDown, func(ev event.Event) {
		ke := ev.(*event.KeyEvent)
		if ke.MatchString("ctrl+p") {
			_ = eng.Dump("kite-dump-test.json")
		}
	})

	dispatchKeyToTarget(txa, key.Key{Code: 'p', Mod: key.ModCtrl})
	eng.Frame()

	if _, err := os.Stat("kite-dump-test.json"); os.IsNotExist(err) {
		t.Fatalf("kite-dump-test.json was not created")
	}
	defer os.Remove("kite-dump-test.json")

	data, err := os.ReadFile("kite-dump-test.json")
	if err != nil {
		t.Fatalf("failed to read dump: %v", err)
	}

	var dump struct {
		ScreenSize struct {
			Width int `json:"width"`
		} `json:"screen_size"`
	}
	if err := json.Unmarshal(data, &dump); err != nil {
		t.Fatalf("failed to unmarshal dump: %v", err)
	}

	if dump.ScreenSize.Width != 80 {
		t.Errorf("dump.ScreenSize.Width = %d, want 80", dump.ScreenSize.Width)
	}
}

// TestTextArea_Bug4_DownFromLastRowStaysOnLastRow verifies that pressing Down
// while the cursor is anywhere on the last content row does not advance the
// cursor to a phantom row beyond the content (i.e. the trailing-<br> line).
//
// Regression for: pressing Down from mid-last-row moves buffer offset to the
// end of the last line (perceived as cursor jumping to last+1).
func TestTextArea_Bug4_DownFromLastRowStaysOnLastRow(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Two lines: move cursor to the START of the last line (not the end).
	txa := element.NewTextArea(eng.Document(), "line1\nline2")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Move cursor to the start of "line2" (offset 6).
	txa.Buffer().SetOffset(6)
	txa.SyncBuffer()
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 1 {
		t.Fatalf("pre-condition: cursor Y = %d, want 1 (start of last line)", cs.Y)
	}
	beforeOffset := txa.Buffer().ByteOffset()
	beforeY := cs.Y

	// Press Down — cursor is on the last content row and must not move.
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
	eng.Frame()

	cs = txa.CursorState()
	if cs.Y != beforeY {
		t.Errorf("Down from last row: cursor Y moved from %d to %d, want no change", beforeY, cs.Y)
	}
	// The buffer offset must not change either — pressing Down on the last line
	// must be a strict no-op, not a jump-to-end-of-line.
	if got := txa.Buffer().ByteOffset(); got != beforeOffset {
		t.Errorf("Down from last row: buffer offset changed from %d to %d, want no change", beforeOffset, got)
	}
}

// TestTextArea_Bug4_DownFromLastRowSoftWrap verifies the same invariant when
// the last line is produced by soft-wrap rather than a hard newline.
func TestTextArea_Bug4_DownFromLastRowSoftWrap(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Single long line that soft-wraps at column 10 → produces two visual rows.
	// Place cursor at start of the second (last) visual row.
	txa := element.NewTextArea(eng.Document(), "1234567890abcde")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Offset 10 = start of the second wrapped row ("abcde" starts at byte 10).
	txa.Buffer().SetOffset(10)
	txa.SyncBuffer()
	eng.Frame()

	cs := txa.CursorState()
	if cs.Y != 1 {
		t.Fatalf("pre-condition: cursor Y = %d, want 1 (soft-wrap last row)", cs.Y)
	}
	beforeOffset := txa.Buffer().ByteOffset()
	beforeY := cs.Y

	// Press Down — cursor is on the last visual row and must not move.
	dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
	eng.Frame()

	cs = txa.CursorState()
	if cs.Y != beforeY {
		t.Errorf("Down from last soft-wrap row: cursor Y moved from %d to %d, want no change", beforeY, cs.Y)
	}
	if got := txa.Buffer().ByteOffset(); got != beforeOffset {
		t.Errorf("Down from last soft-wrap row: buffer offset changed from %d to %d, want no change", beforeOffset, got)
	}
}

// TestTextArea_Bug4_DownFromLastRowWhenOverflow verifies that pressing Down on
// the last row is a strict no-op even when the content overflows the viewport
// and a non-zero scroll offset is in effect. The mid-row variant checks that
// the buffer offset does not jump to the end of the line.
func TestTextArea_Bug4_DownFromLastRowWhenOverflow(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// 12 lines in a 4-row viewport — forces scrolling.
	text := "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11"
	txa := element.NewTextArea(eng.Document(), text)
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(4)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// Navigate from start to the last line via Down presses.
	txa.Buffer().MoveToStart()
	txa.SyncBuffer()
	eng.Frame()

	prevY, prevOff := -1, -1
	for i := 0; i < 20; i++ {
		cs := txa.CursorState()
		off := txa.Buffer().ByteOffset()
		if cs.Y == prevY && off == prevOff {
			break
		}
		prevY, prevOff = cs.Y, off
		dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
		eng.Frame()
	}

	// Now on the last line. Record state.
	lastY := txa.CursorState().Y
	lastOff := txa.Buffer().ByteOffset()

	// Three more Down presses must all be no-ops.
	for i := 1; i <= 3; i++ {
		dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
		eng.Frame()
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
	eng.Frame()

	midY := txa.CursorState().Y
	if midY != lastY {
		t.Fatalf("mid-line pre-condition: Y = %d, want %d", midY, lastY)
	}

	dispatchKeyToTarget(txa, key.Key{Code: key.KeyDown})
	eng.Frame()

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
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "hello\nworld")
	txa.Style(style.Style{
		Width:  style.Some(style.Cells(20)),
		Height: style.Some(style.Cells(5)),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// After the first frame, buffer is at end (offset 11), cursor is at
	// uaDiv Y=1 (last line). CursorState() should be (5,1).
	cs := txa.CursorState()
	if cs.X != 5 || cs.Y != 1 {
		t.Fatalf("pre-condition: cursor = (%d,%d), want (5,1)", cs.X, cs.Y)
	}

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
	// Do NOT call SyncBuffer / eng.Frame() — fragment is still for the old value.

	cs = txa.CursorState()
	// With the fix: should return last known good position (5,1), not (0,0).
	if cs.X != 5 || cs.Y != 1 {
		t.Errorf("CursorState with stale fragment = (%d,%d), want last-known (5,1)", cs.X, cs.Y)
	}

	// After the next frame the fragment is refreshed; cursor is now at the
	// start of the new empty line (offset 12 → uaDiv Y=2, X=0).
	txa.SyncBuffer()
	eng.Frame()

	cs = txa.CursorState()
	if cs.X != 0 || cs.Y != 2 {
		t.Errorf("CursorState after frame = (%d,%d), want (0,2)", cs.X, cs.Y)
	}
}

func TestTextArea_CrashOverflow(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "line1")
	txa.Style(style.Style{
		Width:      style.Some(style.Cells(20)),
		Height:     style.Some(style.Cells(3)),
		Background: style.Some[color.Color](color.White),
	})
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	for i := 0; i < 10; i++ {
		dispatchKeyToTarget(txa, key.Key{Code: key.KeyEnter})
		eng.Frame()
	}

	ro := txa.RenderObject()
	frag := ro.Fragment()
	if frag.Size.Height != 3 {
		t.Errorf("textarea height = %d, want 3", frag.Size.Height)
	}

	fb := paint.NewFrameBuffer(0, 0, 80, 20)
	eng.PaintEngine().Paint(eng.RenderView().Fragment(), fb)
}

func TestTextArea_IsFocusable(t *testing.T) {
	eng := engine.New(mock.New(80, 20), engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "")
	if !txa.IsFocusable() {
		t.Errorf("TextArea should be focusable")
	}
}

func TestTextArea_AutoFocus_And_Sync(t *testing.T) {
	b := mock.New(80, 20)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	txa := element.NewTextArea(eng.Document(), "")
	root := element.Box(txa)
	eng.Mount(root)
	eng.Frame()

	// 1. Verify Auto-focus works on first key press
	eng.ProcessRawEvent(&event.RawKeyEvent{
		Key: key.Key{Code: 'x', Text: "x"},
	})
	eng.Frame()

	if eng.FocusManager().Current() != txa {
		t.Errorf("TextArea should be auto-focused after key press, got %v", eng.FocusManager().Current())
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

func dispatchKeyToTarget(target event.EventTarget, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)

	// Build the path from target up to root.
	var path []event.EventTarget
	curr := target
	for curr != nil {
		path = append(path, curr)
		if n, ok := curr.(dom.Node); ok {
			p := n.Parent()
			if p == nil {
				break
			}
			curr = p
		} else {
			break
		}
	}

	// Reverse the path so it's root -> target.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}
