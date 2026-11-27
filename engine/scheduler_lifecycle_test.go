package engine_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/promise"
)

func TestEngine_Scheduler_RunBackgroundAfterStopDoesNotBlock(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	sched := e.Scheduler()
	e.Stop()

	var executed atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Submit far more jobs than numWorkers*4 to verify RunBackground never
		// blocks on the orphaned jobQueue after Stop().
		for range 100 {
			sched.RunBackground(func(ctx context.Context) {
				executed.Add(1)
			})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunBackground deadlocked after Engine.Stop()")
	}

	if got := executed.Load(); got != 0 {
		t.Fatalf("expected 0 tasks to execute on stopped scheduler, got %d", got)
	}
}

func TestEngine_PromiseAgainstStoppedSchedulerRejectsCleanly(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	sched := e.Scheduler()
	e.Stop()

	promise.SetScheduler(sched)
	defer promise.SetScheduler(nil)

	for i := range 32 {
		p := promise.New(func(ctx context.Context) (int, error) {
			return i, nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := p.Await(ctx)
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("promise %d: expected context.Canceled, got %v", i, err)
		}

		var caught error
		finallyCalled := false
		p.Catch(func(e error) { caught = e })
		p.Finally(func() { finallyCalled = true })
		if !errors.Is(caught, context.Canceled) {
			t.Fatalf("promise %d: expected Catch to fire with context.Canceled, got %v", i, caught)
		}
		if !finallyCalled {
			t.Fatalf("promise %d: expected Finally to fire synchronously on stopped scheduler", i)
		}
	}
}

func TestEngine_Stop_ClearsPromiseSchedulerWithoutClobberingNewer(t *testing.T) {
	b1 := mock.New(80, 24)
	e1 := engine.New(b1, engine.Options{NumWorkers: 2})

	b2 := mock.New(80, 24)
	e2 := engine.New(b2, engine.Options{NumWorkers: 2})
	defer e2.Stop()

	// Stopping e1 after e2 started must not clear e2's scheduler.
	e1.Stop()

	p := promise.New(func(ctx context.Context) (string, error) {
		return "ok", nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	got, err := p.Await(ctx)
	cancel()
	if err != nil || got != "ok" {
		t.Fatalf("expected promise on e2 to succeed, got (%q, %v)", got, err)
	}

	// Stopping e2 must clear its own global registration so subsequent promises
	// fall back cleanly instead of targeting a stopped scheduler.
	e2.Stop()

	for i := range 32 {
		p2 := promise.New(func(ctx context.Context) (int, error) {
			return i, nil
		})
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		val, err2 := p2.Await(ctx2)
		cancel2()
		if err2 != nil || val != i {
			t.Fatalf("fallback promise %d failed: (%d, %v)", i, val, err2)
		}
	}
}

func TestEngine_MacroTaskDuration_Unlimited(t *testing.T) {
	clk := newFakeClock(time.Unix(1000, 0))
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{
		Clock:             clk,
		MacroTaskDuration: engine.UnlimitedMacroTaskDuration,
	})
	defer e.Stop()

	secondRan := false
	e.PostMacro(func() {
		// Simulate heavy OS preemption advancing wall clock well past DefaultMacroTaskDuration.
		clk.mu.Lock()
		clk.now = clk.now.Add(5 * time.Second)
		clk.mu.Unlock()
		e.PostMacro(func() {
			secondRan = true
		})
	})

	e.Frame()

	if !secondRan {
		t.Fatal("expected follow-up macrotask to drain when MacroTaskDuration is UnlimitedMacroTaskDuration")
	}
	if e.HasPendingTasks() {
		t.Fatal("expected no pending tasks after drain")
	}
}

func TestEngine_Scheduler_SpawnNonStarvationAndPanicRecovery(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	defer e.Stop()

	sched := e.Scheduler()

	// Saturate both RunBackground workers.
	workerGate := make(chan struct{})
	var workersBusy atomic.Int32
	var workersDone atomic.Int32
	for range 2 {
		sched.RunBackground(func(ctx context.Context) {
			workersBusy.Add(1)
			select {
			case <-ctx.Done():
			case <-workerGate:
			}
			workersDone.Add(1)
		})
	}

	deadline := time.Now().Add(time.Second)
	for workersBusy.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if workersBusy.Load() != 2 {
		t.Fatalf("expected 2 workers busy, got %d", workersBusy.Load())
	}

	// Spawn 6 tasks concurrently while all workers are blocked, including a panicking one.
	const spawnCount = 6
	var spawnedStarted atomic.Int32
	var spawnedCompleted atomic.Int32
	spawnedDone := make(chan struct{})
	for i := range spawnCount {
		idx := i
		sched.Spawn(func(ctx context.Context) {
			spawnedStarted.Add(1)
			if idx == 0 {
				panic("boom in spawned task")
			}
			if spawnedCompleted.Add(1) == spawnCount-1 {
				close(spawnedDone)
			}
		})
	}

	select {
	case <-spawnedDone:
	case <-time.After(time.Second):
		t.Fatalf("expected spawned tasks to run without starving on worker pool (started=%d, completed=%d)", spawnedStarted.Load(), spawnedCompleted.Load())
	}

	close(workerGate)
	deadline = time.Now().Add(time.Second)
	for workersDone.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if workersDone.Load() != 2 {
		t.Fatalf("expected workers to complete after gate release, got %d", workersDone.Load())
	}
}

func TestEngine_Scheduler_SpawnCancelAndDrainOnStop(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	sched := e.Scheduler()

	type spawner interface {
		TrySpawn(func(context.Context)) bool
	}
	ts, ok := sched.(spawner)
	if !ok {
		t.Fatal("expected defaultScheduler to implement TrySpawn")
	}

	if ts.TrySpawn(nil) {
		t.Fatal("expected TrySpawn(nil) to return false")
	}

	const n = 8
	var started atomic.Int32
	var exited atomic.Int32
	for range n {
		if !ts.TrySpawn(func(ctx context.Context) {
			started.Add(1)
			<-ctx.Done()
			time.Sleep(5 * time.Millisecond)
			exited.Add(1)
		}) {
			t.Fatal("expected TrySpawn to succeed before Stop")
		}
	}

	deadline := time.Now().Add(time.Second)
	for started.Load() < n && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if started.Load() != n {
		t.Fatalf("expected %d spawned tasks to start, got %d", n, started.Load())
	}

	// Race concurrent TrySpawn calls against e.Stop().
	var raceWG sync.WaitGroup
	var raceExited atomic.Int32
	var raceStarted atomic.Int32
	for range 16 {
		raceWG.Add(1)
		go func() {
			defer raceWG.Done()
			ts.TrySpawn(func(ctx context.Context) {
				raceStarted.Add(1)
				<-ctx.Done()
				raceExited.Add(1)
			})
		}()
	}

	e.Stop()
	raceWG.Wait()

	if exited.Load() != n {
		t.Fatalf("expected e.Stop() to cancel and drain all %d spawned tasks, got %d exited", n, exited.Load())
	}
	if raceExited.Load() != raceStarted.Load() {
		t.Fatalf("expected all concurrently spawned tasks to drain on Stop (started=%d, exited=%d)", raceStarted.Load(), raceExited.Load())
	}

	var postStopRan atomic.Bool
	if ts.TrySpawn(func(ctx context.Context) {
		postStopRan.Store(true)
	}) {
		t.Fatal("expected TrySpawn after Stop to return false")
	}
	sched.Spawn(func(ctx context.Context) {
		postStopRan.Store(true)
	})
	time.Sleep(10 * time.Millisecond)
	if postStopRan.Load() {
		t.Fatal("expected Spawn after Stop not to run task")
	}
}
