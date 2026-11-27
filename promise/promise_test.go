package promise

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockScheduler struct {
	backgroundTasks []func(ctx context.Context)
	spawnedTasks    []func(ctx context.Context)
	microtasks      []func()
}

func (m *mockScheduler) RunBackground(task func(ctx context.Context)) {
	m.backgroundTasks = append(m.backgroundTasks, task)
}

func (m *mockScheduler) Spawn(task func(ctx context.Context)) {
	m.spawnedTasks = append(m.spawnedTasks, task)
}

func (m *mockScheduler) QueueMicrotask(task func()) {
	m.microtasks = append(m.microtasks, task)
}

func (m *mockScheduler) QueueMacrotask(task func()) {
	// Not used by promise
}

func (m *mockScheduler) flushBackground() {
	for len(m.backgroundTasks) > 0 {
		tasks := m.backgroundTasks
		m.backgroundTasks = nil
		for _, t := range tasks {
			t(context.Background())
		}
	}
}

func (m *mockScheduler) flushMicrotasks() {
	for len(m.microtasks) > 0 {
		tasks := m.microtasks
		m.microtasks = nil
		for _, t := range tasks {
			t()
		}
	}
}

var _ Scheduler = (*mockScheduler)(nil)

func TestPromise_Fulfill(t *testing.T) {
	sched := &mockScheduler{}
	SetScheduler(sched)

	p := New(func(ctx context.Context) (string, error) {
		return "hello", nil
	})

	fulfilledCalled := false
	var result string
	p.Then(func(v string) {
		fulfilledCalled = true
		result = v
	}, nil)

	if fulfilledCalled {
		t.Fatal("fulfilled called prematurely")
	}

	sched.flushBackground()
	if fulfilledCalled {
		t.Fatal("fulfilled called before microtask flush")
	}

	sched.flushMicrotasks()

	if !fulfilledCalled {
		t.Fatal("fulfilled not called")
	}
	if result != "hello" {
		t.Errorf("expected hello, got %s", result)
	}
}

func TestPromise_Reject(t *testing.T) {
	sched := &mockScheduler{}
	SetScheduler(sched)

	errTest := errors.New("test error")
	p := New(func(ctx context.Context) (string, error) {
		return "", errTest
	})

	rejectedCalled := false
	var capturedErr error
	p.Catch(func(err error) {
		rejectedCalled = true
		capturedErr = err
	})

	sched.flushBackground()
	sched.flushMicrotasks()

	if !rejectedCalled {
		t.Fatal("rejected not called")
	}
	if capturedErr != errTest {
		t.Errorf("expected %v, got %v", errTest, capturedErr)
	}
}

func TestPromise_Chaining_Settled(t *testing.T) {
	sched := &mockScheduler{}
	SetScheduler(sched)

	p := New(func(ctx context.Context) (string, error) {
		return "instant", nil
	})

	sched.flushBackground()
	sched.flushMicrotasks()

	// Promise is already fulfilled here.
	called := false
	p.Then(func(v string) {
		called = true
	}, nil)

	if called {
		t.Fatal("callback called synchronously on settled promise")
	}

	sched.flushMicrotasks()
	if !called {
		t.Fatal("callback not called for settled promise")
	}
}

func TestPromise_Finally(t *testing.T) {
	sched := &mockScheduler{}
	SetScheduler(sched)

	p := New(func(ctx context.Context) (int, error) {
		return 42, nil
	})

	finallyCalled := false
	p.Finally(func() {
		finallyCalled = true
	})

	sched.flushBackground()
	sched.flushMicrotasks()

	if !finallyCalled {
		t.Fatal("finally not called")
	}
}

func TestPromise_Await(t *testing.T) {
	// For Await, we need real concurrency or at least a way to trigger resolution.
	// But in this unit test, we'll just check if it works with the mock.
	sched := &mockScheduler{}
	SetScheduler(sched)

	p := New(func(ctx context.Context) (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "awaited", nil
	})

	// Run background in another goroutine so we can Await here
	go func() {
		time.Sleep(5 * time.Millisecond)
		sched.flushBackground()
	}()

	res, err := p.Await(context.Background())
	if err != nil {
		t.Fatalf("Await failed: %v", err)
	}
	if res != "awaited" {
		t.Errorf("expected awaited, got %s", res)
	}
}

func TestPromise_ClearSchedulerIf(t *testing.T) {
	s1 := &mockScheduler{}
	s2 := &mockScheduler{}
	SetScheduler(s1)
	defer SetScheduler(nil)

	if ClearSchedulerIf(nil) {
		t.Fatal("ClearSchedulerIf(nil) should return false")
	}
	if ClearSchedulerIf(s2) {
		t.Fatal("ClearSchedulerIf(s2) should return false when s1 is active")
	}
	if getScheduler() != s1 {
		t.Fatal("expected s1 to remain active")
	}
	if !ClearSchedulerIf(s1) {
		t.Fatal("ClearSchedulerIf(s1) should return true when s1 is active")
	}
	if getScheduler() != nil {
		t.Fatal("expected globalScheduler to be nil after ClearSchedulerIf(s1)")
	}
	if ClearSchedulerIf(s1) {
		t.Fatal("second ClearSchedulerIf(s1) should return false")
	}
}

type tryStoppableMock struct {
	mockScheduler
	stopped    bool
	trySpawned bool
}

func (m *tryStoppableMock) IsStopped() bool { return m.stopped }
func (m *tryStoppableMock) TrySpawn(task func(ctx context.Context)) bool {
	if m.stopped {
		return false
	}
	m.trySpawned = true
	go task(context.Background())
	return true
}

func TestPromise_Spawn(t *testing.T) {
	prev := getScheduler()
	defer SetScheduler(prev)

	if Spawn(nil) {
		t.Fatal("expected Spawn(nil) to return false")
	}

	SetScheduler(nil)
	fallbackDone := make(chan struct{})
	if !Spawn(func(ctx context.Context) {
		if ctx == nil {
			t.Error("expected non-nil context in fallback Spawn")
		}
		close(fallbackDone)
	}) {
		t.Fatal("expected Spawn with nil scheduler to return true")
	}
	select {
	case <-fallbackDone:
	case <-time.After(time.Second):
		t.Fatal("expected fallback Spawn task to execute")
	}

	ms := &mockScheduler{}
	SetScheduler(ms)
	if !Spawn(func(ctx context.Context) {}) {
		t.Fatal("expected Spawn with mockScheduler to return true")
	}
	if len(ms.spawnedTasks) != 1 {
		t.Fatalf("expected 1 spawned task recorded on mockScheduler, got %d", len(ms.spawnedTasks))
	}

	ts := &tryStoppableMock{}
	SetScheduler(ts)
	ran := make(chan struct{})
	if !Spawn(func(ctx context.Context) { close(ran) }) {
		t.Fatal("expected Spawn with active tryStoppableMock to return true")
	}
	if !ts.trySpawned {
		t.Fatal("expected TrySpawn to be used when implemented")
	}
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("expected spawned task to run")
	}

	ts.stopped = true
	if Spawn(func(ctx context.Context) {}) {
		t.Fatal("expected Spawn on stopped scheduler to return false")
	}
}
