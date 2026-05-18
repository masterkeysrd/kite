package engine_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/render"
)

// ---------------------------------------------------------------------------
// Fake helpers
// ---------------------------------------------------------------------------

// fakeClock is an injectable Clock for deterministic time tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	go func() {
		c.mu.Lock()
		target := c.now.Add(d)
		c.mu.Unlock()
		for {
			time.Sleep(time.Millisecond)
			c.mu.Lock()
			n := c.now
			c.mu.Unlock()
			if !n.Before(target) {
				ch <- n
				return
			}
		}
	}()
	return ch
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// newTestEngine creates an Engine with a mock backend and fake layout engine
// for unit tests. Width and height default to 80x24.
func newTestEngine(t *testing.T, opts ...engine.Options) (*engine.Engine, *mock.Backend) {
	t.Helper()
	b := mock.New(80, 24)
	opt := engine.Options{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Clock == nil {
		opt.Clock = engine.RealClock()
	}
	e := engine.New(b, opt)
	return e, b
}

// newTestEngineWithCaps creates an Engine with specific backend capabilities.
func newTestEngineWithCaps(t *testing.T, caps backend.Caps) (*engine.Engine, *mock.Backend) {
	t.Helper()
	b := mock.NewWithCaps(80, 24, caps)
	e := engine.New(b, engine.Options{Clock: engine.RealClock()})
	return e, b
}

// ---------------------------------------------------------------------------
// Phase-gate tests
// ---------------------------------------------------------------------------

// TestEngine_Frame_NoOp_WhenClean verifies that a frame with no dirty bits
// does not trigger BeginFrame / EndFrame on the backend.
func TestEngine_Frame_NoOp_WhenClean(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty bits from engine initialization (SetViewportSize).
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	e.Frame()

	if b.BeginFrameCalls != 0 {
		t.Errorf("BeginFrameCalls = %d, want 0 (no dirty bits)", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 0 {
		t.Errorf("EndFrameCalls = %d, want 0 (no dirty bits)", b.EndFrameCalls)
	}
}

// TestEngine_Frame_RunsAllPhases_WhenAllDirty verifies that a frame with all
// dirty bits set triggers all five phases.
func TestEngine_Frame_RunsAllPhases_WhenAllDirty(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyStyle | render.DirtyLayout | render.DirtyPaint)

	e.Frame()

	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 1 {
		t.Errorf("EndFrameCalls = %d, want 1", b.EndFrameCalls)
	}
	if root.Flags()&render.DirtyLayout != 0 {
		t.Error("layout engine did not clear DirtyLayout")
	}
}

// TestEngine_PhaseGate_StyleOnly verifies that only the style phase runs when
// only DirtyStyle is set.
func TestEngine_PhaseGate_StyleOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags (from SetViewportSize in newTestEngine).
	e.Frame()

	root := e.RenderView()
	root.MarkDirty(render.DirtyStyle)

	e.Frame()

	// Style phase ran but no paint (no paint dirty).
	if b.BeginFrameCalls != 0 {
		t.Errorf("BeginFrameCalls = %d, want 0 (no paint dirty)", b.BeginFrameCalls)
	}
	// Layout Engine checks removed since LayoutNG runs implicitly through root node
}

// TestEngine_PhaseGate_LayoutOnly verifies that only the layout phase runs
// when only DirtyLayout is set.
func TestEngine_PhaseGate_LayoutOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyLayout)

	e.Frame()

	// Layout ran but no paint.
	if b.BeginFrameCalls != 0 {
		t.Errorf("BeginFrameCalls = %d, want 0 (no paint dirty)", b.BeginFrameCalls)
	}
	if root.Flags()&render.DirtyLayout != 0 {
		t.Error("layout engine did not clear DirtyLayout")
	}
}

// TestEngine_PhaseGate_PaintOnly verifies that only the paint phase runs when
// only DirtyPaint is set.
func TestEngine_PhaseGate_PaintOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyPaint)

	e.Frame()

	// Paint phase ran.
	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1 (paint dirty)", b.BeginFrameCalls)
	}
}

// ---------------------------------------------------------------------------
// Frame loop / lifecycle tests
// ---------------------------------------------------------------------------

// TestEngine_Run_BlocksUntilStop verifies that Run blocks until Stop is called.
func TestEngine_Run_BlocksUntilStop(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)

	done := make(chan struct{})
	go func() {
		err := e.Run(context.Background())
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
		close(done)
	}()

	// Give it a moment to start.
	time.Sleep(20 * time.Millisecond)
	e.Stop()

	select {
	case <-done:
		// OK.
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after Stop")
	}
}

// ---------------------------------------------------------------------------
// Macro / Microtask tests
// ---------------------------------------------------------------------------

// TestEngine_Post_RunsInMicrotaskPhase verifies that microtasks run during
// the frame.
func TestEngine_Post_RunsInMicrotaskPhase(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)
	defer e.Stop()

	var run bool
	e.Post(func() { run = true })

	e.Frame()

	if !run {
		t.Error("microtask was not executed")
	}
}

// TestEngine_PostMacro_RespectsBudget verifies that macrotasks are capped
// by the count budget.
func TestEngine_PostMacro_RespectsBudget(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t, engine.Options{
		MacroTaskBudget: 2,
	})
	defer e.Stop()

	var count atomic.Int32
	for range 5 {
		e.PostMacro(func() { count.Add(1) })
	}

	e.Frame()

	if count.Load() != 2 {
		t.Errorf("executed %d macrotasks, want 2 (budget-capped)", count.Load())
	}

	e.Frame()

	if count.Load() != 4 {
		t.Errorf("after 2nd frame, executed %d total macrotasks, want 4", count.Load())
	}
}

// ---------------------------------------------------------------------------
// Worker pool tests
// ---------------------------------------------------------------------------

type countingJob struct {
	runs      atomic.Int32
	completes atomic.Int32
}

func (j *countingJob) Run(ctx context.Context) error {
	j.runs.Add(1)
	return nil
}

func (j *countingJob) OnComplete(result any, err error) {
	j.completes.Add(1)
}

func TestEngine_Submit_RunsOnWorker(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)
	defer e.Stop()

	job := &countingJob{}
	e.Submit(job)

	// Wait for job to run and result to be posted.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if job.runs.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if job.runs.Load() == 0 {
		t.Fatal("job did not run on worker")
	}

	// Result is posted as a microtask; must run a frame to execute OnComplete.
	e.Frame()

	if job.completes.Load() == 0 {
		t.Error("OnComplete was not called")
	}
}
