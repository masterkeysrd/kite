# ADR 021: Imperative Animation Architecture

## Status
Accepted

## Context
Kite needs a way to animate properties (like scroll offsets, layout geometries, or colors) to provide a modern, smooth user experience. However, a terminal is a discrete grid, meaning many physical layout animations (like width) are "stepped", while others (like color fading or logical scroll offsets) can interpolate smoothly.

The two main architectural paradigms are:
1. **Declarative (CSS-style transitions):** The style engine tracks previous states and automatically interpolates values when a style property changes.
2. **Imperative:** Developers manually request frame ticks and explicitly update properties over time.

## Decision
We chose a strictly **Imperative Animation Architecture**:
1. **Standalone Package:** A new `/animation` package will provide generic interpolators (e.g., `Tween[T]`) and easing functions. It has zero knowledge of the DOM, Style, or Layout systems.
2. **Engine Registry:** The `Engine` maintains a registry of active animations.
3. **Tick & Wakeup:** At the start of `engine.Frame()`, the engine calculates `dt` and ticks all registered animations. If animations remain active at the end of the frame, the engine calls `RequestFrame()` to keep itself awake (targeting 60FPS) without requiring a separate background goroutine.
4. **Stateless Style Engine:** The `style` package remains completely pure and stateless. To the rendering pipeline, animations just look like rapid, discrete property mutations.

## Consequences
- **Pros:** Keeps the core Style and Layout engines highly performant and stateless. Developers have explicit control over complex animations (like sequencing). Engine scheduling is highly efficient (sleeps when no animations are active).
- **Cons:** Developers must write slightly more code to animate a property compared to CSS `transition: all 0.2s`.