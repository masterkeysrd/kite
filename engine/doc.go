// Package engine implements the kitex (kite v2) event loop, task queues,
// worker pool, and five-phase frame pipeline.
//
// Each frame runs dirty-gated phases in order: Tasks → Style → Layout →
// Paint → Sync. Hit-testing is not a phase; it is an on-demand query
// dispatched by the engine between frames.
//
// # Hardware Cursor
//
// The engine automatically manages the terminal's hardware cursor based on the
// currently focused node. If the focused node's render object implements the
// cursor.Provider interface, the engine translates its local cursor state to
// absolute screen coordinates and updates the terminal backend. This ensures
// that cursor management remains decoupled from logical element logic.
//
// The engine coordinates all other
// kitex packages and is the only package that imports all of them.
package engine
