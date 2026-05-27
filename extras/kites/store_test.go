package kites

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoreBasic(t *testing.T) {
	type State struct {
		Value int
		Name  string
	}

	initial := State{Value: 1, Name: "initial"}
	store := Create(initial)

	if store.Get() != initial {
		t.Errorf("expected initial state %v, got %v", initial, store.Get())
	}

	var callCount int32
	var lastNew, lastOld State

	unsubscribe := store.Subscribe(func(newVal, oldVal State) {
		atomic.AddInt32(&callCount, 1)
		lastNew = newVal
		lastOld = oldVal
	})

	// Update state
	store.Set(func(s State) State {
		s.Value = 2
		return s
	})

	if store.Get().Value != 2 {
		t.Errorf("expected updated value to be 2, got %d", store.Get().Value)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected subscriber to be called 1 time, got %d", callCount)
	}

	if lastNew.Value != 2 || lastOld.Value != 1 {
		t.Errorf("expected lastNew.Value=2 and lastOld.Value=1, got new=%d, old=%d", lastNew.Value, lastOld.Value)
	}

	// Unsubscribe
	unsubscribe()

	// Update again
	store.Set(func(s State) State {
		s.Value = 3
		return s
	})

	if store.Get().Value != 3 {
		t.Errorf("expected updated value to be 3, got %d", store.Get().Value)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected callCount to remain 1 after unsubscribe, got %d", callCount)
	}
}

func TestStoreConcurrency(t *testing.T) {
	store := Create(0)

	var wg sync.WaitGroup
	numGoroutines := 20
	iterations := 100

	// Readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = store.Get()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	// Writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.Set(func(val int) int {
					return val + 1
				})
				time.Sleep(time.Microsecond)
			}
		}()
	}

	// Subscribers subscribing and unsubscribing concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				unsub := store.Subscribe(func(newVal, oldVal int) {})
				time.Sleep(time.Millisecond)
				unsub()
			}
		}()
	}

	wg.Wait()

	expectedVal := numGoroutines * iterations
	if store.Get() != expectedVal {
		t.Errorf("expected final value to be %d, got %d", expectedVal, store.Get())
	}
}

func TestStoreNoDeadlock(t *testing.T) {
	store := Create(10)

	// A subscriber that modifies state or reads state inside notification
	store.Subscribe(func(newVal, oldVal int) {
		// Attempting to Get() or Set() inside callback.
		// If s.mu is held when notifying, this would deadlock.
		_ = store.Get()

		if newVal == 10 {
			store.Set(func(v int) int {
				return v + 5
			})
		}
	})

	done := make(chan bool)
	go func() {
		store.Set(func(v int) int {
			return v // trigger notification
		})
		done <- true
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock detected when calling Get/Set from inside subscriber notification")
	}
}
