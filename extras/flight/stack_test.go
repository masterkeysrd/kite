package flight

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/internal/focus"
)

type HomeRoute struct{}
type DetailsRoute struct{ ID string }

func TestStackNavigation(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	var nav Navigator

	HomeView := kitex.SimpleFC("HomeView", func() kitex.Node {
		nav = UseNavigation()
		return kitex.Box(kitex.BoxProps{ID: "home"})
	})

	DetailsView := kitex.FC("DetailsView", func(props struct{ ID string }) kitex.Node {
		nav = UseNavigation()
		return kitex.Box(kitex.BoxProps{ID: "details_" + props.ID})
	})

	stack := Stack(StackProps{
		InitialRoute: HomeRoute{},
		RenderRoute: func(r Route) kitex.Node {
			switch route := r.(type) {
			case HomeRoute:
				return HomeView()
			case DetailsRoute:
				return DetailsView(struct{ ID string }{ID: route.ID})
			default:
				return kitex.Box(kitex.BoxProps{ID: "unknown"})
			}
		},
	})

	// 1. Initial Render
	kitex.Render(stack, container)

	if nav == nil {
		t.Fatal("expected UseNavigation to retrieve navigator")
	}

	// Verify home is rendered
	first := container.FirstChild()
	if first == nil {
		t.Fatal("expected container to have children")
	}
	screen, ok := first.(dom.Element)
	if !ok {
		t.Fatalf("expected active screen element, got %T", first)
	}
	if screen.ID() != "home" {
		t.Errorf("expected active screen to be 'home', got %q", screen.ID())
	}

	// 2. Push DetailsRoute
	nav.Push(DetailsRoute{ID: "42"})

	// Verify details rendered
	screen = container.FirstChild().(dom.Element)
	if screen.ID() != "details_42" {
		t.Errorf("expected active screen to be 'details_42', got %q", screen.ID())
	}

	// 3. Pop
	nav.Pop()
	screen = container.FirstChild().(dom.Element)
	if screen.ID() != "home" {
		t.Errorf("expected active screen to return to 'home', got %q", screen.ID())
	}

	// 4. Pop beyond bounds (bounds check)
	nav.Pop()
	screen = container.FirstChild().(dom.Element)
	if screen.ID() != "home" {
		t.Errorf("expected active screen to remain 'home' after popping beyond bounds, got %q", screen.ID())
	}

	// 5. Replace
	nav.Replace(DetailsRoute{ID: "99"})
	screen = container.FirstChild().(dom.Element)
	if screen.ID() != "details_99" {
		t.Errorf("expected active screen to be replaced with 'details_99', got %q", screen.ID())
	}

	// 6. Reset
	nav.Reset(HomeRoute{})
	screen = container.FirstChild().(dom.Element)
	if screen.ID() != "home" {
		t.Errorf("expected active screen to be reset to 'home', got %q", screen.ID())
	}
}

func TestFocusScopeIsolation(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	app := kitex.SimpleFC("App", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{},
			kitex.Button(kitex.ButtonProps{ID: "outer-btn"}),
			Stack(StackProps{
				InitialRoute: HomeRoute{},
				RenderRoute: func(r Route) kitex.Node {
					return kitex.Box(kitex.BoxProps{ID: "screen"},
						kitex.Button(kitex.ButtonProps{ID: "inner-btn-1"}),
						kitex.Button(kitex.ButtonProps{ID: "inner-btn-2"}),
					)
				},
			}),
		)
	})

	kitex.Render(app(), container)

	scope := doc.ActiveScope()
	if scope == nil {
		t.Fatal("expected Document to provide ActiveScope()")
	}

	outerBtn := doc.GetElementByID("outer-btn")
	innerBtn1 := doc.GetElementByID("inner-btn-1")
	innerBtn2 := doc.GetElementByID("inner-btn-2")

	if outerBtn == nil || innerBtn1 == nil || innerBtn2 == nil {
		t.Fatal("expected buttons to be instantiated and reachable via GetElementByID")
	}

	screenEl := doc.GetElementByID("screen")

	// Verify that the focus scope is active
	activeScope := doc.ActiveScope()
	if activeScope == nil {
		t.Fatal("expected active focus scope")
	}
	if activeScope.Root != screenEl.Unwrap() {
		t.Errorf("expected active scope root to be screen element DOM node, got %v", activeScope.Root)
	}

	// Set focus to innerBtn1
	doc.Focus(innerBtn1)
	if doc.CurrentFocus() != innerBtn1 {
		t.Errorf("expected inner-btn-1 to be focused, got %v", doc.CurrentFocus())
	}

	// Cycle next: innerBtn1 -> innerBtn2
	doc.NextFocus()
	if doc.CurrentFocus() != innerBtn2 {
		t.Errorf("expected focus to move to inner-btn-2, got %v", doc.CurrentFocus())
	}

	// Cycle next: innerBtn2 -> innerBtn1 (should wrap within scope, skipping outerBtn!)
	doc.NextFocus()
	if doc.CurrentFocus() != innerBtn1 {
		t.Errorf("expected focus to wrap to inner-btn-1, got %v", doc.CurrentFocus())
	}
}
