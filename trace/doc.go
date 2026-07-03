// Package trace provides Chrome Trace Event format instrumentation for the
// Kite engine. Events are captured as a simple event list and written as a
// JSON array consumable by chrome://tracing.
//
// # Tracer
//
// Tracer captures named trace events with microsecond timestamps. Each
// event is a pair of Begin/End markers that chrome://tracing renders as a
// horizontal bar on a timeline.
//
// # Usage
//
// The Begin method returns a cleanup function suitable for defer:
//
//	tracer := trace.NewTracer()
//	defer tracer.WriteJSON(os.Stdout)
//
//	func() {
//	     defer tracer.Begin("Layout")()
//	     // ... layout work ...
//	}()
//
// BeginThread(name, tid) scopes events to a specific thread ID.
// BeginWithArgs(name, args) attaches a map of additional arguments.
//
// # Nil Safety
//
// A nil *Tracer is safe — all methods return Noop() which is a no-op
// function, allowing trace calls to be left in production code without
// overhead.
package trace
