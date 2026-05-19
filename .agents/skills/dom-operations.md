---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: dom-operations
  description: Instructions for managing the logical DOM tree, implementing lifecycle hooks, and respecting the adoption registry (ADR-0036).
  displayName: DOM Operations
---

# DOM Operations Skill

When working within the `dom` package or manipulating the Kite document structure, you must adhere to the following strict rules:

## 1. The Logical Tree Only
The `dom` package models the logical node tree. It does **not** handle layout geometry, computed styles, or drawing. 
* DOM nodes (`dom.Node`, `dom.Element`) may carry a reference to a `render.Object` (`SetRenderObject`), but they do not execute rendering logic.
* **Semantic State:** The DOM owns interactive semantics. Properties like `Focusable()` and `Disabled()` belong to `dom.Element` or specific `dom.Focusable` interfaces, not to `render.Object`.

## 2. Synchronization Flags
The DOM manages structural lifecycle signals. Do not manually create or swap `render.Object`s inside elements.
* If you call `AppendChild()` or `RemoveChild()`, the DOM element must flag itself as `NeedsSync = true` and bubble `ChildNeedsSync = true` up to the root.
* The engine will pull this flag during the `Sync Phase` to create, prune, and rewire the render objects automatically.

## 2. Adoption and Element Identity (ADR-0036)
When a widget embeds a raw `*dom.element`, it must be properly "adopted".
* Every element carries an `outer` back-pointer. 
* This ensures that `event.Target()`, `GetElementByID()`, and `RenderObject.Node()` return the outermost user-visible wrapper, rather than the internal embedded element.
* **Never reset the `outer` pointer to `nil` on detach.** The element's identity remains stable even when disconnected.

## 3. Node Lifecycle Hooks
Types implementing the `Lifecycle` interface receive `OnConnected` and `OnDisconnected` callbacks.
* **OnConnected**: Fired in pre-order (parent before children) during the attach walk. `IsConnected()` will be true.
* **OnDisconnected**: Fired in post-order (children before parent) during the detach walk.
* **Constraint**: You may perform self- and descendant-mutations inside these callbacks. Ancestor-mutations will panic.

## Code Snippet Example
When creating a custom widget:
```go
type MyWidget struct {
    dom.Element
}

func NewMyWidget(doc dom.Document) *MyWidget {
    w := &MyWidget{
        Element: doc.CreateElement("my-widget"),
    }
    // The DOM engine will set 'outer' during the attach walk.
    return w
}
```
