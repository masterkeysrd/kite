# Developer Task List

This document tracks the status of all generated developer tasks for Kite (v2). 

## Rules
1. **Status Progression:** Tasks move from `Pending` -> `In Progress` -> `Done`.
2. **Start Work:** When you begin a task, change its status below to **`In Progress`**. Do not start a task unless it is currently `Pending`.
3. **Completion:** You may **only** change a status to **`Done`** after the Lead Designer (the User) has explicitly confirmed the task is completed and verified. Do not mark a task as done prematurely.
4. **Immutability of Done:** Do not modify tasks or their corresponding Markdown files once they are marked as **`Done`**.

---

## Tasks

| Task ID | Description | Status | Markdown File |
|---------|-------------|--------|---------------|
| TSK-001 | Implement List Layout Algorithm (Virtual Markers) | Done | `tasks/TSK-001-feature-list-layout.md` |
| TSK-002 | Implement List and ListItem DOM Components | Done | `tasks/TSK-002-feature-list-components.md` |
| TSK-003 | Implement Table Layout Algorithm | Done | `tasks/TSK-003-table-layout-algorithm.md` |
| TSK-004 | Implement Table DOM Components and Fault Tolerance | Done | `tasks/TSK-004-table-components-fault-tolerance.md` |
| TSK-005 | Implement Implicit DOM Adoption | Done | `tasks/TSK-005-dom-implicit-adoption.md` |
| TSK-006 | Implement Declarative API for Elements | Done | `tasks/TSK-006-declarative-element-api.md` |
| TSK-007 | Move Styling State to Logical DOM | Done | `tasks/TSK-007-dom-owned-styles.md` |
| TSK-008 | Table Section Grouping (thead, tbody, tfoot) | Done | `tasks/TSK-008-table-grouping.md` |
| TSK-009 | Border Rendering Engine and Fluent API | Done | `tasks/TSK-009-border-rendering-devx.md` |
| TSK-010 | Screen-Space Border Intersection Resolver | Done | `tasks/TSK-010-border-post-processor.md` |
| TSK-012 | Table Layout Builder Pattern | Done | `tasks/TSK-012-table-layout-builder.md` |
| TSK-013 | Flex Layout Builder Refactor | Done | `tasks/TSK-013-flex-layout-builder.md` |
| TSK-014 | Global Border Style Metadata | Done | `tasks/TSK-014-border-style-metadata.md` |
| TSK-015 | Cursor Package and Engine Integration | Done | `tasks/TSK-015-cursor-package.md` |
| TSK-016 | Custom Render Object Hook | Done | `tasks/TSK-016-custom-render-object-hook.md` |
| TSK-017 | Logical Text Controller | Done | `tasks/TSK-017-logical-text-controller.md` |
| TSK-018 | UA Shadow Subtree Primitive (supersedes prior Input/TextArea task) | Done | `tasks/TSK-018-ua-shadow-subtree-primitive.md` |
| TSK-019 | Document Overlay API and Render Root | Done | `tasks/TSK-019-document-overlay-api.md` |
| TSK-020 | Element Bounding Client Rect | Done | `tasks/TSK-020-element-bounding-client-rect.md` |
| TSK-021 | Overlay and Dialog Components | Done | `tasks/TSK-021-overlay-and-dialog-components.md` |
| TSK-022 | Intrinsic Style Layer | Done | `tasks/TSK-022-intrinsic-style-layer.md` |
| TSK-023 | cursor.FromTextFragment Helper | Done | `tasks/TSK-023-cursor-from-text-fragment.md` |
| TSK-024 | Implement `<input>` onto UA Shadow Subtree | Done | `tasks/TSK-024-input-on-ua-subtree.md` |
| TSK-025 | Implement `<textarea>` onto UA Shadow Subtree | Done | `tasks/TSK-025-textarea-on-ua-subtree.md` |
| TSK-026 | IFC Honors `overflow-wrap` (deferred IFC cleanup) | Done | `tasks/TSK-026-ifc-honors-overflow-wrap.md` |
| TSK-027 | Paint Honors `overflow: clip` / `overflow: hidden` | Done | `tasks/TSK-027-paint-overflow-clipping.md` |
| TSK-028 | Generic Scroll Offset on DOM Elements | Done | `tasks/TSK-028-generic-scroll-offset.md` |
| TSK-029 | Unified Text Control Base | Done | `tasks/TSK-029-unified-text-control-base.md` |
| TSK-030 | Implement Headless Test Environment (`testenv`) | Done | `tasks/TSK-030-headless-testenv.md` |
| TSK-031 | Implement Golden Testing & Visual Dumps | Done | `tasks/TSK-031-golden-testing.md` |
| TSK-032 | Implement Web-Based DOM Inspector (SSE) | Done | `tasks/TSK-032-web-inspector.md` |
| TSK-033 | Implement Terminal X-Ray Mode | Done | `tasks/TSK-033-xray-mode.md` |
| TSK-034 | Event Coalescing and Throttling in Engine | Done | `tasks/TSK-034-event-coalescing.md` |
| TSK-035 | Deferred Scroll & Cursor Rendering for Text Controls | Done | `tasks/TSK-035-deferred-scroll-rendering.md` |
| TSK-036 | Customizable Visual Scrollbars | Done | `tasks/TSK-036-visual-scrollbars.md` |
| TSK-037 | Implement Button Element | Done | `tasks/TSK-037-button-element.md` |
| TSK-038 | Implement Checkbox and Radio Components | Done | `tasks/TSK-038-checkbox-radio-elements.md` |
| TSK-039 | Implement Select (Dropdown) Component | Done | `tasks/TSK-039-select-element.md` |
| TSK-040 | Audit and Enforce Strict Border-Box Sizing | Done | `tasks/TSK-040-strict-box-model.md` |
| TSK-041 | Introduce ContainingSpace and ContainerSpace into Layout | Done | `tasks/TSK-041-layout-container-spaces.md` |
| TSK-042 | Refactor Engine to use Pipeline Decorator and Inline TraceContext | Done | `tasks/TSK-042-profiler-pipeline-decorator.md` |
| TSK-043 | Implement DevTools Profiler Endpoints and Flamechart UI | Done | `tasks/TSK-043-devtools-flamechart-ui.md` |
| TSK-044 | Migrate DevTools Frontend to Preact and Vite | Done | `tasks/TSK-044-devtools-frontend-preact.md` |
| TSK-045 | Implement Animation System and Engine Integration | Done | `tasks/TSK-045-animation-system.md` |
| TSK-046 | Logical Text Selection (DOM) | Done | `tasks/TSK-046-logical-text-selection.md` |
| TSK-047 | Paint Masking for Text Selection | Done | `tasks/TSK-047-paint-selection-mask.md` |
| TSK-048 | User Interaction and Hit-Testing for Selection | Done | `tasks/TSK-048-selection-hit-testing.md` |
| TSK-049 | System Clipboard Integration | Done | `tasks/TSK-049-clipboard-integration.md` |
| TSK-050 | Text Control Local Selection | Done | `tasks/TSK-050-text-control-selection.md` |
| TSK-051 | Text Control Clipboard Mechanics | Done | `tasks/TSK-051-text-control-clipboard.md` |
| TSK-052 | CSS Grid Style API | Done | `tasks/TSK-052-grid-style-api.md` |
| TSK-053 | Grid Builder and Auto-Placement | Done | `tasks/TSK-053-grid-builder-placement.md` |
| TSK-054 | Grid Layout Algorithm | Done | `tasks/TSK-054-grid-algorithm.md` |
| TSK-055 | Grid Animation Interpolator | Done | `tasks/TSK-055-grid-animator.md` |
| TSK-056 | Kitex VDOM Primitive Factories | Done | `tasks/TSK-056-kitex-vdom-primitives.md` |
| TSK-057 | Kitex FC & Implicit Hooks Context | Done | `tasks/TSK-057-kitex-hooks-context.md` |
| TSK-058 | Kitex VDOM Reconciler Engine | Done | `tasks/TSK-058-kitex-reconciler.md` |
| TSK-059 | Kitex Ref Standardization | Todo | `tasks/TSK-059-kitex-ref-standardization.md` |
| TSK-060 | Kitex DevTools Integration | Todo | `tasks/TSK-060-kitex-devtools-integration.md` |
| TSK-061 | Kitex Automatic Component Memoization | Todo | `tasks/TSK-061-kitex-memoization.md` |
