# AI Agent Rules & Context for Kite (v2)

This document provides guidelines and architectural context for AI assistants and coding agents operating within the Kite repository.

## 🧠 System Context

*   **Project Purpose:** A web-like Terminal UI framework that uses a DOM, Flexbox layout, and standard event propagation to render rich TUIs.
*   **Tech Stack:** Go 1.26.1.
*   **Key Dependencies:** Charmbracelet ecosystem (`github.com/charmbracelet/ultraviolet`, `github.com/charmbracelet/colorprofile`), and `github.com/rivo/uniseg` for text shaping.
*   **Database/Storage:** None. The project operates purely in-memory.

## 🏛 Architectural Rules

1.  **Strict Package Isolation:**
    *   The `/dom` package models the logical tree only. It **must not** contain layout algorithms, computed styles, or drawing logic. 
    *   The `/style` package has **no dependencies** on other Kitex packages. Keep it isolated.
    *   State bridging happens via `/render` objects. DOM nodes optionally point to a `render.Object` (`ADR-0002`, `ADR-0003`), but they do not own the rendering lifecycle.
2.  **Element Identity & Adoption (ADR-0036):**
    *   Every `dom.Element` carries an `outer` back-pointer. This pointer ensures that when widgets wrap standard elements, functions like `event.Target()`, `GetElementByID()`, and `RenderObject.Node()` always return the outermost, user-visible wrapper.
    *   Do not reset the `outer` pointer to `nil` on detach. The identity must remain stable.
3.  **Styling Paradigm:**
    *   Always use the `Optional[T]` wrapper (e.g., `style.Some(val)`) when defining properties in `style.Style`. This distinguishes between a field that is explicitly unset versus a zero-value.
    *   The bridge to the rendering layer is `style.Computed`, which contains raw values (no optionals) after the resolver applies inheritance.
4.  **Event Bubbling:**
    *   Events must strictly follow the Capture -> Target -> Bubble sequence. 
    *   Avoid introducing "IntentEvents" (a deprecated concept from v1). Rely on the `Synthesizer` to convert raw inputs into semantic events.

## 🧑‍💻 Coding Conventions

1.  **Continuous Documentation Maintenance:**
    *   **Always** keep `README.md` and `AGENT.md` up-to-date. If you introduce new packages, modify core architectural patterns, or change significant dependencies, you must update these files to reflect the new state of the project.
2.  **Modern Go Features:** 
    *   Utilize Go 1.24+ standard library features.
    *   Use iterators (`iter.Seq[T]`) for traversing collections, such as `Node.Children()`.
3.  **Interfaces and Embedding:** 
    *   Favor small, composable interfaces (e.g., `dom.Node`, `dom.Element`, `dom.TextNode`).
    *   When creating internal implementations, use unexported structs (e.g., `element`) and assert compile-time interface compliance (`var _ Element = (*element)(nil)`).
4.  **Documentation:** 
    *   All packages must contain a `doc.go` file summarizing the package's responsibility.
    *   Reference ADRs (Architecture Decision Records) in docstrings when touching core mechanics (e.g., `ADR-0036` for DOM adoption).

## 🧪 Testing Strategy

1.  **Table-Driven Tests:** Prefer table-driven structures using the standard `testing` package.
2.  **Mocking:** Use the `backend/mock` package for testing the Render and Paint pipelines without requiring a physical TTY or terminal emulator.
3.  **Benchmarks:** Any changes to `/layout`, `/style` resolving, or `/paint` logic must be accompanied by `testing.B` benchmarks, as performance is critical in a 60FPS UI loop.
4.  **No Panics:** Ensure test assertions do not result in raw panics. Handled disconnected/nil states gracefully in DOM manipulation tests.
