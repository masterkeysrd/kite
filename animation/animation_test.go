package animation

import (
	"image/color"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/style"
)

func TestEasingFunctions(t *testing.T) {
	funcs := map[string]EasingFunction{
		"Linear":         Linear,
		"EaseInQuad":     EaseInQuad,
		"EaseOutQuad":    EaseOutQuad,
		"EaseInOutCubic": EaseInOutCubic,
	}

	for name, fn := range funcs {
		t.Run(name, func(t *testing.T) {
			// Bounds
			if val := fn(0.0); val != 0.0 {
				t.Errorf("%s(0.0) = %f; want 0.0", name, val)
			}
			if val := fn(1.0); val != 1.0 {
				t.Errorf("%s(1.0) = %f; want 1.0", name, val)
			}

			// Intermediate progression
			half := fn(0.5)
			if half < 0.0 || half > 1.0 {
				t.Errorf("%s(0.5) = %f; out of bounds [0.0, 1.0]", name, half)
			}
		})
	}
}

func TestInterpolators(t *testing.T) {
	t.Run("Float", func(t *testing.T) {
		val := FloatInterpolator(10.0, 20.0, 0.5)
		if val != 15.0 {
			t.Errorf("FloatInterpolator(10, 20, 0.5) = %f; want 15.0", val)
		}
	})

	t.Run("Int", func(t *testing.T) {
		val := IntInterpolator(10, 20, 0.5)
		if val != 15 {
			t.Errorf("IntInterpolator(10, 20, 0.5) = %d; want 15", val)
		}
	})

	t.Run("Color", func(t *testing.T) {
		c1 := color.RGBA{R: 100, G: 200, B: 50, A: 255}
		c2 := color.RGBA{R: 200, G: 100, B: 150, A: 255}
		c := ColorInterpolator(c1, c2, 0.5)

		r, g, b, a := c.RGBA()
		// RGBA returns values premultiplied in [0, 65535]
		expectedR := uint32((100 + 200) / 2 * 257) // approx conversion to 16-bit
		expectedG := uint32((200 + 100) / 2 * 257)
		expectedB := uint32((50 + 150) / 2 * 257)
		expectedA := uint32(255 * 257)

		// Allow minor rounding tolerance (+-257, i.e., 1 in 8-bit scale)
		withinTol := func(v1, v2 uint32) bool {
			diff := int64(v1) - int64(v2)
			if diff < 0 {
				diff = -diff
			}
			return diff <= 512
		}

		if !withinTol(r, expectedR) || !withinTol(g, expectedG) || !withinTol(b, expectedB) || !withinTol(a, expectedA) {
			t.Errorf("ColorInterpolator(c1, c2, 0.5) = RGBA(%d,%d,%d,%d); want RGBA(%d,%d,%d,%d)", r, g, b, a, expectedR, expectedG, expectedB, expectedA)
		}
	})
}

func TestTween(t *testing.T) {
	var currentVal int
	tween := NewTween(0, 100, 100*time.Millisecond, Linear, IntInterpolator, func(v int) {
		currentVal = v
	})

	// Ticking: 0ms elapsed -> 0 progress
	finished := tween.Tick(0)
	if finished {
		t.Error("tween finished prematurely at 0ms")
	}
	if currentVal != 0 {
		t.Errorf("currentVal = %d; want 0", currentVal)
	}

	// Ticking: 50ms elapsed -> 0.5 progress
	finished = tween.Tick(50 * time.Millisecond)
	if finished {
		t.Error("tween finished prematurely at 50ms")
	}
	if currentVal != 50 {
		t.Errorf("currentVal = %d; want 50", currentVal)
	}

	// Ticking: 50ms more (100ms total) -> finished
	finished = tween.Tick(50 * time.Millisecond)
	if !finished {
		t.Error("tween did not finish at 100ms")
	}
	if currentVal != 100 {
		t.Errorf("currentVal = %d; want 100", currentVal)
	}

	// Ticking past duration
	finished = tween.Tick(10 * time.Millisecond)
	if !finished {
		t.Error("tween did not report finished after duration expired")
	}
}

