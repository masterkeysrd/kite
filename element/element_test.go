package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestDeclarativeAPI(t *testing.T) {
	// Test case 1: Box("Hello", Span("World"))
	box := element.Box("Hello", element.Span("World"))

	if box.NodeName() != "box" {
		t.Errorf("expected box, got %s", box.NodeName())
	}

	// First child should be text "Hello"
	child1 := box.FirstChild()
	if child1 == nil || child1.NodeName() != "#text" || child1.TextContent() != "Hello" {
		t.Errorf("expected first child to be text 'Hello', got %v", child1)
	}

	// Second child should be span "World"
	child2 := child1.NextSibling()
	if child2 == nil || child2.NodeName() != "span" {
		t.Errorf("expected second child to be span, got %v", child2)
	}
	if child2.TextContent() != "World" {
		t.Errorf("expected span text to be 'World', got %q", child2.TextContent())
	}

	// Test case 2: Flattening slices
	box2 := element.Box([]any{"a", "b"})
	if box2.FirstChild().TextContent() != "a" {
		t.Errorf("expected 'a'")
	}
	if box2.FirstChild().NextSibling().TextContent() != "b" {
		t.Errorf("expected 'b'")
	}
}

func TestInlineTreeConstruction(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Using declarative API
	box := element.Box(
		element.Span(
			element.Text("Hello"),
		),
	)
	eng.Mount(box)

	// Trigger sync phase
	eng.Frame()

	// Verify Render Objects are created
	boxRO := eng.RenderObject(box)
	if boxRO == nil {
		t.Fatal("Box render object not created")
	}

	span := box.FirstChild()
	spanRO := eng.RenderObject(span)
	if spanRO == nil {
		t.Fatal("Span render object not created")
	}

	text := span.FirstChild()
	textRO := eng.RenderObject(text)
	if textRO == nil {
		t.Fatal("Text render object not created")
	}

	// Verify Render Tree Structure
	if spanRO.Parent() != boxRO {
		t.Errorf("Span RO parent should be Box RO")
	}
	if textRO.Parent() != spanRO {
		t.Errorf("Text RO parent should be Span RO")
	}
}

func TestMixedTreeConstruction(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	box := element.Box(
		element.Span(),
		element.Box(),
	)
	eng.Mount(box)

	eng.Frame()

	boxRO := eng.RenderObject(box)
	span := box.FirstChild()
	spanRO := eng.RenderObject(span)
	nestedBox := span.NextSibling()
	nestedBoxRO := eng.RenderObject(nestedBox)

	if boxRO.FirstChild() != spanRO {
		t.Errorf("expected Span RO as first child")
	}
	if spanRO.NextSibling() != nestedBoxRO {
		t.Errorf("expected nested Box RO as next sibling of Span RO")
	}
}

func TestElementInterface_SetStyleAndSetHidden(t *testing.T) {
	box := element.Box("hello")
	var el element.Element = box

	s := style.S().Padding(2).Background(style.Hex("#123456"))
	el.SetStyle(s)
	if !el.RawStyle().Equal(s) {
		t.Fatalf("RawStyle mismatch after SetStyle")
	}
	if !box.IsDirtyStyle() {
		t.Fatal("SetStyle must mark style dirty")
	}

	el.SetHidden(true)
	if !el.IsHidden() {
		t.Fatal("IsHidden should be true after SetHidden(true)")
	}
	el.SetHidden(false)
	if el.IsHidden() {
		t.Fatal("IsHidden should be false after SetHidden(false)")
	}
}

