# Package dom

Package `dom` implements the logical node tree for Kite. It is responsible for modeling the structure of the document and its semantic state.

## 🏛 Architecture

The `dom` package is strictly separated from layout and rendering logic:
1. **Logical Tree Only:** `dom` nodes model the hierarchy and semantic state (`Focusable()`, `Disabled()`). They do **not** contain layout algorithms or drawing logic.
2. **Synchronization Flags:** Changes to the DOM structure trigger synchronization flags (`NeedsSync`, `ChildNeedsSync`) that the engine uses to update the render tree during the Sync Phase.
3. **Element Identity (ADR-0036):** Every element carries an `outer` back-pointer (exposed as `self` in some implementations) to ensure that identity remains stable even when elements are wrapped by widgets.

## 🔑 Key Interfaces

### Node
The base interface for every node in the tree. It provides methods for tree traversal (`Parent`, `FirstChild`, `NextSibling`, etc.) and management of the associated `render.Object`.

### Element
Extends `Node` with identity attributes like `TagName` and `ID`.

### Disableable
Indicates that an element can be semantically disabled.
```go
type Disableable interface {
    IsDisabled() bool
    SetDisabled(bool)
}
```

### Focusable
Indicates that an element can receive keyboard focus.
```go
type Focusable interface {
    IsFocusable() bool
    Focus()
    Blur()
}
```

## 🔄 Lifecycle Hooks

Nodes can implement the `Lifecycle` interface to receive notifications when they enter or leave the live tree:
* **OnConnected:** Fired in pre-order during the attach walk.
* **OnDisconnected:** Fired in post-order during the detach walk.

## 🔗 Connection and Adoption

* **Connected State:** A node is "connected" when it is reachable from the `Document` root.
* **Adoption:** When a widget embeds an element, it must respect the adoption registry to ensure that event targets and lookups return the user-visible wrapper.
