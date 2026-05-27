package element_test

// Unit tests for TSK-024: InputElement on UA Shadow Subtree.
//
// These tests verify:
//   - Public ChildNodes() hides the UA text node.
//   - After typing, the UA text node reflects Buffer.Value().
//   - IntrinsicStyle() forces display, overflow, and white-space.
//   - Author-set Display:Block is resisted by the intrinsic layer.
//   - CursorState() returns correct coordinates.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Unit: public traversal invisibility
// ---------------------------------------------------------------------------

// TestInput_PublicChildren_HidesUANode verifies that ChildNodes() returns no
// children — the UA text node must be invisible to public traversal.
func TestInput_PublicChildren_HidesUANode(t *testing.T) {
	inp := element.Input("")

	count := 0
	for range inp.ChildNodes() {
		count++
	}
	if count != 0 {
		t.Errorf("ChildNodes count = %d, want 0 (UA text node must be invisible)", count)
	}
}

// TestInput_FirstLastChild_Nil verifies that FirstChild and LastChild are nil.
func TestInput_FirstLastChild_Nil(t *testing.T) {
	inp := element.Input("hello")
	if inp.FirstChild() != nil {
		t.Error("FirstChild must be nil — UA root must not be visible")
	}
	if inp.LastChild() != nil {
		t.Error("LastChild must be nil — UA root must not be visible")
	}
}

// ---------------------------------------------------------------------------
// Unit: value / UA text node synchronisation
// ---------------------------------------------------------------------------

// TestInput_InitialValue_ReflectsInValue verifies that Value() matches the
// string passed to Input().
func TestInput_InitialValue_ReflectsInValue(t *testing.T) {
	const initial = "hello"
	inp := element.Input(initial)
	if got := inp.Value(); got != initial {
		t.Errorf("Value() = %q, want %q", got, initial)
	}
}

// TestInput_SetValue_UpdatesValue verifies SetValue.
func TestInput_SetValue_UpdatesValue(t *testing.T) {
	inp := element.Input("old")
	inp.SetValue("new")
	if got := inp.Value(); got != "new" {
		t.Errorf("Value() = %q after SetValue, want %q", got, "new")
	}
}

// TestInput_Buffer_Insert_UpdatesValue verifies that direct buffer manipulation
// followed by SyncBuffer propagates to Value().
func TestInput_Buffer_Insert_UpdatesValue(t *testing.T) {
	inp := element.Input("hello")
	inp.Buffer().Insert(" world")
	inp.SyncBuffer()
	if got := inp.Value(); got != "hello world" {
		t.Errorf("Value() = %q, want %q", got, "hello world")
	}
}

// ---------------------------------------------------------------------------
// Unit: IntrinsicStyle
// ---------------------------------------------------------------------------

// TestInput_IntrinsicStyle_Properties verifies that IntrinsicStyle() returns
// the required UA-forced properties.
func TestInput_IntrinsicStyle_Properties(t *testing.T) {
	inp := element.Input("")
	is := inp.IntrinsicStyle()

	if !is.OverflowX.IsSet() || is.OverflowX.Value() != style.OverflowClip {
		t.Errorf("IntrinsicStyle.OverflowX = %v, want OverflowClip", is.OverflowX)
	}
	if !is.OverflowY.IsSet() || is.OverflowY.Value() != style.OverflowClip {
		t.Errorf("IntrinsicStyle.OverflowY = %v, want OverflowClip", is.OverflowY)
	}
	if !is.WhiteSpace.IsSet() || is.WhiteSpace.Value() != style.WhiteSpacePre {
		t.Errorf("IntrinsicStyle.WhiteSpace = %v, want WhiteSpacePre", is.WhiteSpace)
	}
}

