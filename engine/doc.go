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
// # Animations
//
// The engine supports an imperative animation system. Registered animations are
// tracked in an active registry and ticked at the very top of each frame loop.
// If any animations remain active at the end of the frame, the engine schedules
// another frame wake-up (targeting 60FPS) to keep the animation progressing. The
// animation tick phase is fully integrated with the built-in profiler.
//
// # Profiler
//
// Kite includes a built-in performance profiler based on a Hybrid Architecture
// (ADR-019). High-level engine phases are timed using a Pipeline decorator,
// while deep-tree granular timings are captured via an inlineable TraceContext
// injected into layout and paint phases. The profiler can be enabled via
// engine.WithProfiler(true) and exports data in the Chrome Trace Event Format.
//
// # Caret & Spatial Focus Navigation
//
// The engine coordinates caret-level character navigation and spatial focus jumps.
// When directional key events (up, down, left, right) are dispatched, they are
// routed to the active focused element's MoveCaret method if it implements
// [dom.SpatialCaret]. If the caret hits the text boundary, a spatial focus
// shift is triggered to focus the nearest logical element in that direction.
//
// Developers can trigger these focus and caret movements programmatically using
// the high-level Engine.MoveCaret(dir) and Engine.NavigateFocus(dir) methods.
package engine
