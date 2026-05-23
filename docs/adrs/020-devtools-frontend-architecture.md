# ADR 020: DevTools Frontend Architecture (Preact + Vite)

## Status
Accepted

## Context
The Kite DevTools Inspector currently uses a plain `index.html` file with vanilla JavaScript to display a live DOM tree via Server-Sent Events (SSE). With the addition of the new Profiler subsystem (ADR 019), we need to build an interactive Flamechart UI. Building and maintaining complex data visualizations like a flamechart (which requires precise absolute positioning, panning, zooming, and state management) in vanilla JavaScript is highly error-prone and scales poorly.

However, Kite is a Go project. We want to avoid forcing end-users or Go contributors to install Node.js just to run or import the DevTools package.

## Decision
We will migrate the DevTools frontend to use **Preact** (a lightweight 3kB alternative to React) bundled with **Vite**.

1. **Component Architecture:** We will use Preact to build a modular UI (e.g., `<DOMTree>`, `<Flamechart>`, `<NodeDetails>`).
2. **Single-File Output:** We will configure Vite (`vite-plugin-singlefile`) to inline all JavaScript and CSS directly into the final `index.html` output.
3. **Go Embedding:** The generated `index.html` will be placed in `devtools/inspector/ui/dist/` and served via Go's `//go:embed`.
4. **Developer Experience:** Contributors working on the DevTools UI will use Node.js to run the Vite dev server with Hot Module Replacement (HMR) for rapid prototyping. Normal Go developers building Kite apps will just use the pre-compiled embedded asset and will never need Node.js.

## Consequences
- **Pros:** Massively improved maintainability for the frontend UI. Building the Flamechart will be significantly easier. Hot Module Replacement (HMR) will speed up UI development.
- **Cons:** Introduces Node.js and NPM as a dependency *only* for contributors who want to modify the DevTools UI itself.
- **Mitigation:** We must ensure the compiled `index.html` is committed to the repository so standard Go users can run `go get` without needing to build the UI themselves.