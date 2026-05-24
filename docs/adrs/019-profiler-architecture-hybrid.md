# ADR 019: Profiler Architecture (Hybrid Approach)

## Status
Accepted

## Context
Kite needs a profiler to help developers identify performance bottlenecks in their terminal UI applications. Given Kite's strict multi-phase rendering pipeline and a 60FPS target, any tracing instrumentation must introduce zero overhead when disabled. The profiler needs to be granular enough to track both high-level engine phases (like Layout, Paint) and deep-tree metrics (like specific component layout algorithms). We also need a way to visualize the data, and capture "First Time to Render" on initialization.

The two standard approaches in Go are:
1. **Decorators/Interfaces**: Clean architecture but expensive for deep-tree profiling due to allocations and dynamic dispatch.
2. **Inlineable Structs**: Highly performant and allows deep-tree tracing with minimal overhead, but scatters timing logic throughout the codebase.

## Decision
We decided on a **Hybrid Approach** to profiling:

1. **Top-Level Decorator**: The core engine phases are abstracted into a `Pipeline` interface. A `ProfilingPipeline` decorator captures the duration of major phases cleanly without polluting `engine.Frame()`.
2. **Inlineable Struct for Deep Tracing**: A lightweight `TraceContext` struct is injected into context objects (like `LayoutContext`). Deep components can use it via `defer ctx.Tracer.Begin("Layout(Table)")()`. The compiler inlines these methods when profiling is disabled, resulting in zero allocations.
3. **Format**: The profiler outputs data in the Chrome Trace Event Format (JSON), tracking sync and async jobs.
4. **DevTools Integration**: Profiling is integrated into the DevTools, featuring a built-in Flamechart/Waterfall view to keep developers in the terminal environment, with an option to download the raw JSON for viewing in external tools like Perfetto.
5. **Startup Tracing**: Profiling can be activated on engine creation via `engine.WithProfiler(true)` to capture the application's initial mount and first render.

## Consequences
- **Performance**: Zero allocations or measurable overhead when profiling is disabled.
- **Maintainability**: High-level loops stay clean, while deep component instrumentation is localized to the components themselves.
- **Refactoring**: Requires extracting the engine phase loops into a new `Pipeline` interface and updating contexts to carry the `TraceContext`.