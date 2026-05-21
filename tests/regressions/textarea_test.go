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
