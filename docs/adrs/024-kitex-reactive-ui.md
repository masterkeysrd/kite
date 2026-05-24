# ADR 024: Kitex Reactive UI Framework

## Status
Accepted

## Context
Kite provides a robust, imperative `dom` package matching the logical DOM tree of a browser. However, building complex UI applications strictly via imperative DOM manipulation (e.g., `AppendChild()`, `SetAttribute()`, manually wiring state changes to text nodes) quickly leads to tight coupling, fragile state synchronization, and complex boilerplates.

To enable an ergonomic, scalable developer experience for application development, we require a higher-level state management and declarative component model that sits above the base logical DOM. 

We considered reactive signal models (like SolidJS) but ultimately determined that a React-style Functional Component (FC) model with a Virtual DOM (VDOM) diffing engine provides the best DX and closest alignment with industry standards for complex UI composition.

## Decision
We will build a high-level Virtual DOM framework called `kitex` (Kite Reactive) in the `extras/kitex` package.

### 1. Functional Components via Generics
Custom components will be fully typed using Go Generics and declared via factory functions rather than interfaces or classes.

```go
var Counter = react.FC("Counter", func(props CounterProps) react.Node {
    // Component logic
})

var Card = react.FCWithChildren("Card", func(props CardProps, children []react.Node) react.Node {
    // Component logic returning children
})
```
* **Why:** This forces components to be strictly pure functions while allowing the engine to return a specific `ComponentNode[T]` wrapper. The wrapper acts as a clear VDOM boundary for the diffing engine to track component identity and re-renders.

### 2. Kitex VDOM Primitives
The `kitex` package will define lightweight, fully-typed Virtual DOM representations for every element in the root `element` package (e.g., `kitex.Button`, `kitex.Div`).
* **Why:** The root `element` package contains heavy, real DOM element wrappers (e.g., `element.Button`). To keep the VDOM diffing extremely fast and garbage-collection friendly, `kitex` needs its own lightweight structures.
* **Mapping:** `kitex.Button(kitex.ButtonProps{...})` creates a VDOM node. During the mount/reconciliation phase, the `kitex` engine maps this VDOM node 1:1 to instantiate and update a real `element.Button` under the hood. This provides a fully typed developer experience without polluting or mutating the base `element` package API.

### 3. Implicit Hook Context (Internal Execution Stack)
React-style Hooks (e.g., `react.UseState`) will not require passing a `Context` argument down through every component function.
* **Why:** We will maintain an internal, single-threaded execution stack in the `kitex` reconciler. When a component is rendering, it is pushed to the top of the stack. A call to `react.UseState(initial)` checks the active component on the stack to attach or retrieve its persistent state. This achieves a pristine DX matching modern React without the "context drilling" normally required in Go.

### 4. VDOM Reconciliation (Diffing)
The `kitex` package will implement a diffing algorithm (Reconciler).
* When state changes, the root component (or sub-component) re-renders, yielding a new tree of `react.Node` interfaces.
* The reconciler compares this against the previous frame's VDOM tree and generates a minimal set of imperative mutations (`AppendChild`, `SetAttribute`, `RemoveChild`) applied to the real, underlying `dom.Node` tree.
* These mutations trigger Kite's existing `NeedsSync` / `ChildNeedsSync` engine pipeline.

## Consequences

* **Positive:** Developers get a familiar, React-like DX with 100% type safety and zero boilerplate inside the render function.
* **Positive:** Components are extremely composable. The base engine remains unpolluted by app-level state management.
* **Negative:** We must implement and optimize a VDOM diffing engine in Go.
* **Negative:** Go's garbage collector may face increased pressure from short-lived VDOM nodes generated on every state tick, requiring careful attention to struct sizing and potential object pooling within the `kitex` implementation.
