# Kite (v2) Design Roadmap

This document outlines the major architectural features and subsystems that still need to be designed and agreed upon in future sessions.

## 1. Interactive Components (Forms & Buttons)
*   **Context:** We have designed text inputs and buttons, but a standard UI framework requires a comprehensive set of controls.
*   **Design Needs:**
    *   ~~`Button`~~ (Implemented in TSK-037).
    *   `Checkbox` / `Toggle`.
    *   `RadioGroup`: Requires shared state management to ensure mutual exclusivity.
    *   `Select` (Dropdown): Will heavily utilize the new `Overlay` system designed in ADR-008.

## 2. Animations & Transitions
*   **Context:** UI states change instantly.
*   **Design Needs:**
    *   A subsystem to transition styles (e.g., color fading, width expanding) smoothly over time.
    *   Integration with `Engine.Clock` to schedule micro-frames over a set duration and force `DirtyStyle` updates.

## 3. Drag and Drop (DND)
*   **Context:** Standard mouse interactions support `mousedown`, `mousemove`, and `mouseup`, but semantic dragging requires cross-component communication.
*   **Design Needs:**
    *   How does an element declare itself draggable?
    *   How do drop targets register themselves?
    *   Visual representation of the dragged item (potentially a ghost Overlay).

## 4. Check white-space collapsing
*  **Context:* We are currently collapsing all whitespace but what about If we want to conserve the original whitespace and print it as is?
