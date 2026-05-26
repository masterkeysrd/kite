# ADR 025: Kitex DevTools Integration — Components Inspector & Source Tracking

## Status
Accepted

## Context
The `extras/kitex` reactive framework (ADR-024) provides a React-style functional component and VDOM reconciler for building Kite TUI applications. However, the existing DevTools inspector (ADR-014, ADR-020) only exposes the raw logical DOM tree — it has no awareness of the higher-level VDOM component graph.

Developers debugging Kitex applications faced several pain points:

1. **No component visibility**: The Elements tab shows real DOM nodes but not which functional component (`FC`) or which props/hooks are associated with a subtree.
2. **No hook introspection**: There was no way to observe live `UseState` / `UseRef` values without adding `fmt.Println` statements.
3. **No source attribution**: It was impossible to identify which line in user code created a particular native element (e.g. `kitex.Box`, `kitex.Button`).
4. **No instantiation context**: Even for component nodes, the DevTools could not distinguish _where_ in the user's component tree a given element was used.

## Decision

We extend the DevTools inspector with a **Components tab** specific to `extras/kitex`, driven by three coordinated sub-decisions:

### 1. Inspector Extension API (`devtools/inspector`)
We add a lightweight `Extension` interface to the inspector:

```go
type Extension interface {
    Name() string
    GetPayload(eng *engine.Engine) any
}
```

`Inspector.RegisterExtension(ext Extension)` stores extensions by name. On every SSE snapshot, each extension's `GetPayload` is called and the result is included under `payload.extensions[name]`. This keeps the core inspector generic and completely decoupled from Kitex.

`devtools.Install` is updated to return `(*inspector.Inspector, error)` so callers can hold the inspector pointer and register extensions after installation.

### 2. VDOM Source Tracking (`extras/kitex`)
Every VDOM node (`elementNode[P]`, `textNode`, `ComponentNode[P]`) gains four source fields:

```go
declFile string  // file where the kitex factory is defined
declLine int
instFile string  // file in user code that called the factory
instLine int
```

A `trackSource(node Node, skip int) Node` helper populates these fields using `runtime.Caller` **only when `kitex.EnableDevMode == true`**, making the cost strictly zero in production. For native element factories (`Box`, `Button`, `TD`, etc.) `skip=1` resolves: frame 0 = `trackSource`, frame 1 = the factory function itself (the declaration), and then it walks up the call stack skipping frames inside `extras/kitex` to find the first user-land frame (the instantiation site).

`ComponentNode[P]` also exposes the `componentNodeInspector` interface:

```go
type componentNodeInspector interface {
    getHooks() []any
    getRendered() Node
    getDecl() (string, int)
    getInst() (string, int)
}
```

Hooks that hold inspectable values implement `hookValuer`:

```go
type hookValuer interface {
    getValue() any
}
```

Both `hookState[T]` and `RefObject[T]` implement `hookValuer`.

### 3. Kitex DevTools Bridge (`extras/kitex/kitexdt`)
A minimal bridge package `kitexdt` wraps everything behind a single call:

```go
kitexdt.Register(insp)
```

Internally it registers a `kitexExtension` that calls `kitex.BuildDevToolsSnapshot(eng)` on every inspector tick. `BuildDevToolsSnapshot` walks the active VDOM roots (locked under `renderMutex`) and produces a JSON-serialisable `[]*VDOMSnapshot` tree that the frontend consumes.

### 4. Frontend Components Tab (`devtools/ui`)
The Preact frontend detects the presence of `payload.extensions.kitex` and renders a **Components** tab alongside Elements and Profiler. Key UI features:

- **Tree**: Shows functional components by name and native elements by tag. Component nodes display their rendered subtree.
- **Hooks panel** (renamed from "State"): Each hook value is displayed as `hookN: <value>`. If the value is a Go struct, map, or slice it renders as a collapsible JSON-style tree, matching the React DevTools and Chrome DevTools experience.
- **Props panel**: Serialised props for any selected node (event handler functions are omitted).
- **Source panel**: Shows *Declared at* and *Instantiated at* with short relative paths.
- **⇒ Elements link**: Cross-links the selected VDOM node to its corresponding DOM node in the Elements tab using the `domUniqueId` field.

## Consequences

**Positive:**
- Zero overhead in production (`EnableDevMode` guard).
- No coupling between the core `devtools` package and `extras/kitex`; the Extension API is fully generic.
- Developers get React DevTools-quality component introspection including live hook state, source locations, and cross-linking to the DOM inspector.
- All 18 native element factories are covered: `Text`, `Box`, `Div`, `Span`, `Button`, `Checkbox`, `RadioGroup`, `Radio`, `Select`, `Option`, `Input`, `TextArea`, `Table`, `THead`, `TBody`, `TFoot`, `TR`, `TD`, `Br`, `Overlay`, `Dialog`.

**Negative:**
- `devtools.Install` signature changed (now returns `(*inspector.Inspector, error)`); callers that previously discarded the return value need a minor update.
- `runtime.Caller` walks add ~1–3 µs per node creation in dev mode. This is acceptable for development but must not be enabled in benchmarks or production.
- The Preact UI bundle must be rebuilt (`npm run build` inside `devtools/ui`) whenever the frontend is modified. The compiled `devtools/static/index.html` is committed to the repository so Go users never need Node.js.

## Usage Summary

```go
kitex.EnableDevMode = true          // must be set before first Render
kitex.Render(Root(Props{}), host)

insp, _ := devtools.Install(eng, devtools.Options{})
kitexdt.Register(insp)              // activates the Components tab
```
