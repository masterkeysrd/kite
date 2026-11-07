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
	for range numGoroutines {
		wg.Go(func() {
			for range iterations {
				_ = store.Get()
				time.Sleep(time.Microsecond)
			}
		})
	}

	// Writers
	for range numGoroutines {
		wg.Go(func() {
			for range iterations {
				store.Set(func(val int) int {
					return val + 1
				})
				time.Sleep(time.Microsecond)
			}
		})
	}

	// Subscribers subscribing and unsubscribing concurrently
	for range 5 {
		wg.Go(func() {
			for range 10 {
				unsub := store.Subscribe(func(newVal, oldVal int) {})
				time.Sleep(time.Millisecond)
				unsub()
			}
		})
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

func TestStoreCopyOnWriteIsolation(t *testing.T) {
	type Message struct {
		Role    string
		Content string
		Tags    []string
	}
	type SessionState struct {
		Status   string
		Messages []Message
		Meta     map[string]string
	}

	store := Create(SessionState{
		Status: "ready",
		Messages: []Message{
			{Role: "assistant", Content: "Hello", Tags: []string{"greeting"}},
		},
		Meta: map[string]string{"model": "kite-1"},
	})

	snapBefore := store.Get()

	var subOld, subNew SessionState
	unsub := store.Subscribe(func(newVal, oldVal SessionState) {
		subNew = newVal
		subOld = oldVal
	})
	defer unsub()

	// Mutate slice element in-place, append to slice, and mutate map inside Update.
	store.Update(func(s *SessionState) {
		s.Messages[0].Content += " world"
		s.Messages[0].Tags[0] = "updated"
		s.Messages = append(s.Messages, Message{Role: "user", Content: "Hi"})
		s.Meta["model"] = "kite-2"
	})

	// Previous snapshot held by a reader (e.g., the UI drawing thread) must remain untouched.
	if snapBefore.Messages[0].Content != "Hello" {
		t.Fatalf("expected previous snapshot Content to remain 'Hello', got %q", snapBefore.Messages[0].Content)
	}
	if snapBefore.Messages[0].Tags[0] != "greeting" {
		t.Fatalf("expected previous snapshot Tag to remain 'greeting', got %q", snapBefore.Messages[0].Tags[0])
	}
	if snapBefore.Meta["model"] != "kite-1" {
		t.Fatalf("expected previous snapshot Meta['model'] to remain 'kite-1', got %q", snapBefore.Meta["model"])
	}

	// Subscriber must receive genuine oldVal and newVal snapshots.
	if subOld.Messages[0].Content != "Hello" {
		t.Fatalf("expected subscriber oldVal Content 'Hello', got %q", subOld.Messages[0].Content)
	}
	if subNew.Messages[0].Content != "Hello world" || len(subNew.Messages) != 2 {
		t.Fatalf("unexpected subscriber newVal: %+v", subNew)
	}

	// Direct SetState replacement.
	store.SetState(SessionState{Status: "done"})
	if store.Get().Status != "done" || len(store.Get().Messages) != 0 {
		t.Fatalf("expected SetState to replace state, got %+v", store.Get())
	}
}

