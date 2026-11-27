// Regression tests for Engine, Kitex, Trail, and TestEnv — covers GitHub issue #17 (stopped-scheduler deadlocks, unmount global-scheduler clobbers, and flaky Flush() dropping passive UseKeyboard effects).
package regressions

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/trail"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/testenv"
)

func TestIssue17_SequentialTeardownDoesNotDeadlockPromises(t *testing.T) {
	b := mock.New(80, 10)
	inst, err := trail.New(trail.Text("hello"), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("trail.New failed: %v", err)
	}
	inst.Frame()
	inst.Unmount()
	inst.Engine().Stop()

	// Previously, Engine.Stop() left promise's globalScheduler pointing to the
	// stopped scheduler, causing the 17th promise.New call to block forever.
	for i := range 32 {
		p := promise.New(func(ctx context.Context) (int, error) {
			return i, nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		got, err := p.Await(ctx)
		cancel()
		if err != nil || got != i {
			t.Fatalf("promise %d failed after teardown: (%d, %v)", i, got, err)
		}
	}
}

func TestIssue17_OverlappingEngineTeardownDoesNotClobberNewerEngine(t *testing.T) {
	b1 := mock.New(80, 10)
	inst1, err := trail.New(trail.Text("first"), trail.WithBackend(b1))
	if err != nil {
		t.Fatalf("trail.New(1) failed: %v", err)
	}
	inst1.Frame()

	b2 := mock.New(80, 10)
	inst2, err := trail.New(trail.Text("second"), trail.WithBackend(b2))
	if err != nil {
		t.Fatalf("trail.New(2) failed: %v", err)
	}
	defer inst2.Engine().Stop()
	inst2.Frame()

	// Stop the older instance after the newer one initialized.
	inst1.Engine().Stop()

	for i := range 32 {
		p := promise.New(func(ctx context.Context) (int, error) {
			return i, nil
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		got, err := p.Await(ctx)
		cancel()
		if err != nil || got != i {
			t.Fatalf("promise %d failed after overlapping engine stop: (%d, %v)", i, got, err)
		}
	}
}

func TestIssue17_FlushDrainsPassiveUseKeyboardDespiteTinyMacroTaskDuration(t *testing.T) {
	// Use a custom engine with a 1ns MacroTaskDuration to deterministically
	// reproduce wall-clock expiry between Macrotask 1 (flushDirtyComponents)
	// and Macrotask 2 (flushPendingEffects attaching UseKeyboard).
	b := mock.New(80, 10)
	eng := engine.New(b, engine.Options{
		MacroTaskDuration: time.Nanosecond,
	})
	env := testenv.New(t, 80, 10, testenv.WithEngine(eng), testenv.WithBackend(b))

	var keyCount atomic.Int32
	var openChild func(bool)

	Child := kitex.FC("KeyChild", func(_ kitex.NoProps) kitex.Node {
		kitex.UseKeyboard(func(ke event.KeyEvent) {
			if ke.Text == "x" {
				keyCount.Add(1)
			}
		})
		return kitex.Text("child active")
	})

	App := kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
		show, setShow := kitex.UseState(false)
		openChild = setShow
		if show() {
			return Child()
		}
		return kitex.Text("initial")
	})

	container := element.NewBox(env.Document())
	env.Mount(container)
	kitex.Render(App(), container)
	env.Flush()

	// Trigger state update that mounts Child in Macrotask 1, which queues
	// UseKeyboard's UseEffectCleanup as Macrotask 2.
	openChild(true)
	env.Flush()

	env.KeyPress("x")
	env.Flush()

	if got := keyCount.Load(); got != 1 {
		t.Fatalf("expected UseKeyboard listener to receive key press after Flush(), got %d", got)
	}
}
