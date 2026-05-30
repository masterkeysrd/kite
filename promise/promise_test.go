package promise

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockScheduler struct {
	backgroundTasks []func(ctx context.Context)
	microtasks      []func()
}

func (m *mockScheduler) RunBackground(task func(ctx context.Context)) {
	m.backgroundTasks = append(m.backgroundTasks, task)
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
