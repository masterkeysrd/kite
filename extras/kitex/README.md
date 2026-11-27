# ⚛️ Kitex

Kitex is a lightweight, reactive Virtual DOM (VDOM) framework for Kite. It provides a declarative, type-safe API for building complex terminal UIs using functional components and hooks, similar to React.

<p align="center">
  <img width="1000" height="600" alt="kitex_demo" src="https://github.com/user-attachments/assets/0aab873b-8bb8-4f02-9323-6041bfc301a3" />
</p>

## ✨ Features

- 🧩 **Functional Components**: Define reusable UI logic using `kitex.FC` (leaf components) or `kitex.FCC` (container components accepting `children ...kitex.Node`), with `kitex.NoProps` for components without props. Call `Comp()` / `Cont(children...)` for zero-value props or `Comp.Props(p)` / `Cont.Props(p, children...)` for explicit props.
- 🧱 **Typed Element Wrappers**: Build UIs with typed `ContainerTag` and `LeafTag` constructors for Kite elements such as `Box`, `Span`, `Button`, `Table`, `UL`, `OL`, and `LI`. Call `kitex.Box(kitex.BoxProps{...}, children...)` for full props, or `kitex.Box.Style(s, children...)` / `kitex.Span.Style(s, children...)` / `kitex.Input.Style(s)` when only setting `Style`.
- 🪝 **Hooks**: Manage state and lifecycle with `UseState`, `UseRef`, `UseMemo`, `UseReducer`, and `UseCallback`.
- 🔄 **Efficient Reconciliation**: A high-performance diffing algorithm that updates only what changed in the real DOM.
- 🔑 **Keyed Lists**: Optimized list updates using unique keys to track element identity.
- 🛠 **DevTools Integration**: Deep integration with the Kite Web Inspector, including a dedicated **Components** tab.

## 🚀 Getting Started

### Basic Usage

```go
package main

import (
    "github.com/masterkeysrd/kite/event"
    "github.com/masterkeysrd/kite/extras/kitex"
    "github.com/masterkeysrd/kite/style"
)

// Define a functional component with no props
var Counter = kitex.FC("Counter", func(_ kitex.NoProps) kitex.Node {
    // Use state hook
    count, setCount := kitex.UseState(0)

    return kitex.Box.Style(
        style.S().Padding(1, 1),
        kitex.Textf("Count: %d", count()),
        kitex.Button(kitex.ButtonProps{
            OnClick: func(e *event.MouseEvent) { setCount(count() + 1) },
            Style:   style.S().Hover(style.S().Background(style.Hex("#334155"))),
        }, kitex.Text("Increment")),
    )
})

// Functional component with explicit props
type GreetingProps struct {
    Name string
}

var Greeting = kitex.FC("Greeting", func(props GreetingProps) kitex.Node {
    return kitex.Textf("Hello, %s!", props.Name)
})

// Define a component that accepts children using FCC
type ContainerProps struct {
    Title string
}

var TitledContainer = kitex.FCC("TitledContainer", func(props ContainerProps, children ...kitex.Node) kitex.Node {
    return kitex.Box.Style(
        style.S().Border(style.SingleBorder()),
        kitex.Span.Style(style.S().Bold(true), kitex.Text(props.Title)),
        kitex.Box(kitex.BoxProps{}, children...),
    )
})

// Container component with no props
var Centered = kitex.FCC("Centered", func(_ kitex.NoProps, children ...kitex.Node) kitex.Node {
    return kitex.Box.Style(
        style.S().AlignItems(style.AlignCenter),
        children...,
    )
})

func main() {
    // ... setup engine ...
    eng := engine.New(b, engine.Options{})

    // Render the component into a container element
    container := eng.Document().Body() // or any other element
    // ...
    kitex.Render(
        Centered(
            Greeting.Props(GreetingProps{Name: "Kite"}),
            TitledContainer.Props(ContainerProps{Title: "Stats"}, Counter()),
        ),
        container,
    )
}
```

## 🪝 Hooks

Kitex provides several standard hooks to manage component logic:

