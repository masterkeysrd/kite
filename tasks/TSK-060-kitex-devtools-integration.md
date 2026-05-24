# TSK-060: Kitex DevTools Integration

## Objective
Design and implement an out-of-band integration between `extras/kitex` and `devtools/inspector` to provide a dedicated "Components" tab in the DevTools UI. This tab will visualize the Virtual DOM tree (VDOM), component props, state hooks, and source-code origins, complete with cross-linking to physical bounds.

## Requirements

### 1. DevTools Extensions API (`devtools/inspector`)
- Introduce an `Extension` interface:
  ```go
  type Extension interface {
      Name() string
      GetPayload(eng *engine.Engine) any
  }
  ```
- Implement `RegisterExtension(ext Extension)` on the `Inspector`.
- Update `InspectorPayload` to include `Extensions map[string]any \`json:"extensions,omitempty"\``. The inspector should iterate through registered extensions on every snapshot to populate this map.

### 2. Kitex Introspection (`extras/kitex`)
- Implement a single exported snapshot generator: `func BuildDevToolsSnapshot() any`.
- This function runs internally within `kitex`, meaning it can access unexported fields across the tree without requiring any public `Inspect` methods on the nodes.
- It must recursively walk the `activeRoots` and build a JSON-serializable `VDOMSnapshot` tree holding:
  - `Name` and `Key`.
  - `Props` (the struct `PropsVal`).
  - `State` (raw hook values retrieved by safely interrogating `hookState`).
  - `DeclFile`/`DeclLine` and `InstFile`/`InstLine`.
  - The underlying `dom.Node` ID (if applicable).

### 3. Source Tracking
- Add a global `EnableDevMode bool` flag to `kitex`.
- Add unexported fields `declFile`, `declLine`, `instFile`, and `instLine` to `ComponentNode`.
- In `FC` and `FCC`: Use `runtime.Caller(1)` immediately to capture the declaration site (zero overhead).
- In the inner closure of `FC`/`FCC`: If `EnableDevMode` is true, use `runtime.Caller(1)` to capture the instantiation site (opt-in overhead).
- Read these private fields in `BuildDevToolsSnapshot()`.

### 4. Bridge Package (`extras/kitex/kitexdt`)
- Create a dedicated sub-package `extras/kitex/kitexdt` that imports both `devtools/inspector` and `extras/kitex`.
- It should expose a `Register(inspector *inspector.Inspector)` function.
- This function registers a `kitex` extension which just calls `kitex.BuildDevToolsSnapshot()` in its `GetPayload()`.

### 5. DevTools Frontend (`devtools/inspector/ui`)
- The Preact frontend should check for `payload.extensions.kitex`.
- If present, show a "Components" tab.
- Display the VDOM tree.
- Selecting a node opens a side panel displaying its Props (JSON viewer), State, and Source file locations.
- **Cross-linking:** Hovering over a component in the tree must leverage the same physical bounds overlay mechanism used by the Elements tab to highlight the component in the terminal. Add a button to jump to the corresponding raw node in the Elements tab.

## Verification
- Test that `BuildDevToolsSnapshot` safely handles all `kitex.Node` types (components, fragments, elements, nil).
- Verify that `kitex` compiles and functions perfectly without importing `devtools` directly (relying on `kitexdt` for the bridge).
- Validate the zero-overhead source tracking implementation.