// TestInput_AuthorStyle_OverridesDefault verifies that after an engine
// frame, the resolved computed style reflects the author's DisplayBlock override.
func TestInput_AuthorStyle_OverridesDefault(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("")
	// Author attempts to set Display:Block — should win over default InlineBlock.
	inp.Style(style.Style{
		Display: style.Some(style.DisplayBlock),
	})

	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	ro := inp.RenderObject()
	if ro == nil {
		t.Fatal("InputElement has no render object after Frame")
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

// TestInput_HandlePaste verifies that dispatching a paste event inserts text
// into the buffer.
func TestInput_HandlePaste(t *testing.T) {
	inp := element.Input("hello ")
	// Manually dispatch a paste event.
	ce := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
	ce.Items["text/plain"] = []byte("world")

	// Build path for dispatcher.
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ce, path)

	if got := inp.Value(); got != "hello world" {
		t.Errorf("Value() after paste = %q, want %q", got, "hello world")
	}
}

type mockClipboard struct {
	data string
}

func (m *mockClipboard) GetClipboard() string     { return m.data }
func (m *mockClipboard) SetClipboard(text string) { m.data = text }
func (m *mockClipboard) RequestClipboard()        {}

// ---------------------------------------------------------------------------
// Unit: Focusable
// ---------------------------------------------------------------------------

// TestInput_IsFocusable verifies the element participates in focus navigation.
func TestInput_IsFocusable(t *testing.T) {
	inp := element.Input("")
	type focusable interface{ IsFocusable() bool }
	f, ok := any(inp).(focusable)
	if !ok {
		t.Fatal("InputElement does not implement IsFocusable()")
	}
	if !f.IsFocusable() {
		t.Error("IsFocusable() = false, want true")
	}
}

// TestInput_Focus_SkipsUANodes verifies that focus.Manager does not land on
// any UA node when navigating to an input.
func TestInput_Focus_SkipsUANodes(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame() // run a frame so the render object is created

	fm := focus.NewManager(eng.Document(), event.NewDispatcher())
	if moved := fm.Next(); !moved {
		t.Fatal("focus.Next() must return true when a focusable input exists")
	}
	focused := fm.Current()
	if focused == nil {
		t.Fatal("Current() is nil after Next()")
	}
	// The focused node must be the input itself, not any UA child.
	if focused != inp {
		t.Errorf("Current() = %T %v, want *InputElement", focused, focused)
	}
}

// ---------------------------------------------------------------------------
// Unit: keyboard handling
// ---------------------------------------------------------------------------

// dispatchKeyDown dispatches a synthetic keydown event to inp.
func dispatchKeyDown(inp *element.InputElement, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)
	// Build a single-element path and dispatch to the input element.
	path := []event.EventTarget{inp}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

func dispatchMouseDownInput(target event.EventTarget, x, y int) {
	ev := event.NewMouseEvent(event.EventMouseDown, geom.Point{X: x, Y: y}, event.ButtonLeft, 0)
	ev.Local = geom.Point{X: x, Y: y}
	path := []event.EventTarget{target}
	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

// TestInput_MouseDown_SetsCursor verifies that clicking on the input updates
// the buffer's byte offset.
func TestInput_MouseDown_SetsCursor(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.NewInput(eng.Document(), "hello world")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	// Click at the start (offset 0)
	dispatchMouseDownInput(inp, 0, 0)
	if off := inp.Buffer().ByteOffset(); off != 0 {
		t.Errorf("Click at (0,0) expected offset 0, got %d", off)
	}

	// Click on 'e' (offset 1)
	dispatchMouseDownInput(inp, 1, 0)
	if off := inp.Buffer().ByteOffset(); off != 1 {
		t.Errorf("Click at (1,0) expected offset 1, got %d", off)
	}

	// Click on ' ' (offset 5)
	dispatchMouseDownInput(inp, 5, 0)
	if off := inp.Buffer().ByteOffset(); off != 5 {
		t.Errorf("Click at (5,0) expected offset 5, got %d", off)
	}
}

// TestInput_KeyDown_TypesCharacter verifies that a printable character is
// inserted into the buffer.
func TestInput_KeyDown_TypesCharacter(t *testing.T) {
	inp := element.Input("")
	dispatchKeyDown(inp, key.Key{Code: 'a', Text: "a"})
	if got := inp.Value(); got != "a" {
		t.Errorf("Value() = %q after typing 'a', want %q", got, "a")
	}
}

// TestInput_KeyDown_Backspace_DeletesPrevious verifies backspace.
func TestInput_KeyDown_Backspace_DeletesPrevious(t *testing.T) {
	inp := element.Input("hello")
	dispatchKeyDown(inp, key.Key{Code: key.KeyBackspace})
	if got := inp.Value(); got != "hell" {
		t.Errorf("Value() = %q after backspace, want %q", got, "hell")
	}
}

// TestInput_KeyDown_Delete_DeletesNext verifies delete.
func TestInput_KeyDown_Delete_DeletesNext(t *testing.T) {
	inp := element.Input("hello")
	// Move cursor to start, then delete the first character.
	inp.Buffer().MoveToStart()
	inp.SyncBuffer()
	dispatchKeyDown(inp, key.Key{Code: key.KeyDelete})
	if got := inp.Value(); got != "ello" {
		t.Errorf("Value() = %q after delete, want %q", got, "ello")
	}
}

// TestInput_KeyDown_ArrowKeys verifies cursor movement.
func TestInput_KeyDown_ArrowKeys(t *testing.T) {
	inp := element.Input("abc")
	// Buffer starts at end (offset=3).
	dispatchKeyDown(inp, key.Key{Code: key.KeyLeft})
	if got := inp.Buffer().ByteOffset(); got != 2 {
		t.Errorf("ByteOffset after Left = %d, want 2", got)
	}
	dispatchKeyDown(inp, key.Key{Code: key.KeyRight})
	if got := inp.Buffer().ByteOffset(); got != 3 {
		t.Errorf("ByteOffset after Right = %d, want 3", got)
	}
}

// TestInput_KeyDown_Home_MovesToStart verifies Home key.
func TestInput_KeyDown_Home_MovesToStart(t *testing.T) {
	inp := element.Input("hello")
	dispatchKeyDown(inp, key.Key{Code: key.KeyHome})
	if got := inp.Buffer().ByteOffset(); got != 0 {
		t.Errorf("ByteOffset after Home = %d, want 0", got)
	}
}

// TestInput_KeyDown_End_MovesToEnd verifies End key.
func TestInput_KeyDown_End_MovesToEnd(t *testing.T) {
	inp := element.Input("hello")
	inp.Buffer().MoveToStart()
	inp.SyncBuffer()
	dispatchKeyDown(inp, key.Key{Code: key.KeyEnd})
	if got := inp.Buffer().ByteOffset(); got != 5 {
		t.Errorf("ByteOffset after End = %d, want 5", got)
	}
}

// TestInput_KeyDown_CtrlModifier_NotInserted verifies that ctrl+X combos
// don't insert text.
func TestInput_KeyDown_CtrlModifier_NotInserted(t *testing.T) {
	inp := element.Input("")
	dispatchKeyDown(inp, key.Key{Code: 'c', Text: "c", Mod: key.ModCtrl})
	if got := inp.Value(); got != "" {
		t.Errorf("Value() = %q after Ctrl+C, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// Integration: engine frame renders without panic
// ---------------------------------------------------------------------------

// TestInput_EngineFrame_ProducesFrame verifies that mounting an InputElement
// and running a frame produces a valid surface without panicking.
func TestInput_EngineFrame_ProducesFrame(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	inp := element.Input("hello")
	root := element.Box(inp)
	eng.Mount(root)
	eng.Frame()

	frame := b.LastFrame()
	if frame.Surface == nil {
		t.Fatal("no surface in frame")
	}
}
