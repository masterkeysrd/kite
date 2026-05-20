# Kite (v2) Design Roadmap

This document outlines the major architectural features and subsystems that still need to be designed and agreed upon in future sessions.

## 1. Scroll Containers & Viewport Clipping
*   **Context:** While we have `OverflowScroll` and `OverflowHidden` in the style enums, the layout and paint pipelines do not yet support true scrollable containers.
*   **Design Needs:**
    *   How `Fragments` manage and store their internal `scrollOffset`.
    *   How the `PaintEngine` applies strict bounding-box clipping when drawing children of an `overflow: hidden` element.
    *   How the `element.ScrollBox` component operates.
    *   How `EventWheel` (mouse scrolling) interacts and bubbles up the DOM tree until it hits a scrollable container.

## 2. Interactive Components (Forms & Buttons)
*   **Context:** We have designed text inputs, but a standard UI framework requires a comprehensive set of controls.
*   **Design Needs:**
    *   `Button`: Should it be a specialized component or just a `Box` with standard focus semantics (e.g., `Enter`/`Space` triggering a `click` event)?
    *   `Checkbox` / `Toggle`.
    *   `RadioGroup`: Requires shared state management to ensure mutual exclusivity.
    *   `Select` (Dropdown): Will heavily utilize the new `Overlay` system designed in ADR-008.

## 3. Padding and Margin Math (Box Model Polish)
*   **Context:** Kite relies on `layout.BlockAlgorithm` for sizing, but standard CSS compliance requires strict rules.
*   **Design Needs:**
    *   Do we support **Margin Collapsing** inside Block Formatting Contexts, or do we strictly enforce physical spacing without collapsing (to keep terminal math simple)?
    *   *User Note:* User indicated preference for a strict `BorderBox` model (no margin merging) because TUIs value simplicity and predictable grids. This needs to be formalized in an ADR and task.

## 4. Animations & Transitions
*   **Context:** UI states change instantly.
*   **Design Needs:**
    *   A subsystem to transition styles (e.g., color fading, width expanding) smoothly over time.
    *   Integration with `Engine.Clock` to schedule micro-frames over a set duration and force `DirtyStyle` updates.

## 5. Drag and Drop (DND)
*   **Context:** Standard mouse interactions support `mousedown`, `mousemove`, and `mouseup`, but semantic dragging requires cross-component communication.
*   **Design Needs:**
    *   How does an element declare itself draggable?
    *   How do drop targets register themselves?
    *   Visual representation of the dragged item (potentially a ghost Overlay).