- **`UseState[T](initial T) (func() T, StateSetter[T])`**: Returns a getter and a `StateSetter[T]` handle (`type StateSetter[T any] func(T)`). Call `set(val)` directly to replace the state, or `set.Update(func(prev T) T)` for an atomic, goroutine-safe read-modify-write under the hook mutex. `StateSetter[T]` is directly assignable to `func(T)` callback props (e.g. `OnValueChange: setQuery`). Updating state triggers a re-render of the component.
- **`UseRef[T](initial T) Ref[T]`**: Returns a persistent, mutable reference that doesn't trigger re-renders when modified.
- **`UseMemo[T](factory func() T, deps ...any) T`**: Memoizes an expensive calculation and re-runs it only when dependencies change (runs once on mount when called with no `deps`).
- **`UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A))`**: Manages complex state logic using a reducer pattern. Returns a getter and a dispatch function.
- **`UseCallback[T any](callback T, deps ...any) T`**: Memoizes a callback function and returns the cached reference unless dependencies change (memoizes across component lifetime when called with no `deps`).
- **`UseEffect(effect func(), deps ...any)`**: Schedules a side effect to run asynchronously after the render is committed (runs once on mount when called with no `deps`).
- **`UseEffectCleanup(effect func() func(), deps ...any)`**: Schedules a side effect with a cleanup function that runs before the next effect run or on component destroy.
- **`UseEffectEveryRender(effect func())`** / **`UseEffectEveryRenderCleanup(effect func() func())`**: Schedules a post-commit side effect (with optional cleanup) to run after every render.
- **`UseAsyncEffect(effect func(ctx context.Context), deps ...any)`**: Schedules a cancellable asynchronous side effect managed by the host scheduler.
- **`UseLayoutEffect(effect func(), deps ...any)`**: Schedules a side effect that runs synchronously after reconciliation but before the layout is painted (runs once on mount when called with no `deps`).
- **`UseLayoutEffectCleanup(effect func() func(), deps ...any)`**: Schedules a layout side effect with a cleanup function.
- **`UseLayoutEffectEveryRender(effect func())`** / **`UseLayoutEffectEveryRenderCleanup(effect func() func())`**: Schedules a synchronous layout side effect (with optional cleanup) to run after every render.

### Terminal Convenience Hooks

Kitex includes terminal-specific hooks built on the core primitives:

- **`UseFocus(ref Ref[dom.Element]) bool`**: Tracks whether the referenced element currently has focus. Returns a reactive boolean.
- **`UseKeyboard(handler func(event.KeyEvent), deps ...any)`**: Registers a global keyboard listener on the document. Automatically cleans up when the component is unmounted or when `deps` change.
- **`UseTimeout(handler func(), delay time.Duration, deps ...any)`** / **`UseInterval(handler func(), interval time.Duration, deps ...any)`**: Schedules one-shot or repeating callbacks on the host scheduler with automatic unmount cleanup.
- **`UseDocument() func() dom.Document`**: Returns a lazy getter for the component's owner document. Useful for building custom hooks that need document-level event subscriptions.
- **`UseElement() func() dom.Node`**: Returns a lazy getter for the underlying raw DOM node associated with the current component. Useful for retrieving the component's element inside side effects (e.g., to focus or measure it).
- **`UseViewportSize() geom.Size`**: Subscribes the component to viewport resize events and returns the current terminal size. Triggering a resize dynamically re-renders the component to support responsive layout restructuring.
- **`UseTerminal() func() terminal.Terminal`**: Returns a lazy getter for the terminal object associated with the current component.
- **`UseTitle(initialTitle string) (func() string, func(string))`**: Initializes and manages the terminal window title. Returns a getter and a setter; calling the setter automatically updates the window title.
- **`UseBell() func()`**: Returns a memoized callback function to trigger the terminal hardware bell without causing re-renders.

## 🌐 Context System

The Context system allows sharing values deep down the component tree without manually passing props through every level (avoiding "prop drilling").

### 1. Create a Context
```go
type Theme string
var ThemeContext = kitex.CreateContext[Theme]("light")
```

### 2. Provide a Value
Wrap your subtree with the context provider:
```go
ThemeContext.Provider("dark", 
    MyApp(),
)
```

### 3. Consume the Value
Call `UseContext` inside any functional component in the subtree:
```go
var Button = kitex.FC("Button", func(_ kitex.NoProps) kitex.Node {
    theme := kitex.UseContext(ThemeContext)
    return kitex.Box(kitex.BoxProps{}, kitex.Text(string(theme)))
})
```

