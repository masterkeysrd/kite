# ADR 023: CSS Grid Layout and Animations

## Status
Accepted

## Context
We need to implement a CSS Grid Layout algorithm. Grid is inherently a two-dimensional, multi-pass layout system. To maintain the strict performance and caching guarantees of Kite's `layout` package (inspired by LayoutNG), the algorithm must avoid exponential complexity when measuring `auto` tracks or distributing free space.

Additionally, we need a mechanism to animate grid layouts (e.g., expanding sidebars) without introducing statefulness into the styling or layout engines.

## Decision

### 1. Phased Builder Architecture
We will implement Grid using a dedicated `GridBuilder` that separates matrix coordination from actual node layout.
1. **Track Sizing Pass:** The builder resolves all fixed, percentage, and fractional (`fr`) tracks. For `auto` tracks, it performs intrinsic measure passes (`ComputeMinMaxSizes`) on the children assigned to those tracks.
2. **Auto-Placement Pass:** Explicitly positioned items are placed first. A cursor then traverses the grid to auto-place remaining unassigned items into empty cells.
3. **Layout Pass:** The `GridAlgorithm` iterates over the placed items, creates precise `ConstraintSpace` boundaries based on the final track dimensions, and invokes standard `layout.Compute()`.

### 2. Scope Constraints (v1)
To keep the algorithm strictly linear for v1, we will defer the complex constraint-solving required by `minmax()`, `auto-fit`, and `subgrid`.
- **Included:** Fixed sizes, percentages, `fr`, `auto`, explicit placement, auto-placement, `repeat()` syntax, and `gap`.

### 3. Grid Animation via Generic Interpolators
We will not build a separate "GridAnimator" subsystem. We will leverage the existing stateless `animation.Tween[T]` system (ADR-021).
- We will provide an `animation.InterpolateGridTracks` function conforming to the `Interpolator[[]style.GridTrackSize]` signature.
- Developers instantiate a `Tween` with this interpolator. The engine ticks the tween, firing an `OnUpdate` callback that imperatively overwrites the element's `GridTemplateColumns` style, inherently triggering a layout invalidation on the next frame.

## Consequences
- **Positive:** Grid calculation remains isolated in a builder, preventing complex matrix state from leaking into immutable layout fragments.
- **Positive:** Animations reuse the highly generic Tween architecture, keeping the `style` and `layout` packages completely ignorant of time or interpolation frames.
- **Negative:** Omitting `minmax()` limits some advanced responsive grid patterns natively, requiring developers to use flexbox or media queries as workarounds until v2.