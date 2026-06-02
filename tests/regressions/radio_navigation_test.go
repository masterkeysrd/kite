package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestRadioArrowNavigationFocus(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	cb := element.Checkbox(false)
	r1 := element.Radio("option1")
	r2 := element.Radio("option2")
	r3 := element.Radio("option3")

	rg := element.RadioGroup(
		element.Box(r1, element.Span(" Option 1")),
		element.Box(r2, element.Span(" Option 2")),
		element.Box(r3, element.Span(" Option 3")),
	)

	root := element.Box(cb, rg)
	env.Engine.Mount(root)
	env.Flush()

	// Track focus and blur target events.
	var focusHistory []string
	var blurHistory []string

	env.Engine.Document().AddEventListener(event.EventFocus, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](color.RGBA{R: 255, G: 215, B: 0, A: 255})
				el.Style(s)
				focusHistory = append(focusHistory, "checkbox")
			case *element.RadioElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](color.RGBA{R: 255, G: 215, B: 0, A: 255})
				el.Style(s)
				focusHistory = append(focusHistory, el.Value().(string))
			}
		}
	}, event.Capture())

	env.Engine.Document().AddEventListener(event.EventBlur, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](style.TerminalDefault)
				el.Style(s)
				blurHistory = append(blurHistory, "checkbox")
			case *element.RadioElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](style.TerminalDefault)
				el.Style(s)
				blurHistory = append(blurHistory, el.Value().(string))
			}
		}
	}, event.Capture())

	// Step 0: Check initial autofocus. The Checkbox (first focusable) should be focused.
	focused := env.Engine.Document().CurrentFocus()
	if focused != cb {
		t.Errorf("Step 0: Expected checkbox focused initially, got %T (%v)", focused, focused)
	}

	// Step 1: Send Tab to move focus to the first Radio button (r1)
	env.SendKey(key.Key{Code: key.KeyTab})
	env.Flush()

	focused = env.Engine.Document().CurrentFocus()
	if focused != r1 {
		t.Errorf("Step 1: Expected r1 focused, got %T (%v)", focused, focused)
	}

	// Step 2: Send Down Arrow (navigates within radio group to r2)
	env.SendKey(key.Key{Code: key.KeyDown})
	env.Flush()

	focused = env.Engine.Document().CurrentFocus()
	if focused != r2 {
		t.Errorf("Step 2: Expected r2 focused, got %T (%v)", focused, focused)
	}

	// Check style values to see if r1 reverted to default and r2 is Gold.
	if r1.RawStyle().Foreground.Value() != style.TerminalDefault {
		t.Errorf("Expected r1 style to revert to TerminalDefault, got %v", r1.RawStyle().Foreground.Value())
	}
	gold := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	if r2.RawStyle().Foreground.Value() != gold {
		t.Errorf("Expected r2 style to be Gold, got %v", r2.RawStyle().Foreground.Value())
	}

	// Step 3: Send Down Arrow (navigates to r3)
	env.SendKey(key.Key{Code: key.KeyDown})
	env.Flush()

	focused = env.Engine.Document().CurrentFocus()
	if focused != r3 {
		t.Errorf("Step 3: Expected r3 focused, got %T (%v)", focused, focused)
	}

	// Step 4: Send Down Arrow (wraps back to r1)
	env.SendKey(key.Key{Code: key.KeyDown})
	env.Flush()

	focused = env.Engine.Document().CurrentFocus()
	if focused != r1 {
		t.Errorf("Step 4: Expected r1 focused after wrapping down, got %T (%v)", focused, focused)
	}

	// Step 5: Send Up Arrow (wraps to r3)
	env.SendKey(key.Key{Code: key.KeyUp})
	env.Flush()

	focused = env.Engine.Document().CurrentFocus()
	if focused != r3 {
		t.Errorf("Step 5: Expected r3 focused after wrapping up, got %T (%v)", focused, focused)
	}

	// Print lists to debug
	t.Logf("focusHistory: %v", focusHistory)
	t.Logf("blurHistory: %v", blurHistory)
}