func TestElementTypedEventHelpers(t *testing.T) {
	box := element.Box()

	var (
		clicked    int
		pressed    int
		mouseDown  int
		mouseUp    int
		mouseMove  int
		mouseEnter int
		mouseLeave int
		keyDown    int
		keyUp      int
		keyPress   int
		keyCombo   int
		focused    int
		blurred    int
		wheeled    int
	)

	// Nil handlers must be safely ignored.
	box.OnClick(nil).
		OnPress(nil).
		OnMouseDown(nil).
		OnMouseUp(nil).
		OnMouseMove(nil).
		OnMouseEnter(nil).
		OnMouseLeave(nil).
		OnKeyDown(nil).
		OnKeyUp(nil).
		OnKeyPress(nil).
		OnKey("ctrl+s", nil).
		OnFocus(nil).
		OnBlur(nil).
		OnWheelEvent(nil)

	if len(box.Listeners()) != 0 {
		t.Fatalf("expected 0 listeners after nil registrations, got %d", len(box.Listeners()))
	}

	box.OnClick(func(e *event.MouseEvent) { clicked++ }).
		OnPress(func() { pressed++ }).
		OnMouseDown(func(e *event.MouseEvent) { mouseDown++ }).
		OnMouseUp(func(e *event.MouseEvent) { mouseUp++ }).
		OnMouseMove(func(e *event.MouseEvent) { mouseMove++ }).
		OnMouseEnter(func(e *event.MouseEvent) { mouseEnter++ }).
		OnMouseLeave(func(e *event.MouseEvent) { mouseLeave++ }).
		OnKeyDown(func(e *event.KeyEvent) { keyDown++ }).
		OnKeyUp(func(e *event.KeyEvent) { keyUp++ }).
		OnKeyPress(func(e *event.KeyEvent) { keyPress++ }).
		OnKey("ctrl+s", func(e *event.KeyEvent) { keyCombo++ }).
		OnFocus(func(e *event.FocusEvent) { focused++ }).
		OnBlur(func(e *event.FocusEvent) { blurred++ }).
		OnWheelEvent(func(e *event.WheelEvent) { wheeled++ })

	box.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonRight, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventMouseDown, geom.Point{}, event.ButtonLeft, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventMouseUp, geom.Point{}, event.ButtonLeft, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventMouseMove, geom.Point{}, event.ButtonNone, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventMouseEnter, geom.Point{}, event.ButtonNone, 0))
	box.DispatchEvent(event.NewMouseEvent(event.EventMouseLeave, geom.Point{}, event.ButtonNone, 0))
	box.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'a', Text: "a"}))
	box.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 's', Mod: key.ModCtrl}))
	box.DispatchEvent(event.NewKeyEvent(event.EventKeyUp, key.Key{Code: 'a'}))
	box.DispatchEvent(event.NewKeyEvent(event.EventKeyPress, key.Key{Code: 'a', Text: "a"}))
	box.DispatchEvent(event.NewFocusEvent(event.EventFocus, nil))
	box.DispatchEvent(event.NewFocusEvent(event.EventBlur, nil))
	box.DispatchEvent(event.NewWheelEvent(geom.Point{}, 0, 1, 0))

	if clicked != 2 || pressed != 1 {
		t.Fatalf("click/press mismatch: clicked=%d pressed=%d", clicked, pressed)
	}
	if mouseDown != 1 || mouseUp != 1 || mouseMove != 1 || mouseEnter != 1 || mouseLeave != 1 {
		t.Fatalf("mouse mismatch: down=%d up=%d move=%d enter=%d leave=%d",
			mouseDown, mouseUp, mouseMove, mouseEnter, mouseLeave)
	}
	if keyDown != 2 || keyUp != 1 || keyPress != 1 || keyCombo != 1 {
		t.Fatalf("key mismatch: down=%d up=%d press=%d combo=%d",
			keyDown, keyUp, keyPress, keyCombo)
	}
	if focused != 1 || blurred != 1 || wheeled != 1 {
		t.Fatalf("focus/blur/wheel mismatch: focused=%d blurred=%d wheeled=%d",
			focused, blurred, wheeled)
	}

	// Button keyboard activation fires OnPress.
	btnPressed := 0
	btn := element.Button("OK").OnPress(func() { btnPressed++ })
	btn.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: key.KeyEnter}))
	if btnPressed != 1 {
		t.Fatalf("button Enter should fire OnPress, got %d", btnPressed)
	}
}

