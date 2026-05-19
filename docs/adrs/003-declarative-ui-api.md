# ADR 003: Declarative UI API and DOM Adoption

## Status
Accepted

## Context
Developers currently build Kite UI trees using procedural, imperative code (`doc.CreateElement`, `parent.AppendChild`). This is verbose and lacks the ergonomic flow of modern UI frameworks (like SwiftUI or React). We want to support a declarative syntax:

```go
Box(
   OrderedList(
       ListItem("Go"),
       ListItem("Python"),
   )
)
```

To achieve this, constructors must not require a `dom.Document` parameter to be passed explicitly to every node, and they must accept variadic children `...any` to allow nested composition and auto-boxing of strings.

However, Kite's `dom` package enforces strict document ownership (`ownerDocument`), and cross-document appends currently panic. Generating nodes without the final engine document attached violates this rule.

## Decision

### 1. Implicit DOM Adoption
We will modify the `dom` package to support **implicit document adoption** for detached subtrees.
- When `parent.AppendChild(child)` is called, the DOM checks if `child.OwnerDocument() == parent.OwnerDocument()`.
- If the child belongs to a different document **but is detached** (`!child.IsConnected()`), the DOM will recursively walk the child's subtree and update the `ownerDocument` pointers to match the parent's document.
- If the child is connected to an active document, the engine will still panic. This protects against accidentally moving live nodes between different engine instances.

### 2. Declarative API in `/element`
- **Type Renaming:** Existing structs in the `element` package will be suffixed with `Element` (e.g., `Box` becomes `BoxElement`, `Text` becomes `TextElement`). This frees up the bare names for function constructors.
- **Constructors:** We will introduce functional constructors (e.g., `func Box(children ...any) *BoxElement`).
- **Scratch Document:** Internally, these constructors will generate nodes using a lightweight "scratch" or "orphan" document instance. When the user finally mounts the top-level declarative block to the engine (`engine.Mount(root)`), the entire tree will be implicitly adopted by the engine's main document.
- **Variadic Children:** The constructors will process `...any`:
  - `string` is automatically converted to `TextElement`.
  - `dom.Node` is appended directly.
  - Slices/Arrays are flattened and processed recursively.

## Consequences

### Positive
- **Developer Experience:** Radically improves the ergonomics of building complex terminal layouts in Go.
- **Flexibility:** Allows developers to compose detached widgets in separate functions/packages without needing to thread a `dom.Document` reference everywhere.

### Negative / Trade-offs
- **$O(N)$ Adoption Walk:** Appending a massive detached subtree to the main document will incur an $O(N)$ recursive walk to update the `ownerDocument` pointers. Given typical terminal UI tree depths, this overhead is negligible, especially since it only happens once during initial mount.
