# ✈️ Flight

Flight is a standard-compliant, wrapper-free stack-based navigation system for Kitex applications. By favoring a type-safe stack navigator (push/pop) over URL-based path routing, Flight provides a paradigm that matches TUI (Terminal User Interface) design patterns and enforces strict keyboard focus containment to the active screen.

## ✨ Features

- ✈️ **Type-Safe Stack Navigation**: Define routes as user-defined structs with type-safe parameters, rather than fragile URL/string path routing.
- 🔒 **Keyboard Focus Containment**: Seamlessly restricts Tab focus loop and keyboard interaction to the top-most/active route screen.
- 📦 **Wrapper-Free Design**: Focus containment registers the active route's underlying raw DOM element directly with `focus.Manager` via `UseElement()`, avoiding layout-breaking wrapper components like `<focus-scope>` or `<box>`.
- 🪝 **Ergonomic Hooks**: Access the navigator stack anywhere in the component tree with `UseNavigation()`.

## 🚀 Getting Started

### Basic Navigation

First, define your routes as structs. Concrete routes can hold parameters:

```go
package main

import (
	"github.com/masterkeysrd/kite/extras/flight"
	"github.com/masterkeysrd/kite/extras/kitex"
)

// Define your routes
type HomeRoute struct{}
type DetailsRoute struct {
	ItemID string
}

// Parent App component rendering the stack
var App = kitex.SimpleFC("App", func() kitex.Node {
	return flight.Stack(flight.StackProps{
		InitialRoute: HomeRoute{},
		RenderRoute: func(r flight.Route) kitex.Node {
			switch route := r.(type) {
			case HomeRoute:
				return HomeView()
			case DetailsRoute:
				return DetailsView(route.ItemID)
			default:
				panic("unknown route")
			}
		},
	})
})
```

### Navigating Between Screens

Use the `flight.UseNavigation()` hook to access the `Navigator` interface and execute actions:

```go
var HomeView = kitex.SimpleFC("HomeView", func() kitex.Node {
	nav := flight.UseNavigation()

	return kitex.Box(kitex.BoxProps{},
		kitex.Text("Welcome Home"),
		kitex.Button(kitex.ButtonProps{
			OnClick: func(e event.Event) {
				// Push a new details route onto the stack
				nav.Push(DetailsRoute{ItemID: "42"})
			},
		}, kitex.Text("Go to Details")),
	)
})

var DetailsView = kitex.FC("DetailsView", func(props string) kitex.Node {
	nav := flight.UseNavigation()

	return kitex.Box(kitex.BoxProps{},
		kitex.Text("Item Details: " + props),
		kitex.Button(kitex.ButtonProps{
			OnClick: func(e event.Event) {
				// Pop the screen to return to the home screen
				nav.Pop()
			},
		}, kitex.Text("Go Back")),
	)
})
```

## 🛠 Navigator API

The `Navigator` interface returned by `UseNavigation()` provides the following operations to manage the stack:

- **`Push(r Route)`**: Pushes a new route onto the stack, making it the active screen.
- **`Pop()`**: Pops the current top route from the stack, returning to the previous one (does nothing if only the initial route is present).
- **`Replace(r Route)`**: Replaces the current top route with a new one.
- **`Reset(r Route)`**: Clears the stack entirely and sets the given route as the sole entry.

## 🔒 Keyboard Focus Isolation

Under the hood, Flight leverages Kitex's `UseElement()` and `UseLayoutEffectCleanup` hooks to obtain the raw DOM node of the active screen. When a screen becomes active, Flight pushes a new `focus.Scope` pointing to this DOM node onto the `focus.Manager` stack. This ensures:
- Keyboard tab/arrow navigation is locked inside the active view's boundary.
- Child components do not need to wrap themselves in arbitrary navigation UI wrappers.
- The layout is not broken by structural VDOM wrapping nodes.
