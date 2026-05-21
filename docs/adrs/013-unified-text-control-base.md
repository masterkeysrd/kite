# ADR 013: Unified Text Control Base

## Status
Accepted

## Context
As we developed `<input>` (single-line) and `<textarea>` (multi-line) elements, we encountered massive code duplication. Both components manage an internal `editor.Buffer`, an internal UA `uaDiv` block, and nearly identical implementations for:
- Caret positioning (`CursorState()`)
- Auto-scrolling (`ScrollCursorIntoView()`)
- Hit-testing and coordinate mapping (`handleMouseDown()`)
- Keyboard event routing (`handleKeyDown()`)

This duplication made maintaining the scroll state and caret logic highly error-prone. For instance, any fix to scroll offset translation or border-box inset mapping in one component had to be manually synced to the other. Furthermore, correctly tracking caret positions when layout constraints change or future scrollbar UIs are added requires a single, robust source of truth.

## Decision
We will extract the shared text-editing mechanics into a generic base struct, `textControlBase[T dom.Element]`, which both `InputElement` and `TextAreaElement` will compose (embed).

1. **Shared State**: The base manages the `*editor.Buffer`, the `doc` reference, the `uaDiv` block element, and scroll/focus state flags.
2. **Unified Geometry Math**: `CursorState()`, `ScrollCursorIntoView()`, and `handleMouseDown()` are centralized in the base. They rely entirely on the layout engine's output (`uaDiv.RenderObject().Fragment()`) and standard `dom.Element` scroll APIs (`Scroll()`, `ScrollTo()`).
3. **Multiline Configuration**: The base accepts an `isMultiline` boolean flag. This natively handles the behavioral differences (e.g., `<input>` ignoring `Enter` and Up/Down arrows vs `<textarea>` processing them) without needing completely separate implementations.
4. **Thin Wrappers**: `InputElement` and `TextAreaElement` act as thin wrappers that define specific intrinsic styles (e.g., `OverflowClip` vs `OverflowScroll`), construct their specific UA subtree topologies (e.g., injecting `<br>` for textareas), and provide a `syncCallback` to the base.

## Consequences
- **Positive:** Centralized, testable math for terminal coordinates. Complete immunity to scroll offset bugs caused by desynchronized implementations. Future features like scrollbars will "just work" because the base purely uses the physical layout fragment boundaries and standard `ScrollTo` API.
- **Negative:** Slightly increased complexity in element construction due to Go generics (`textControlBase[T]`) and callback wiring (`syncCallback`).
