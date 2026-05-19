# Kite Render Package

The `render` package manages the **Visual Tree** (Render Tree) for Kite. It is the critical bridge layer that sits between the logical DOM (`dom` package) and the layout/paint pipelines (`layout` and `paint` packages).

## Intent and Philosophy

The DOM is an extremely lightweight, semantic data structure that knows nothing about CSS inheritance, physical sizes, or drawing. The layout engine is a strictly mathematical engine that operates on immutable constraints. The `render` package exists to glue them together.

1. **State Ownership:** The `render.Object` tree owns all "dirty" lifecycle flags (`DirtyStyle`, `DirtyLayout`, `DirtyPaint`).
2. **One-Way Data Flow:** The logical DOM "pushes" dirty notifications to the render tree. The render tree never modifies the logical DOM.
3. **Unified Display:** In Kite's architecture, there are only two core render objects: `render.Box` (which handles Block, Flex, and Grid dynamically via layout algorithms) and `render.Text`. The render object acts as a generic container for physical properties.
4. **Style Caching:** The render tree caches the fully-resolved `style.Computed` output, providing rapid, pre-calculated styling data to the layout and paint phases.

## Core Data Structures

*   **`Object`**: The fundamental interface for all nodes in the visual tree. It enforces tree navigation, dirty flag manipulation, and access to both logical nodes and computed styles.
*   **`BaseRender`**: The unexported struct that implements the vast majority of the `Object` interface. Specific render node types embed `BaseRender` to inherit tree-walking, fragment caching, and dirty-state logic without duplicating boilerplate.
*   **`Box`**: The universal render container for elements. It holds children, background/border styles, and delegates to the appropriate LayoutNG algorithm based on its computed `Display` property.
*   **`Text`**: A specialized render object for textual content. It cannot have children and is managed by the Inline layout algorithms.
*   **`RenderView`**: The absolute root of the render tree. It represents the physical dimensions of the terminal viewport and coordinates global overlays.
*   **`DirtyFlag`**: A bitmask system (e.g., `DirtyStyle`, `ChildNeedsLayout`) used to isolate and optimize the engine's phases, preventing unnecessary full-tree traversals.

## The Synchronization Lifecycle

The render tree must perfectly mirror the visible elements of the logical DOM. This synchronization happens seamlessly during the engine's frame loop through a "Push/Pull" model.

### 1. The Push (DOM Mutations)
When a developer interacts with the logical DOM (e.g., `el.SetStyle(...)` or `el.AppendChild(...)`), the DOM node does not instantiate render objects directly. Instead, it locates its currently attached `render.Object` and pushes a dirty flag.
* If a style changes, it calls `MarkDirty(DirtyStyle)`.
* If a child is added or removed, the parent calls `MarkDirty(DirtyStructure)`.

These flags instantly bubble up the render tree to the `RenderView` as "relay flags" (e.g., `ChildNeedsStyle`), establishing a highly optimized path for the engine to follow later.

### 2. The Pull (On-the-Fly Attachment)
During the engine's Style Phase (`style.ResolveTree`), the resolver walks top-down, skipping any subtrees that don't have `ChildNeedsStyle` or `ChildNeedsStructure`.

When it hits a node requiring structural work (a DOM node was added/removed), the walker performs an **On-the-Fly Sync**:
1. **Creation:** If a logical DOM node has no attached `render.Object`, the walker immediately instantiates a `render.Box` (or `Text`), attaches it, and pulls the raw style from the DOM.
2. **Destruction (Display: None):** If the computed style resolves to `display: none`, the walker unlinks and destroys the existing `render.Object`. The subtree is ignored.
3. **Pruning:** If the DOM node was removed entirely, the walker detects the orphaned `render.Box`es and completely unlinks that visual subtree.

### 3. Layout & Paint Handoff
Once the render tree is structurally sound and fully styled, the layout engine takes over. The `render.Object` provides the `layout.Node` interface required by LayoutNG. 

When layout completes, the engine calls `renderObject.SetCachedLayout(frag)`. If the new `Fragment` pointer differs from the previous one (meaning physical sizes actually changed), the `render` package automatically flags `DirtyPaint`, queuing it for the final terminal draw.
