# Package animation

Package `animation` provides a generic, imperative framework for property interpolation and tweening within the Kite framework. It allows for smooth transitions of numeric values, colors, and other types over time using customizable easing functions.

## 🚀 Key Concepts

### Animator
The core interface for any time-based animation.
```go
type Animator interface {
    // Tick updates the animation state by the given delta time.
    // Returns true if the animation has finished.
    Tick(dt time.Duration) bool
}
```

### Tween
A `Tween[T]` is a concrete implementation of `Animator` that interpolates a value of type `T` from a `Start` value to an `End` value over a specific `Duration`.

### Easing Functions
Easing functions control the rate of change of a parameter over time. The package includes several common easing functions:
- `Linear`
- `EaseInQuad`
- `EaseOutQuad`
- `EaseInOutCubic`

### Interpolators
Interpolators define how to calculate a value between two points given a progress value (`0.0` to `1.0`).
- `FloatInterpolator`: For `float64` values.
- `IntInterpolator`: For `int` values.
- `ColorInterpolator`: For `image/color.Color` (handles alpha-premultiplied RGBA).

## 🛠 Usage Example

To run an animation, you typically create a `Tween` and register it with the `engine`. The `engine` will then call `Tick` on each frame.

```go
// Create a tween that animates an element's width from 0 to 100 pixels over 1 second.
tween := animation.NewTween(
    0,                                // Start
    100,                              // End
    1 * time.Second,                  // Duration
    animation.EaseInOutCubic,          // Easing
    animation.IntInterpolator,         // Interpolator
    func(val int) {                   // OnUpdate callback
        s := element.RawStyle()
        s.Width = style.Some(style.Cells(val))
        element.Style(s)
    },
)

// Register with the engine
engine.RegisterAnimation(tween)
```

## 🔄 Integration with Kite Engine

The `engine` package provides a mechanism to register and manage `Animator` instances. During each frame pipeline, the engine's internal loop calls `Tick` with the elapsed time since the last frame. When `Tick` returns `true`, the engine automatically removes the animation from its registry.
