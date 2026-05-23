# Kite Project Instructions

Kite is a modern, high-performance DOM-like terminal UI framework for Go. It brings web-inspired development paradigms—including a logical DOM tree, CSS-style flexbox/block/table layouts, standard event propagation, and scrolling—to the terminal.

## 🏗 Project Overview

*   **Tech Stack:** Go 1.26.1+.
*   **Core Philosophy:** Strict separation of concerns across a multi-phase rendering pipeline.
*   **Performance:** Targets 60FPS on the main thread, with expensive operations handled via concurrent Jobs.

### Rendering Pipeline Phases
1.  **Input Buffering & Coalescing:** Collect and squash raw input into semantic events.
2.  **Job Sync:** Integrate completed async job results.
3.  **Synchronize Phase:** Project DOM changes into the render tree (flags dirty nodes).
4.  **Task Draining:** Execute macrotasks (budget-capped) and microtasks.
5.  **Style Phase:** Resolve the four-layer cascade into computed values.
6.  **Layout Phase:** Execute LayoutNG-inspired algorithms (Block, Flex, Inline, Table, List) to produce immutable physical fragment trees.
7.  **Paint Phase:** Draw fragments onto the framebuffer with clipping and border junction resolution.
8.  **Commit:** Push the framebuffer to the backend (e.g., `ultraviolet`).

## 📁 Core Subsystems

| Package | Responsibility |
| :--- | :--- |
| **`/dom`** | Logical node tree (`Document`, `Element`, `TextNode`), lifecycle, UA Shadow Subtrees (ADR-009). |
| **`/style`** | Sparse styling (`Optional[T]`), four-layer cascade resolution (Inherited < Default < Author < Intrinsic). |
| **`/layout`** | Geometry computations (Block, Flex, Inline, Table, List) producing `Fragment` trees. |
| **`/render`** | Visual bridge between DOM and Layout; tracks dirty flags. |
| **`/paint`** | Rasterization, clipping (ADR-011), and global border junction post-processing. |
| **`/engine`** | Central orchestrator of the frame loop and concurrent Jobs. |
| **`/event`** | Capture/Target/Bubble propagation and raw-to-semantic synthesis. |
| **`/focus`** | Logical focus management and spatial navigation (queries physical fragments). |
| **`/backend`** | Decoupling layer for terminal output (includes `mock` for tests). |
| **`/element`** | Declarative UI components (Box, Input, Table, List, Overlay, etc.). |

## 🚀 Building and Running

*   **Go Version:** 1.26.1 or higher.
*   **Test Suite:** `go test ./...`
*   **Benchmarks:** `go test -bench=. ./...` (Required for performance-sensitive changes in layout/style/paint).
*   **Devtools:** Enable via `devtools.Install(eng, options)` for web inspector and X-Ray mode.

## 🛠 Development Workflow

### 1. Consensus Mode (Design First)
*   **Design before Code:** Reach agreement on architectural changes before implementation.
*   **Artifacts:** Update `docs/architecture.md`, `docs/decisions.md`, and create/update ADRs in `docs/adrs/`.
*   **Tasks:** Generate detailed tasks in `./tasks/` (e.g., `TSK-001-feature.md`) and log them in `tasks/task_list.md`.

### 2. Coding Conventions
*   **Declarative UI:** Use `element` package constructors (e.g., `element.Box(...)`) for UI trees.
*   **Styling:** Use `style.Some(val)` for sparse definitions. Intrinsic UA styles belong in `IntrinsicStyle()`.
*   **Identity:** Respect the `outer` back-pointer for stable element identity.
*   **Iterators:** Use `iter.Seq[T]` for collection traversal (Go 1.24+ feature).
*   **Interface Guards:** Mandatory `var _ Interface = (*Concrete)(nil)` for all public implementors.

### 3. Testing Standards
*   **Table-Driven Tests:** Standard for unit and integration testing.
*   **Regression Tests:** Place in `tests/regressions/` with a header indicating the component and task/bug ID.
*   **Headless Testing:** Use `devtools/testenv` for DOM assertions and visual snapshots.
*   **Performance:** Any logic change in `/layout`, `/style`, or `/paint` MUST be benchmarked.

## 🧠 Agent Guidance

*   **Workflow Violation:** You MUST mark a task as `In Progress` in `tasks/task_list.md` before starting any work.
*   **Strict Verification:** NEVER assume Go types or signatures. Use grep/read to verify source code before documenting or implementing.
*   **No Batching:** Execute documentation updates (ADRs, etc.) as blockers before generating developer tasks.
*   **Source Map:** Reference `AGENT.md` for a detailed Concern → File Map before searching.
