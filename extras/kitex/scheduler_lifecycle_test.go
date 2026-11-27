package kitex

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/promise"
)

func TestRender_UnmountDoesNotClobberNewerEngineScheduler(t *testing.T) {
	prevSched := getInternalScheduler()
	defer setInternalScheduler(prevSched)

	b1 := mock.New(80, 24)
	e1 := engine.New(b1, engine.Options{NumWorkers: 2})
	c1 := element.NewBox(e1.Document())
	e1.Mount(c1)
	Render(Text("first"), c1)

	if getInternalScheduler() != e1.Scheduler() {
		t.Fatal("expected internalScheduler to be e1's scheduler after mounting on e1")
	}

	b2 := mock.New(80, 24)
	e2 := engine.New(b2, engine.Options{NumWorkers: 2})
	defer e2.Stop()
	c2 := element.NewBox(e2.Document())
	e2.Mount(c2)
	Render(Text("second"), c2)

	if getInternalScheduler() != e2.Scheduler() {
		t.Fatal("expected internalScheduler to be e2's scheduler after mounting on e2")
	}

	// Unmounting and stopping e1 must not overwrite or clear e2's scheduler.
	Unmount(c1)
	if getInternalScheduler() != e2.Scheduler() {
		t.Fatal("Unmount(c1) clobbered e2's internalScheduler")
	}

	e1.Stop()
	if getInternalScheduler() != e2.Scheduler() {
		t.Fatal("e1.Stop() clobbered e2's internalScheduler")
	}

	// Promises must still run cleanly (not pointed at e1's stopped scheduler).
	for i := range 24 {
		p := promise.New(func(ctx context.Context) (int, error) {
			return i, nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		got, err := p.Await(ctx)
		cancel()
		if err != nil || got != i {
			t.Fatalf("promise %d failed after e1 stopped: (%d, %v)", i, got, err)
		}
	}

	// Stopping e2 should clear kitex's internalScheduler since e2 owns it.
	e2.Stop()
	if getInternalScheduler() != nil {
		t.Fatal("expected internalScheduler to be nil after e2.Stop()")
	}
}

func TestUseAsyncEffect_SchedulerStopCancelsAndDrains(t *testing.T) {
	prevSched := getInternalScheduler()
	defer setInternalScheduler(prevSched)

	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	c := element.NewBox(e.Document())
	e.Mount(c)

	started := make(chan struct{})
	cancelled := make(chan struct{})
	var exited atomic.Bool

	comp := FC("AsyncStopComp", func(_ NoProps) Node {
		UseAsyncEffect(func(ctx context.Context) {
			close(started)
			<-ctx.Done()
			time.Sleep(5 * time.Millisecond)
			exited.Store(true)
			close(cancelled)
		})
		return Box(BoxProps{})
	})

	Render(comp(), c)
	e.Frame()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("expected UseAsyncEffect to start via scheduler.Spawn")
	}

	// Stop engine BEFORE unmounting component.
	e.Stop()

	select {
	case <-cancelled:
	default:
		t.Fatal("expected e.Stop() to cancel UseAsyncEffect context and drain before returning")
	}
	if !exited.Load() {
		t.Fatal("expected UseAsyncEffect goroutine to finish before e.Stop() returned")
	}
}