func TestInterpolateGridTracks(t *testing.T) {
	t.Run("Same Kind Interpolation", func(t *testing.T) {
		start := []style.GridTrackSize{style.Fr(1)}
		end := []style.GridTrackSize{style.Fr(3)}
		result := InterpolateGridTracks(start, end, 0.5)

		if len(result) != 1 {
			t.Fatalf("expected length 1, got %d", len(result))
		}
		if result[0].Kind() != style.KindFr {
			t.Fatalf("expected KindFr, got %v", result[0].Kind())
		}
		if result[0].FrValue() != 2.0 {
			t.Errorf("expected 2.0fr, got %f", result[0].FrValue())
		}
	})

	t.Run("Different Kinds Snap", func(t *testing.T) {
		start := []style.GridTrackSize{style.Auto}
		end := []style.GridTrackSize{style.Fr(1)}

		// Progress < 0.5: snap to start
		res1 := InterpolateGridTracks(start, end, 0.4)
		if len(res1) != 1 || res1[0].Kind() != style.KindAuto {
			t.Errorf("expected [Auto] at progress 0.4, got %v", res1)
		}

		// Progress >= 0.5: snap to end
		res2 := InterpolateGridTracks(start, end, 0.5)
		if len(res2) != 1 || res2[0].Kind() != style.KindFr || res2[0].FrValue() != 1.0 {
			t.Errorf("expected [1fr] at progress 0.5, got %v", res2)
		}
	})

	t.Run("Different Lengths", func(t *testing.T) {
		start := []style.GridTrackSize{style.Fr(1)}
		end := []style.GridTrackSize{style.Fr(3), style.Cells(10)}

		// Progress < 0.5
		res1 := InterpolateGridTracks(start, end, 0.4)
		if len(res1) != 1 {
			t.Fatalf("expected length 1 at progress 0.4, got %d", len(res1))
		}
		if res1[0].FrValue() != 1.8 { // 1 + (3-1)*0.4 = 1.8
			t.Errorf("expected 1.8fr, got %f", res1[0].FrValue())
		}

		// Progress >= 0.5
		res2 := InterpolateGridTracks(start, end, 0.5)
		if len(res2) != 2 {
			t.Fatalf("expected length 2 at progress 0.5, got %d", len(res2))
		}
		if res2[0].FrValue() != 2.0 { // 1 + (3-1)*0.5 = 2.0
			t.Errorf("expected 2.0fr, got %f", res2[0].FrValue())
		}
		if res2[1].Kind() != style.KindCells || res2[1].CellsValue() != 10 {
			t.Errorf("expected 10 cells, got %v", res2[1])
		}
	})

	t.Run("Empty to Non-Empty", func(t *testing.T) {
		start := []style.GridTrackSize{}
		end := []style.GridTrackSize{style.Fr(1)}

		res1 := InterpolateGridTracks(start, end, 0.4)
		if len(res1) != 0 {
			t.Errorf("expected length 0, got %d", len(res1))
		}

		res2 := InterpolateGridTracks(start, end, 0.5)
		if len(res2) != 1 || res2[0].FrValue() != 1.0 {
			t.Errorf("expected [1fr], got %v", res2)
		}
	})
}

func TestTypedTweenConstructorsAndBuilders(t *testing.T) {
	t.Run("Int defaults to EaseOutQuad and fires OnComplete once", func(t *testing.T) {
		var latest int
		var completed int

		tw := Int(0, 100, 100*time.Millisecond, func(v int) {
			latest = v
		}).OnComplete(func() {
			completed++
		})

		if tw.Tick(50 * time.Millisecond) {
			t.Fatal("expected tween not to finish at 50ms")
		}
		// EaseOutQuad(0.5) = 0.75 -> 75
		if latest != 75 {
			t.Errorf("expected latest 75 with default EaseOutQuad, got %d", latest)
		}
		if completed != 0 {
			t.Errorf("expected OnComplete not yet called, got %d", completed)
		}

		if !tw.Tick(50 * time.Millisecond) {
			t.Fatal("expected tween to finish at 100ms")
		}
		if latest != 100 {
			t.Errorf("expected latest 100 at completion, got %d", latest)
		}
		if completed != 1 {
			t.Errorf("expected OnComplete called once, got %d", completed)
		}

		// Subsequent ticks should not invoke OnComplete again.
		if !tw.Tick(10 * time.Millisecond) {
			t.Fatal("expected tween to remain finished")
		}
		if completed != 1 {
			t.Errorf("expected OnComplete still called once, got %d", completed)
		}
	})

	t.Run("Float with Ease override", func(t *testing.T) {
		var latest float64
		tw := Float(0, 10, 100*time.Millisecond, func(v float64) {
			latest = v
		}).Ease(Linear)

		tw.Tick(50 * time.Millisecond)
		if latest != 5.0 {
			t.Errorf("expected 5.0 with Linear ease, got %f", latest)
		}
		if !tw.Tick(50 * time.Millisecond) {
			t.Fatal("expected Float tween to finish")
		}
		if latest != 10.0 {
			t.Errorf("expected 10.0 at end, got %f", latest)
		}
	})

	t.Run("Color with zero duration triggers OnComplete once", func(t *testing.T) {
		c1 := color.RGBA{R: 0, G: 0, B: 0, A: 255}
		c2 := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		var latest color.Color
		completed := 0

		tw := Color(c1, c2, 0, func(c color.Color) {
			latest = c
		}).OnComplete(func() {
			completed++
		})

		if !tw.Tick(0) {
			t.Fatal("expected zero-duration tween to finish immediately")
		}
		if latest != c2 {
			t.Errorf("expected end color %v, got %v", c2, latest)
		}
		if completed != 1 {
			t.Errorf("expected OnComplete called once, got %d", completed)
		}
		tw.Tick(time.Millisecond)
		if completed != 1 {
			t.Errorf("expected OnComplete still called once, got %d", completed)
		}
	})
}
