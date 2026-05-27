# ⚛️ Kitex

Kitex is a lightweight, reactive Virtual DOM (VDOM) framework for Kite. It provides a declarative, type-safe API for building complex terminal UIs using functional components and hooks, similar to React.

## ✨ Features

- 🧩 **Functional Components**: Define reusable UI logic using `kitex.FC` (or `kitex.FCC` for components with children). For components without props, use `kitex.SimpleFC` or `kitex.SimpleFCC`.
- 🪝 **Hooks**: Manage state and lifecycle with `UseState`, `UseRef`, `UseMemo`, `UseReducer`, and `UseCallback`.
- 🔄 **Efficient Reconciliation**: A high-performance diffing algorithm that updates only what changed in the real DOM.
- 🔑 **Keyed Lists**: Optimized list updates using unique keys to track element identity.
- 🛠 **DevTools Integration**: Deep integration with the Kite Web Inspector, including a dedicated **Components** tab.

## 🚀 Getting Started

### Basic Usage

```go
package main

import (
    "github.com/masterkeysrd/kite/extras/kitex"
    "github.com/masterkeysrd/kite/style"
)

// Define a functional component
var Counter = kitex.FC("Counter", func(props struct{}) kitex.Node {
    // Use state hook
    count, setCount := kitex.UseState(0)

    return kitex.Box(kitex.BoxProps{
        Style: style.Style{ Padding: style.Some(style.Edges(1, 1)) },
    },
        kitex.Text(fmt.Sprintf("Count: %d", count())),
        kitex.Button(kitex.ButtonProps{
            OnClick: func(e event.Event) { setCount(count() + 1) },
        }, kitex.Text("Increment")),
    )
})

// Simpler version without props
var SimpleCounter = kitex.SimpleFC("SimpleCounter", func() kitex.Node {
    count, setCount := kitex.UseState(0)
    return kitex.Button(kitex.ButtonProps{
        OnClick: func(e event.Event) { setCount(count() + 1) },
    }, kitex.Text(fmt.Sprintf("Count: %d", count())))
})

// Define a component that accepts children using FCC
type ContainerProps struct {
    Title    string
    Children []kitex.Node // Required field name for FCC injection
}

var TitledContainer = kitex.FCC("TitledContainer", func(props ContainerProps) kitex.Node {
    return kitex.Box(kitex.BoxProps{
        Style: style.Style{ Border: style.SingleBorder().Some() },
    },
        kitex.Box(kitex.BoxProps{
            Style: style.Style{ Bold: style.Some(true) },
        }, kitex.Text(props.Title)),
        kitex.Box(kitex.BoxProps{}, props.Children...),
    )
})

// Simpler version for children-only components
var Centered = kitex.SimpleFCC("Centered", func(children []kitex.Node) kitex.Node {
    return kitex.Box(kitex.BoxProps{
        Style: style.Style{ AlignItems: style.Some(style.AlignCenter) },
    }, children...)
})

func main() {
    // ... setup engine ...
    eng := engine.New(b, engine.Options{})

    // Bridge Kitex effects to the engine's macrotask queue.
    kitex.SetPostMacroFn(eng.PostMacro)

    // Render the component into a container element
    container := element.NewBox(eng.Document())
    // ...
    kitex.Render(ui, container)
}
```

## 🪝 Hooks

Kitex provides several standard hooks to manage component logic:

- **`UseState[T](initial T) (func() T, func(T))`**: Returns a getter and a setter for a state variable. Updating state triggers a re-render of the component.
- **`UseRef[T](initial T) Ref[T]`**: Returns a persistent, mutable reference that doesn't trigger re-renders when modified.
- **`UseMemo[T](factory func() T, deps []any) T`**: Memoizes an expensive calculation and re-runs it only when dependencies change.
- **`UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A))`**: Manages complex state logic using a reducer pattern. Returns a getter and a dispatch function.
- **`UseCallback[T any](callback T, deps []any) T`**: Memoizes a callback function and returns the cached reference unless dependencies change.
- **`UseEffect(effect func(), deps []any)`**: Schedules a side effect to run asynchronously after the render is committed.
- **`UseEffectCleanup(effect func() func(), deps []any)`**: Schedules a side effect with a cleanup function that runs before the next effect run or on component destroy.
- **`UseLayoutEffect(effect func(), deps []any)`**: Schedules a side effect that runs synchronously after reconciliation but before the layout is painted.
- **`UseLayoutEffectCleanup(effect func() func(), deps []any)`**: Schedules a layout side effect with a cleanup function.

### Terminal Convenience Hooks

Kitex includes terminal-specific hooks built on the core primitives:

- **`UseFocus(ref Ref[dom.Element]) bool`**: Tracks whether the referenced element currently has focus. Returns a reactive boolean.
- **`UseKeyboard(handler func(event.KeyEvent), deps []any)`**: Registers a global keyboard listener on the document. Automatically cleans up when the component is unmounted or when `deps` change.
- **`UseDocument() func() dom.Document`**: Returns a lazy getter for the component's owner document. Useful for building custom hooks that need document-level event subscriptions.
- **`UseElement() func() dom.Node`**: Returns a lazy getter for the underlying raw DOM node associated with the current component. Useful for retrieving the component's element inside side effects (e.g., to focus or measure it).

## ⚙️ Engine Wiring

For `UseEffect` and state-driven updates to function correctly within the engine's render loop, you must bridge the Kitex macrotask queue to the `engine.PostMacro` function:

```go
func main() {
    // ... setup engine ...
    eng := engine.New(b, engine.Options{})

    // Bridge Kitex effects to the engine's macrotask queue.
    // This is required for UseEffect and reactive state updates.
    kitex.SetPostMacroFn(eng.PostMacro)
    
    // ... rest of setup ...
}
```

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
var Button = kitex.SimpleFC("Button", func() kitex.Node {
    theme := kitex.UseContext(ThemeContext)
    return kitex.Box(kitex.BoxProps{}, kitex.Text(string(theme)))
})
```

When the provider value changes, only the components consuming the context (via `UseContext`) will re-render. Memoized wrapper components will not block this update, and the VDOM reconciler flattens the provider node to guarantee **zero DOM footprint** (no layout-breaking elements).

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
