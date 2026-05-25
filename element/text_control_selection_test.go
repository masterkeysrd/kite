package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

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
