package animation

import (
	"image/color"
	"time"

	"github.com/masterkeysrd/kite/style"
)

// Animator represents an animation that can be ticked over time.
type Animator interface {
	// Tick updates the animation state by the given delta time.
	// Returns true if the animation has finished.
	Tick(dt time.Duration) bool
}

// Ensure *Tween[int] implements Animator.
var _ Animator = (*Tween[int])(nil)

// EasingFunction maps progress in the range [0.0, 1.0] to an eased progress value in [0.0, 1.0].
type EasingFunction func(t float64) float64

// Linear is a straight linear easing.
func Linear(t float64) float64 {
	return t
}

// EaseInQuad is a quadratic ease-in.
func EaseInQuad(t float64) float64 {
	return t * t
}

// EaseOutQuad is a quadratic ease-out.
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutCubic is a cubic ease-in-ease-out.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	f := -2*t + 2
	return 1 - f*f*f/2
}

// Interpolator defines the function signature for interpolating between two values of type T.
type Interpolator[T any] func(start, end T, progress float64) T

// FloatInterpolator interpolates between two float64 values.
func FloatInterpolator(start, end float64, progress float64) float64 {
	return start + (end-start)*progress
}

// IntInterpolator interpolates between two int values.
func IntInterpolator(start, end int, progress float64) int {
	return int(float64(start) + float64(end-start)*progress)
}

// ColorInterpolator interpolates the alpha-premultiplied RGBA channels of image/color.Color.
func ColorInterpolator(start, end color.Color, progress float64) color.Color {
	r1, g1, b1, a1 := start.RGBA()
	r2, g2, b2, a2 := end.RGBA()

	r := uint32(float64(r1) + float64(int32(r2)-int32(r1))*progress)
	g := uint32(float64(g1) + float64(int32(g2)-int32(g1))*progress)
	b := uint32(float64(b1) + float64(int32(b2)-int32(b1))*progress)
	a := uint32(float64(a1) + float64(int32(a2)-int32(a1))*progress)

	return color.RGBA64{
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a),
	}
}

// Tween implements Animator, performing interpolation between a start and end value over a duration.
type Tween[T any] struct {
	Start    T
	End      T
	Duration time.Duration
	Easing   EasingFunction
	Interp   Interpolator[T]
	OnUpdate func(current T)

	elapsed time.Duration
}

// NewTween creates a new Tween animator.
func NewTween[T any](start, end T, duration time.Duration, easing EasingFunction, interp Interpolator[T], onUpdate func(current T)) *Tween[T] {
	return &Tween[T]{
		Start:    start,
		End:      end,
		Duration: duration,
		Easing:   easing,
		Interp:   interp,
		OnUpdate: onUpdate,
	}
}

// Tick ticks the animation by dt. Returns true if the animation is finished.
func (t *Tween[T]) Tick(dt time.Duration) bool {
	t.elapsed += dt
	if t.Duration <= 0 {
		t.OnUpdate(t.End)
		return true
	}

	progress := float64(t.elapsed) / float64(t.Duration)
	if progress >= 1.0 {
		t.OnUpdate(t.End)
		return true
	}

	eased := progress
	if t.Easing != nil {
		eased = t.Easing(progress)
	}

	current := t.Interp(t.Start, t.End, eased)
	t.OnUpdate(current)
	return false
}

// InterpolateGridTracks interpolates between two slices of GridTrackSize.
// It iterates through the tracks matching by index.
// If start and end tracks are of the same Kind, their values are interpolated based on progress.
// If they are of different kinds, or if the slice lengths differ, it snaps to start (if progress < 0.5)
// or end (if progress >= 0.5).
func InterpolateGridTracks(start, end []style.GridTrackSize, progress float64) []style.GridTrackSize {
	maxLen := len(start)
	if len(end) > maxLen {
		maxLen = len(end)
	}

	resultLen := len(start)
	if progress >= 0.5 {
		resultLen = len(end)
	}

	result := make([]style.GridTrackSize, 0, resultLen)

	for i := 0; i < maxLen; i++ {
		sExists := i < len(start)
		eExists := i < len(end)

		var track style.GridTrackSize
		addTrack := false

		if sExists && eExists {
			s := start[i]
			e := end[i]

			if s.Kind() == e.Kind() {
				switch s.Kind() {
				case style.KindCells:
					track = style.Cells(IntInterpolator(s.CellsValue(), e.CellsValue(), progress))
					addTrack = true
				case style.KindPercent:
					track = style.Percent(float32(FloatInterpolator(float64(s.PercentValue()), float64(e.PercentValue()), progress)))
					addTrack = true
				case style.KindFr:
					track = style.Fr(float32(FloatInterpolator(float64(s.FrValue()), float64(e.FrValue()), progress)))
					addTrack = true
				default:
					if progress < 0.5 {
						track = s
					} else {
						track = e
					}
					addTrack = true
				}
			} else {
				if progress < 0.5 {
					track = s
				} else {
					track = e
				}
				addTrack = true
			}
		} else if sExists {
			if progress < 0.5 {
				track = start[i]
				addTrack = true
			}
		} else if eExists {
			if progress >= 0.5 {
				track = end[i]
				addTrack = true
			}
		}

		if addTrack {
			result = append(result, track)
		}
	}

	return result
}