func TestControlParity_InputTextAreaCheckboxOnChange(t *testing.T) {
	t.Run("InputElement", func(t *testing.T) {
		env := testenv.New(t, 40, 10)
		defer env.Close()

		var changes []string
		inp := element.Input("init").WithID("inp").OnChange(func(val string) {
			changes = append(changes, val)
		})
		env.Mount(element.Box(inp))
		env.Flush()

		if inp.Text() != "init" {
			t.Fatalf("Text() = %q, want %q", inp.Text(), "init")
		}

		// Programmatic SetValue must NOT fire OnChange.
		inp.SetValue("prog")
		if inp.Text() != "prog" {
			t.Fatalf("Text() after SetValue = %q, want %q", inp.Text(), "prog")
		}
		if len(changes) != 0 {
			t.Fatalf("SetValue must not fire OnChange, got %v", changes)
		}

		// User typing must fire OnChange.
		env.Type("!")
		env.Flush()
		if inp.Text() != "prog!" {
			t.Fatalf("Text() after Type = %q, want %q", inp.Text(), "prog!")
		}
		if len(changes) != 1 || changes[0] != "prog!" {
			t.Fatalf("expected OnChange [\"prog!\"], got %v", changes)
		}
	})

	t.Run("TextAreaElement", func(t *testing.T) {
		env := testenv.New(t, 40, 10)
		defer env.Close()

		var changes []string
		txa := element.TextArea("hi").WithID("txa").OnChange(func(val string) {
			changes = append(changes, val)
		})
		env.Mount(element.Box(txa))
		env.Flush()

		if txa.Text() != "hi" {
			t.Fatalf("Text() = %q, want %q", txa.Text(), "hi")
		}

		// Programmatic SetValue must NOT fire OnChange.
		txa.SetValue("line1")
		if len(changes) != 0 {
			t.Fatalf("SetValue must not fire OnChange, got %v", changes)
		}

		// User newline + typing fires OnChange.
		env.KeyPress("enter")
		env.Type("b")
		env.Flush()
		if txa.Text() != "line1\nb" {
			t.Fatalf("Text() = %q, want %q", txa.Text(), "line1\nb")
		}
		if len(changes) != 2 || changes[1] != "line1\nb" {
			t.Fatalf("expected OnChange on newline & char, got %v", changes)
		}
	})

	t.Run("CheckboxElement", func(t *testing.T) {
		env := testenv.New(t, 40, 10)
		defer env.Close()

		var changes []bool
		cb := element.Checkbox(false).WithID("cb").OnChange(func(checked bool) {
			changes = append(changes, checked)
		})
		env.Mount(element.Box(cb))
		env.Flush()

		// Programmatic SetChecked must NOT fire OnChange.
		cb.SetChecked(true)
		if !cb.Checked() {
			t.Fatal("expected Checked() == true")
		}
		if len(changes) != 0 {
			t.Fatalf("SetChecked must not fire OnChange, got %v", changes)
		}

		// User toggle (Space) fires OnChange.
		env.KeyPress("space")
		env.Flush()
		if cb.Checked() {
			t.Fatal("expected Checked() == false after Space toggle")
		}
		if len(changes) != 1 || changes[0] != false {
			t.Fatalf("expected OnChange [false], got %v", changes)
		}
	})
}

func TestDisableable_SetDisabled_MarksStyleDirtyAndEvaluatesPseudo(t *testing.T) {
	env := testenv.New(t, 60, 20)
	defer env.Close()

	cBase := style.Hex("#111111")
	cDis := style.Hex("#999999")
	st := style.S().Background(cBase).Disabled(style.S().Background(cDis))

	btn := element.Button("B").Style(st)
	cb := element.Checkbox(false).Style(st)
	r := element.Radio("a").Style(st)
	rg := element.RadioGroup(r).Style(st)
	opt := element.Option("O", "o").Style(st)
	sel := element.Select(opt).Style(st)
	inp := element.Input("").Style(st)
	txa := element.TextArea("").Style(st)

	root := element.Box(btn, cb, rg, sel, inp, txa)
	env.Mount(root)
	env.Flush()

	controls := []struct {
		name string
		el   element.Element
		dis  interface {
			SetDisabled(bool)
			IsDisabled() bool
		}
	}{
		{"Button", btn, btn},
		{"Checkbox", cb, cb},
		{"Radio", r, r},
		{"RadioGroup", rg, rg},
		{"Select", sel, sel},
		{"Input", inp, inp},
		{"TextArea", txa, txa},
	}

	for _, c := range controls {
		if ro := env.RenderObject(c.el); ro.ComputedStyle().Background != cBase {
			t.Fatalf("%s initial Background = %v, want %v", c.name, ro.ComputedStyle().Background, cBase)
		}
		c.dis.SetDisabled(true)
	}
	opt.SetDisabled(true)
	if !opt.IsDisabled() {
		t.Fatal("OptionElement should be disabled")
	}

	env.Flush()
	for _, c := range controls {
		if ro := env.RenderObject(c.el); ro.ComputedStyle().Background != cDis {
			t.Fatalf("%s disabled Background = %v, want %v", c.name, ro.ComputedStyle().Background, cDis)
		}
		c.dis.SetDisabled(false)
	}

	env.Flush()
	for _, c := range controls {
		if ro := env.RenderObject(c.el); ro.ComputedStyle().Background != cBase {
			t.Fatalf("%s re-enabled Background = %v, want %v", c.name, ro.ComputedStyle().Background, cBase)
		}
	}
}
