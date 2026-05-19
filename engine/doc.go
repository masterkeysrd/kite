// Package engine implements the kitex (kite v2) event loop, task queues,
// worker pool, and five-phase frame pipeline.
//
// Each frame runs dirty-gated phases in order: Tasks → Style → Layout →
// Paint → Sync. Hit-testing is not a phase; it is an on-demand query
// dispatched by the engine between frames. The engine coordinates all other
// kitex packages and is the only package that imports all of them.
package engine
