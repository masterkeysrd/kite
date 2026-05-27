package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/flight"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/key"
)

// TestRegression_StartupAutofocus verifies that the first focusable element
// is automatically focused on the first frame, and its cursor is shown.
func TestRegression_StartupAutofocus(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	inp := element.NewInput(env.Document(), "initial")
	env.Mount(inp)

	// Before any frame, focus should be nil (no focused node).
	if env.CurrentFocus() != nil {
		t.Fatal("pre-frame focus should be nil")
	}

	// Trigger the first frame.
	env.RenderFrame()

	// 1. Focus should be on the input.
	testenv.Expect(t, inp).ToHaveFocus(env)

	// 2. The hardware cursor should be visible in the backend.
	testenv.Expect(t, inp).ExpectHardwareCursorVisible(env)
}

// TestRegression_StackNavigation_FocusTransition verifies that when navigation swaps screens,
// the focus manager blurs the disconnected element and automatically focuses the first
// focusable child in the new scope even if the autofocus target (the route root) is not focusable.
func TestRegression_StackNavigation_FocusTransition(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// 1. Initial screen structure
	container := element.NewBox(env.Document())
	env.Mount(container)

	btn1 := element.NewButton(env.Document())
	screen1 := element.Box(btn1)
	container.AppendChild(screen1)

	env.Flush()

	// Initial autofocus should focus btn1
	if env.CurrentFocus() != btn1 {
		t.Fatalf("expected btn1 to be focused initially, got %v", env.CurrentFocus())
	}

	// 2. Navigate: unmount screen1, mount screen2
	container.RemoveChild(screen1)

	btn2 := element.NewButton(env.Document())
	screen2 := element.Box(btn2)
	container.AppendChild(screen2)

	// Mimic flight.Stack pushing the new route's scope
	var rawEl dom.Node = screen2
	for {
		if unwrapped := rawEl.Unwrap(); unwrapped != nil {
			rawEl = unwrapped
		} else {
			break
		}
	}
	domEl := rawEl.(dom.Element)

	scope := &dom.FocusScope{
		Root:      domEl,
		Autofocus: domEl,
	}
	env.Document().PushScope(scope)

	env.Flush()

	// Focus should have transitioned to btn2
	if env.CurrentFocus() != btn2 {
		t.Fatalf("expected focus to transition to btn2, got %v", env.CurrentFocus())
	}
}

// TestRegression_StackNavigation_UseKeyboard verifies that key hooks inside
// routed screens are registered correctly and handle keystrokes like Esc.
func TestRegression_StackNavigation_UseKeyboard(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Register PostMacro in kitex
	kitex.SetPostMacroFn(func(fn func()) {
		env.Engine.Post(fn)
	})

	container := element.NewBox(env.Document())
	env.Mount(container)

	type Route1 struct{}
	type Route2 struct{}

	var nav flight.Navigator

	View1 := kitex.SimpleFC("View1", func() kitex.Node {
		nav = flight.UseNavigation()
		kitex.UseKeyboard(func(e event.KeyEvent) {
			t.Logf("View1 received key: %s (code: %d)", e.Text, e.Code)
			if e.MatchString("enter") {
				t.Log("View1 enter matched, pushing Route2")
				nav.Push(Route2{})
			}
		}, []any{nav})

		return kitex.Box(kitex.BoxProps{},
			kitex.Button(kitex.ButtonProps{
				ID: "btn1",
				OnClick: func(e event.Event) {
					nav.Push(Route2{})
				},
			}),
		)
	})

	View2 := kitex.SimpleFC("View2", func() kitex.Node {
		nav = flight.UseNavigation()
		kitex.UseKeyboard(func(e event.KeyEvent) {
			t.Logf("View2 received key: %s (code: %d)", e.Text, e.Code)
			if e.MatchString("esc") || e.MatchString("escape") {
				t.Log("View2 esc matched, popping")
				nav.Pop()
			}
		}, []any{nav})

		return kitex.Box(kitex.BoxProps{},
			kitex.Button(kitex.ButtonProps{
				ID: "btn2",
				OnClick: func(e event.Event) {
					nav.Pop()
				},
			}),
		)
	})

	app := flight.Stack(flight.StackProps{
		InitialRoute: Route1{},
		RenderRoute: func(r flight.Route) kitex.Node {
			switch r.(type) {
			case Route1:
				return View1()
			case Route2:
				return View2()
			default:
				return kitex.Box(kitex.BoxProps{})
			}
		},
	})

	kitex.Render(app, container)
	env.Flush()

	// Initial screen should render and focus btn1
	btn1 := env.Document().GetElementByID("btn1")
	if btn1 == nil {
		t.Fatal("expected btn1 to be rendered")
	}
	if env.CurrentFocus() != btn1 {
		t.Fatalf("expected btn1 to be focused, got %v", env.CurrentFocus())
	}

	// Press Enter to push Route2
	env.SendKey(key.Key{Code: 13}) // Enter
	env.Flush()

	// Verify details screen is loaded and btn2 is focused
	btn2 := env.Document().GetElementByID("btn2")
	if btn2 == nil {
		t.Fatal("expected btn2 to be rendered")
	}
	if env.CurrentFocus() != btn2 {
		t.Fatalf("expected btn2 to be focused after push, got %v", env.CurrentFocus())
	}

	// Press Esc to pop back to Route1
	env.SendKey(key.Key{Code: 27}) // Escape
	env.Flush()

	// Verify we are back on Route1 and btn1 is focused again
	btn1AfterPop := env.Document().GetElementByID("btn1")
	if btn1AfterPop == nil {
		t.Fatal("expected btn1 to be rendered after pop")
	}
	if env.CurrentFocus() != btn1AfterPop {
		t.Fatalf("expected focus to return to btn1 after escape, got %v", env.CurrentFocus())
	}
}
