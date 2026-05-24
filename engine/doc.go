// Package engine implements the kitex (kite v2) event loop, task queues,
// worker pool, and five-phase frame pipeline.
//
// Each frame runs dirty-gated phases in order: Tasks → Style → Layout →
// Paint → Sync. Hit-testing is not a phase; it is an on-demand query
// dispatched by the engine between frames.
//
// The engine supports multi-root rendering via the Document Overlay API.
// Each frame synchronizes the document's top-layer overlays into the render
// tree, ensures they are styled and laid out against the full viewport, and
// paints them sequentially above the main document flow.
//
// # Hardware Cursor
//
// The engine automatically manages the terminal's hardware cursor based on the
// currently focused node. If the focused node's render object implements the
// cursor.Provider interface, the engine translates its local cursor state to
// absolute screen coordinates and updates the terminal backend. This ensures
// that cursor management remains decoupled from logical element logic.
//
// // The engine coordinates all other
// kitex packages and is the only package that imports all of them.
//
// # Profiler
//
// Kite includes a built-in performance profiler based on a Hybrid Architecture
// (ADR-019). High-level engine phases are timed using a Pipeline decorator,
// while deep-tree granular timings are captured via an inlineable TraceContext
// injected into layout and paint phases. The profiler can be enabled via
// engine.WithProfiler(true) and exports data in the Chrome Trace Event Format.
package engine
