package dom_test

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func TestSelectionInteraction(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	env.Mount(element.Box(
		element.Text("Hello World"),
	).WithID("container"))

	env.Flush()

	// "Hello World" is at 0,0.
	// H: 0,0
	// e: 1,0
	// l: 2,0
	// l: 3,0
	// o: 4,0
	//  : 5,0
	// W: 6,0
	// o: 7,0
	// r: 8,0
	// l: 9,0
	// d: 10,0

	// MouseDown on 'e' (1,0)
	env.MouseDown(1, 0, event.ButtonLeft)
	env.Flush()

	// MouseMove to 'o' (7,0)
	env.MouseMove(7, 0)
	env.Flush()

	doc := env.Document()
	sel := doc.Selection()

	if sel.RangeCount() != 1 {
		t.Fatalf("expected 1 range after mouse move, got %d", sel.RangeCount())
	}

	expected := "ello W"
	if sel.String() != expected {
		t.Errorf("expected selection %q after move, got %q", expected, sel.String())
	}

	// MouseUp at 'o' (7,0)
	env.MouseUp(7, 0, event.ButtonLeft)
	env.Flush()

	if sel.RangeCount() != 1 {
		t.Fatalf("expected 1 range after mouse up, got %d", sel.RangeCount())
	}

	if sel.String() != expected {
		t.Errorf("expected selection %q after up, got %q", expected, sel.String())
	}
}

func TestSelectionInteraction_Reverse(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	env.Mount(element.Box(
		element.Text("Hello World"),
	))

	env.Flush()

	// Drag from 'o' (7,0) back to 'e' (1,0)
	env.MouseDown(7, 0, event.ButtonLeft)
	env.Flush()

	env.MouseMove(1, 0)
	env.Flush()

	env.MouseUp(1, 0, event.ButtonLeft)
	env.Flush()

	sel := env.Document().Selection()
	expected := "ello W"
	if sel.String() != expected {
		t.Errorf("expected selection %q, got %q", expected, sel.String())
	}
}

func TestSelectionInteraction_ClearOnClick(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	env.Mount(element.Box(
		element.Text("Hello World"),
	))

	env.Flush()

	// Select something first
	env.MouseDown(1, 0, event.ButtonLeft)
	env.MouseMove(7, 0)
	env.MouseUp(7, 0, event.ButtonLeft)
	env.Flush()

	if env.Document().Selection().RangeCount() == 0 {
		t.Fatal("expected selection to be present")
	}

	// Single click should clear selection
	env.Click(2, 0)
	env.Flush()

	if env.Document().Selection().RangeCount() != 0 {
		t.Errorf("expected selection to be cleared after click, got %d ranges", env.Document().Selection().RangeCount())
	}
}

func TestSelectionInteraction_MultiLine(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Box with fixed width to force wrapping
	env.Mount(element.Box(
		element.Text("Line One. Line Two."),
	).Style(style.Style{
		Width:   style.Some(style.Cells(10)), // Wrap after "Line One. "
		Display: style.Some(style.DisplayBlock),
	}))

	env.Flush()

	// Drag from 'n' in "Line" (2,0) to 'n' in "Line" of second line (2,1)
	env.MouseDown(2, 0, event.ButtonLeft)
	env.Flush()

	env.MouseMove(2, 1)
	env.Flush()

	env.MouseUp(2, 1, event.ButtonLeft)
	env.Flush()

	sel := env.Document().Selection()
	expected := "ne One. Li"

	if sel.String() != expected {
		t.Errorf("expected selection %q, got %q", expected, sel.String())
	}
}
