// Package animation provides utilities for imperative property interpolation and tweening.
//
// It defines the [Animator] interface for time-based updates and a generic [Tween]
// implementation that can interpolate any type using a provided [Interpolator].
//
// Standard easing functions are provided: [Linear], [EaseInQuad], [EaseOutQuad],
// and [EaseInOutCubic].
//
// Standard interpolators are provided: [FloatInterpolator], [IntInterpolator],
// and [ColorInterpolator].
//
// Tweening example:
//
//	tween := animation.NewTween(0, 100, 1*time.Second, animation.EaseInOutCubic, animation.IntInterpolator, func(val int) {
//		s := element.RawStyle()
//		s.Width = style.Some(style.Cells(val))
//		element.Style(s)
//	})
//	engine.RegisterAnimation(tween)
package animation
