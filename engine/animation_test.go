package engine

import (
	"testing"
	"time"

	"github.com/masterkeysrd/kite/animation"
	"github.com/masterkeysrd/kite/backend/mock"
)

type testFakeClock struct {
	now time.Time
}

func (c *testFakeClock) Now() time.Time {
	return c.now
}

func (c *testFakeClock) After(d time.Duration) <-chan time.Time {
	return nil
}

func TestEngine_AnimationRegistryAndTicking(t *testing.T) {
	b := mock.New(80, 24)
	clk := &testFakeClock{now: time.Now()}
	e := New(b, Options{Clock: clk})
	defer e.Stop()

	// Initially, no animations and frame is not requested
	if len(e.activeAnimations) != 0 {
		t.Errorf("len(activeAnimations) = %d; want 0", len(e.activeAnimations))
	}

	var progressValues []int
	tween := animation.NewTween(0, 100, 100*time.Millisecond, animation.Linear, animation.IntInterpolator, func(v int) {
		progressValues = append(progressValues, v)
	})

	e.RegisterAnimation(tween)

	if len(e.activeAnimations) != 1 {
		t.Errorf("len(activeAnimations) = %d; want 1", len(e.activeAnimations))
	}
	if !e.frameRequested {
		t.Error("frameRequested is false after registering animation; want true")
	}

	// First Frame: dt is 0
	e.Frame()

	if len(progressValues) != 1 || progressValues[0] != 0 {
		t.Errorf("progressValues = %v; want [0]", progressValues)
	}
	if len(e.activeAnimations) != 1 {
		t.Error("animation was removed prematurely on first frame")
	}
	if !e.frameRequested {
		t.Error("frameRequested is false after Frame() with active animation; want true")
	}

	// Advance clock by 50ms
	clk.now = clk.now.Add(50 * time.Millisecond)
	e.Frame()

	if len(progressValues) != 2 || progressValues[1] != 50 {
		t.Errorf("progressValues = %v; want [0 50]", progressValues)
	}
	if len(e.activeAnimations) != 1 {
		t.Error("animation was removed prematurely on second frame")
	}
	if !e.frameRequested {
		t.Error("frameRequested is false after Frame() with active animation; want true")
	}

	// Advance clock by 50ms (total 100ms)
	clk.now = clk.now.Add(50 * time.Millisecond)
	e.Frame()

	if len(progressValues) != 3 || progressValues[2] != 100 {
		t.Errorf("progressValues = %v; want [0 50 100]", progressValues)
	}
	// Animation should be finished and removed
	if len(e.activeAnimations) != 0 {
		t.Errorf("len(activeAnimations) = %d; want 0 after animation finished", len(e.activeAnimations))
	}
	if e.frameRequested {
		t.Error("frameRequested is true after animation finished; want false")
	}
	if !e.lastFrameTime.IsZero() {
		t.Error("lastFrameTime is not zeroed out when no active animations are running")
	}
}
