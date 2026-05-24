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

## 3. CSS Grid Layout
*   **Context:** While Flexbox and Table handle 1D and data-driven 2D layouts, CSS Grid provides the ultimate structural control for complex 2D application dashboards.
*   **Design Needs:**
    *   Grid template rows/columns parsing and track sizing algorithms.

## 4. Text Selection & Clipboard API
*   **Context:** Native terminal selection (via mouse) often breaks or captures unwanted characters when selecting across complex flex/table layouts.
*   **Design Needs:**
    *   A logical DOM-level selection model.
    *   OS-level clipboard integration (Copy/Paste).

## 5. Ecosystem Addons (External to Core Engine)
*   **Context:** Features that significantly enhance the developer experience but should remain outside the core engine to prevent bloat.
*   **Design Needs:**
    *   **Reactive Mini-Framework:** A React-like declarative UI layer built on top of the imperative Kite DOM for reactive state management.
    *   **Routing System:** Managing multiple screens, views, and history navigation.
    *   **Markdown / Rich Text Parser:** Converting Markdown strings into structured Kite DOM trees with styling.
