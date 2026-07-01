# Kite (v2) Design Roadmap

This document outlines the major architectural features and subsystems of the Kite framework, tracking their design and implementation status.

## ⏳ Active / Planned Subsystems

### 1. Drag and Drop (DND)
*   **Context:** Standard mouse interactions support `mousedown`, `mousemove`, and `mouseup`, but semantic dragging requires cross-component communication.
*   **Design Needs:**
    *   How does an element declare itself draggable?
    *   How do drop targets register themselves?
    *   Visual representation of the dragged item (potentially a ghost Overlay).

### 2. Global Keyboard Shortcuts & Command Palette (`extras/commander`)
*   **Context:** Complex TUIs rely heavily on contextual keyboard shortcuts and discoverable action menus (like VS Code's Cmd+P).
*   **Design Needs:**
    *   A centralized Command Registry with default keybindings, descriptions, and dynamic enable/disable states.
    *   A built-in `<CommandPalette>` UI overlay for discovering and executing registered commands.

### 3. Theming & Design Tokens
*   **Context:** Hardcoding colors and margins doesn't scale well, especially when supporting light/dark modes or customizable color palettes.
*   **Design Needs:**
    *   A formalized Theme structure (colors, spacing, typography) that propagates through the component tree.
    *   Integration with the `kitex` Context system (e.g., `ThemeProvider`, `UseTheme`).

---

## ✅ Completed / Implemented Subsystems

### 1. Reactive Mini-Framework (`extras/kitex`)
*   **Status:** **[Completed]**
*   **Outcome:** A React-like VDOM framework built on top of the Kite DOM, supporting state/effect hooks, context propagation, ref binders, automatic memoization, and DevTools integration.

### 2. Form Management & Validation (`extras/form`)
*   **Status:** **[Completed]**
*   **Outcome:** High-level validation and state management wrapping the logical `<form>` and `FormControl` elements. Maps raw DOM form data to generic validator schemas.

### 3. Whitespace Collapsing & Caret/Selection
*   **Status:** **[Completed]**
*   **Outcome:** The text layout engine respects original vs. collapsed whitespace (`white-space: pre-wrap` vs `normal`), allowing selection/caret offsets to match the visual presentation accurately.

### 4. Caret & Spatial Focus Navigation (ADR-045)
*   **Status:** **[Completed]**
*   **Outcome:** Integrated character-level text selection carets with spatial focus transitions. Exposed logical APIs on `dom.Document` and `dom.FocusHandle` with zero-allocation layout version caching.
