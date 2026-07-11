package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestSelect_LayoutWidth(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Case 1: Select with explicit width.
	// The internal button should stretch to match this width.
	s1 := element.Select(
		element.Option("Option 1", "1"),
	).WithID("select-fixed").Style(style.S().Width(style.Cells(25)))

	env.Mount(s1)
	env.Flush()

	sel1 := env.GetNodeByID("select-fixed").(*element.SelectElement)
	ro1 := env.RenderObject(sel1)
	if ro1 == nil {
		t.Fatal("expected render object for select-fixed")
	}

	if w := ro1.Fragment().Size.Width; w != 25 {
		t.Errorf("expected select width 25, got %d", w)
	}

	// Internal button should be 21 (25 - 4 for border + padding).
	btn1 := findUAButton(sel1)
	if btn1 == nil {
		t.Fatal("could not find internal button for select-fixed")
	}
	if w := env.RenderObject(btn1).Fragment().Size.Width; w != 21 {
		t.Errorf("expected internal button width 21, got %d", w)
	}

	// Case 2: Select with Auto width (default).
	// It should shrink/grow to fit the internal button's content.
	s2 := element.Select(
		element.Option("A very long option name that should make it grow", "1"),
	).WithID("select-auto")

	env.Mount(s2)
	env.Flush()

	sel2 := env.GetNodeByID("select-auto").(*element.SelectElement)
	ro2 := env.RenderObject(sel2)

	w2 := ro2.Fragment().Size.Width
	t.Logf("Auto Select width: %d", w2)
	if w2 < 15 {
		t.Errorf("expected auto select width to be at least 15, got %d", w2)
	}

	btn2 := findUAButton(sel2)
	if btn2Width := env.RenderObject(btn2).Fragment().Size.Width; btn2Width != w2-4 {
		t.Errorf("expected internal button width %d to match host content width %d, got %d", w2-4, w2-4, btn2Width)
	}

	// Case 3: Narrow host.
	// The internal button should be constrained to the host width.
	s3 := element.Select().Style(style.S().Width(style.Cells(10)))
	env.Mount(s3)
	env.Flush()

	if w := env.RenderObject(s3).Fragment().Size.Width; w != 10 {
		t.Errorf("expected select width 10, got %d", w)
	}
	btn3 := findUAButton(s3)
	if w := env.RenderObject(btn3).Fragment().Size.Width; w != 6 {
		t.Errorf("expected internal button width 6, got %d", w)
	}
}

func TestSelect_LayoutHeight(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Select inside a tall flex container.
	// It should NOT stretch to fill the height unless explicitly told to.
	s := element.Select(
		element.Option("Option 1", "1"),
	).WithID("select")

	container := element.Box(s).Style(style.S().Display(style.DisplayFlex).AlignItems(style.AlignStretch).Height(style.Cells(20)))

	env.Mount(container)
	env.Flush()

	sel := env.GetNodeByID("select").(*element.SelectElement)
	ro := env.RenderObject(sel)

	if h := ro.Fragment().Size.Height; h > 5 {
		t.Errorf("expected select to maintain natural height, got %d", h)
	}

	btn := findUAButton(sel)
	if h := env.RenderObject(btn).Fragment().Size.Height; h > 5 {
		t.Errorf("expected internal button to maintain natural height, got %d", h)
	}
}

func TestSelect_KeyboardNavigation(t *testing.T) {
	t.Skip("Skipping until we implement proper focus management for select dropdowns")
	env := testenv.Default(80, 24)
	defer env.Close()

	s := element.Select(
		element.Option("Option 1", "opt1"),
		element.Option("Option 2", "opt2"),
		element.Option("Option 3", "opt3"),
	).WithID("select")

	env.Mount(s)
	env.Flush()

	// 1. Press Down on closed select -> opens and focuses Option 1
	doc := env.Engine.Document()
	doc.Focus(s)
	env.Flush()

	env.SendKey(key.Key{Code: key.KeyDown})
	env.Flush()

	current := doc.CurrentFocus()
	if current == nil {
		t.Fatal("expected focused element")
	}

	// The select element remains focused because the overlay is just a child of the document
	// but fm.Current() should return the focused node in the active scope.
	// Wait, if PushScope happened, the focus manager's current node should have changed.

	t.Logf("Focused node type: %T", current)

	btn1, ok := current.(*element.ButtonElement)
	if !ok {
		t.Fatalf("expected focused ButtonElement, got %T", current)
	}
	if btn1.TextContent() != " Option 1" {
		t.Errorf("expected focused Option 1, got %q", btn1.TextContent())
	}

	// 2. Press Down again -> focuses Option 2
	env.SendKey(key.Key{Code: key.KeyDown})
	env.Flush()
	current = doc.CurrentFocus()
	if current.(*element.ButtonElement).TextContent() != " Option 2" {
		t.Errorf("expected focused Option 2, got %q", current.(*element.ButtonElement).TextContent())
	}

	// 3. Press Up -> focuses Option 1
	env.SendKey(key.Key{Code: key.KeyUp})
	env.Flush()
	current = doc.CurrentFocus()
	if current.(*element.ButtonElement).TextContent() != " Option 1" {
		t.Errorf("expected focused Option 1, got %q", current.(*element.ButtonElement).TextContent())
	}

	// 4. Press Enter -> selects Option 1 and closes
	env.SendKey(key.Key{Code: key.KeyEnter})
	env.Flush()
	if s.Value() != "opt1" {
		t.Errorf("expected value opt1, got %q", s.Value())
	}
}

func findUAButton(el element.Element) *element.ButtonElement {
	uaRoot := dom.UARoot(el)
	if uaRoot == nil {
		return nil
	}
	for child := range uaRoot.ChildNodes() {
		if b, ok := child.(*element.ButtonElement); ok {
			return b
		}
		// Also check unwrapped
		for cur := child; cur != nil; cur = cur.Unwrap() {
			if b, ok := cur.(*element.ButtonElement); ok {
				return b
			}
		}
	}
	return nil
}
