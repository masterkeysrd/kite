# TSK-043: Implement DevTools Profiler Endpoints and Flamechart UI

## Context
Following ADR 019, Kite developers need a way to view profiling data. The DevTools server needs to be extended to toggle the profiler, serve the trace data, and provide an internal UI for flamecharts.

## Requirements
1. **HTTP Endpoints:** Add `/debug/trace/start` and `/debug/trace/stop` to the DevTools Inspector.
    - `start` should toggle the engine's active pipeline to the `ProfilingPipeline` and start recording.
    - `stop` should halt recording, revert to the standard pipeline, and return the accumulated JSON trace.
2. **Download Feature:** Allow downloading the trace JSON directly from the DevTools UI so developers can inspect it in `chrome://tracing` or Perfetto.
3. **Internal Waterfall UI:** Build a basic Flamechart/Waterfall view directly into the web-based DevTools DOM Inspector interface. It should read the JSON trace and render the main phases (and any recorded inline deep events) as absolute-positioned visual blocks.
4. **Async Jobs Mapping:** Ensure the UI correctly visually maps asynchronous Jobs back to their initiating main-thread frame.

## Constraints
- The UI should be pure HTML/CSS/JS (vanilla or lightweight framework) added to the existing DevTools frontend.
- Do not introduce heavy third-party graphing libraries unless absolutely necessary; basic HTML elements for the flamechart are preferred.

## Tests
- Headless testing using `/devtools/testenv` to verify the HTTP endpoints successfully return valid JSON traces when profiling is toggled.

## Documentation Updates
- Update `README.md` to show developers how to use the new DevTools profiling commands and endpoints.