When the provider value changes, only the components consuming the context (via `UseContext`) will re-render. Memoized wrapper components will not block this update, and the VDOM reconciler flattens the provider node to guarantee **zero DOM footprint** (no layout-breaking elements).

## 📝 Forms

Kitex provides a declarative `<form>` wrapper via `kitex.Form` to handle form submission and value collection. Form controls (such as `kitex.Input`, `kitex.Checkbox`, `kitex.Radio`, and `kitex.Select`) must specify a `Name` attribute. When the form is submitted (either by pressing `Enter` on a single-line input field or clicking a button with `Type: "submit"`), the form automatically gathers the name-value mappings of all descendants and triggers `OnSubmit`.

### Example Form

```go
var UserForm = kitex.FC("UserForm", func(_ kitex.NoProps) kitex.Node {
    handleSubmit := func(data map[string]any) {
        fmt.Printf("Form submitted: %v\n", data)
    }

    return kitex.Form(kitex.FormProps{
        OnSubmit: handleSubmit,
    },
        kitex.Input(kitex.InputProps{
            Name:  "username",
            Value: "default_user",
        }),
        kitex.Checkbox(kitex.CheckboxProps{
            Name:    "agree",
            Checked: false,
        }),
        kitex.Button(kitex.ButtonProps{
            Type: "submit",
        }, kitex.Text("Submit")),
    )
})
```

## 🧰 Utilities

Kitex provides several ergonomic helpers to make VDOM construction cleaner, especially for declarative rendering:

- **`kitex.Map(items []D, fn func(D, int) Node) Node`**: Transforms a slice of data into a Node (a Fragment). Excellent for rendering keyed lists.
- **`kitex.Nodes(nodes ...Node) Node`**: Merges multiple nodes (including Fragments and individual nodes) into a single flat Fragment. Filters out `nil` values.
- **`kitex.If(cond bool, fn func() Node) Node`**: Conditionally renders a node if `cond` is true; otherwise, returns `Empty()` (which is safely ignored by Kitex).
- **`kitex.IfElse(cond bool, thenNode Node, elseNode Node) Node`**: Returns `thenNode` if true, or `elseNode` if false. Both branches are evaluated eagerly.
- **`kitex.IfElseFn(cond bool, thenFn, elseFn func() Node) Node`**: Lazily evaluates only `thenFn` when `cond` is true or `elseFn` when false, returning `Empty()` if the chosen closure is `nil` or returns `nil`.
- **`kitex.Fragment(children ...Node) Node`**: Returns the provided children grouped together as a single Node without introducing a wrapper element in the DOM.

### Example Utilities Usage

```go
var ItemList = kitex.FC("ItemList", func(props struct{ Items []string }) kitex.Node {
    return kitex.Box(kitex.BoxProps{},
        kitex.If(len(props.Items) == 0, func() kitex.Node {
            return kitex.Text("No items found.")
        }),
        
        kitex.Box(kitex.BoxProps{}, 
            kitex.Map(props.Items, func(item string, idx int) kitex.Node {
                return kitex.Textf("%d: %s", idx, item)
            }),
        ),
    )
})
```

## 🛠 Developer Tools

When using Kitex, you can enable the **Components** inspector in the Kite Web Inspector. This provides a live view of your VDOM tree, including:

- **Component Hierarchy**: Inspect the nesting of functional components.
- **Hook State**: View the current values of all `UseState` and `UseRef` hooks.
- **Props**: See the properties passed to each component.
- **Source Tracking**: Jump directly to the file and line where a component or element was instantiated.

### Enabling DevTools

```go
import (
    "github.com/masterkeysrd/kite/extras/kitex"
    "github.com/masterkeysrd/kite/extras/kitex/kitexdt"
)

func main() {
    // 1. Enable source tracking
    kitex.EnableDevMode = true

    // 2. Register the bridge with your inspector
    insp, _ := devtools.Install(eng, devtools.Options{})
    kitexdt.Register(insp)
}
```

## 📖 Best Practices

- **Use Keys**: Always provide a unique `Key` prop when rendering lists of elements to ensure optimal reconciliation.
- **Keep Components Small**: Break down complex UIs into smaller, focused functional components.
- **Memoize Expensive Renders**: Use `UseMemo` for heavy computations or complex subtrees that don't change often.
