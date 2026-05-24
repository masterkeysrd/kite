# TSK-042: Refactor Engine to use Pipeline Decorator and Inline TraceContext

## Context
As per ADR 019, Kite is adopting a Hybrid Profiler Architecture. We need to refactor the core `engine.Engine` frame loop to use a `Pipeline` interface. When profiling is enabled, the Engine will wrap this pipeline with a decorator to capture high-level phase durations. Furthermore, a `TraceContext` struct must be added to the layout context to support deep-tree granularity with zero allocations when disabled.

## Requirements
1. **Define `trace.Tracer`:** Create a new `trace` package containing the Chrome Trace Event formatting logic and the inlineable `Tracer` struct.
2. **Refactor Engine `Frame()`:** Extract the core layout, style, and paint phases from `Engine` into a new `engine.Pipeline` interface.
3. **Implement `ProfilingPipeline`:** Create a decorator that wraps `engine.Pipeline`. It should time the high-level phases and inject an active `*trace.Tracer` into the respective contexts (like `layout.Context`).
4. **Update Layout/Paint Contexts:** Add `Tracer *trace.Tracer` to `layout.Context` (or equivalent context structs) so that deep layout components like `Table` or `Flex` can call `defer ctx.Tracer.Begin("Layout(Table)")()`.
5. **Startup Profiling:** Add `engine.WithProfiler(true)` to engine options to enable the `ProfilingPipeline` immediately on startup, capturing the "First Time to Render".

## Constraints
- **Zero Allocation:** When profiling is disabled, `ctx.Tracer` must act as a Noop and compile away, causing zero heap allocations or significant overhead.
- **Trace Format:** Must export to standard Chrome Trace Event JSON.

## Tests
- Write a benchmark in `/engine` comparing `StandardPipeline` vs `ProfilingPipeline` to verify zero overhead when profiling is disabled.
- Add unit tests for the `trace` package to ensure the generated JSON structure is valid.

## Documentation Updates
- Update the package-level docs (`doc.go`) for `/engine` and `/layout` to mention the profiler trace context.