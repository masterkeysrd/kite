# Kite (v2) Design Roadmap

This document outlines the major architectural features and subsystems that still need to be designed and agreed upon in future sessions.

## 1. Drag and Drop (DND)
*   **Context:** Standard mouse interactions support `mousedown`, `mousemove`, and `mouseup`, but semantic dragging requires cross-component communication.
*   **Design Needs:**
    *   How does an element declare itself draggable?
    *   How do drop targets register themselves?
    *   Visual representation of the dragged item (potentially a ghost Overlay).

## 2. Check white-space collapsing
*  **Context:* We are currently collapsing all whitespace but what about If we want to conserve the original whitespace and print it as is?

## 5. Ecosystem Addons (External to Core Engine)
*   **Context:** Features that significantly enhance the developer experience but should remain outside the core engine to prevent bloat.
*   **Design Needs:**
    *   **Reactive Mini-Framework:** A React-like declarative UI layer built on top of the imperative Kite DOM for reactive state management.
    *   **Markdown / Rich Text Parser:** Converting Markdown strings into structured Kite DOM trees with styling.

## 6. Global Keyboard Shortcuts & Command Palette (`extras/commander`)
*   **Context:** Complex TUIs rely heavily on contextual keyboard shortcuts and discoverable action menus (like VS Code's Cmd+P).
*   **Design Needs:**
    *   A centralized Command Registry with default keybindings, descriptions, and dynamic enable/disable states.
    *   A built-in `<CommandPalette>` UI overlay for discovering and executing registered commands.

## 7. Theming & Design Tokens
*   **Context:** Hardcoding colors and margins doesn't scale well, especially when supporting light/dark modes or customizable color palettes.
*   **Design Needs:**
    *   A formalized Theme structure (colors, spacing, typography) that propagates through the component tree.
    *   Integration with the `kitex` Context system (e.g., `ThemeProvider`, `UseTheme`).

## 9. Form Management & Validation
*   **Context:** Wiring up forms with multiple fields, tracking "touched" state, handling validation, and managing submission state is tedious with low-level primitives.
*   **Design Needs:**
    *   A higher-level form wrapper to aggregate values from Kite form controls.
    *   Support for synchronous and asynchronous validation schemas.
