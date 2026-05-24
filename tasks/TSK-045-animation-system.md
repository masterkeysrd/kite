# TSK-045: Implement Animation System and Engine Integration

## Context
Following ADR 021, we are introducing a new `/animation` package to handle imperative property interpolation, completely divorced from the `style` engine. The central `engine.Engine` will track and tick these animations during its frame loop.

## Requirements

### 1. The `/animation` Package
- Define an `Animator` interface with a `Tick(dt time.Duration) bool` method (returns true if the animation is finished).
- Implement standard easing functions (e.g., `Linear`, `EaseInQuad`, `EaseOutQuad`, `EaseInOutCubic`).
- Create a generic `Tween[T any]` struct that implements `Animator`.
- Define an `Interpolator[T any]` function signature: `func(start, end T, progress float64) T`.
- Provide standard interpolators:
    - `FloatInterpolator`
    - `IntInterpolator`
    - `ColorInterpolator` (interpolates the RGBA channels of a `style.Color`).

### 2. Engine Integration (`engine.go`)
- Add an `activeAnimations []animation.Animator` slice to `Engine`.
- Create a public `func (e *Engine) RegisterAnimation(anim animation.Animator)` method.
- **Tick Phase:** At the very top of `func (e *Engine) Frame()`, calculate the time elapsed (`dt`) since the last frame (using `e.clock.Now()`). Iterate backwards through `activeAnimations`, calling `Tick(dt)`. If an animation finishes, remove it from the slice.
- **Self-Scheduling:** At the bottom of `Frame()`, check if `len(e.activeAnimations) > 0`. If true, call `e.RequestFrame()` to guarantee the engine stays awake for the next frame tick.

## Tests
- Write unit tests in the `/animation` package verifying the interpolators and easing math.
- Write an engine test verifying that registering an animation causes `Frame()` to automatically invoke `RequestFrame()` until the animation duration expires.

## Documentation
- Update `engine/doc.go` to mention the animation tick phase.
- Write package-level documentation in `animation/doc.go` with an example of how to use a `Tween`.