# Task: Grid Animation Interpolator

## Description
Enable animation of Grid track sizes by implementing a generic interpolator for the `animation` package.

## Requirements
- In the `animation` package, implement:
  `func InterpolateGridTracks(start, end []style.GridTrackSize, progress float64) []style.GridTrackSize`
- **Interpolation Logic**:
  - Iterate through the tracks matching by index.
  - If `start` and `end` tracks are of the same `Kind` (e.g., both Fractional, or both Cells), mathematically interpolate their values based on `progress`.
  - If they are of different kinds (e.g., animating from `auto` to `1fr`), implement a snap behavior (e.g., switch to `end` at 50% progress) since you cannot smoothly lerp discrete units in this architecture.
  - Handle mismatched slice lengths safely.

## Tests
- Write unit tests in `animation/animation_test.go`.
- Assert that interpolating `[1fr]` to `[3fr]` at `0.5` progress yields `[2fr]`.
- Ensure it does not panic if slice lengths differ.