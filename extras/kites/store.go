package kites

import (
	"sync"
)

// Store holds the state of type T and manages subscribers to changes.
// It is fully thread-safe.
type Store[T any] struct {
	mu          sync.RWMutex
	state       T
	subscribers map[uint64]func(newVal T, oldVal T)
	nextSubID   uint64
}

// Create initializes a new Store with the given initial state.
func Create[T any](initial T) *Store[T] {
	return &Store[T]{
		state:       initial,
		subscribers: make(map[uint64]func(T, T)),
	}
}

// Get returns the current state.
func (s *Store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set updates the state using the provided updater function.
// It calls the updater under a write lock, updates the state,
// and then notifies subscribers after unlocking to avoid potential deadlocks.
func (s *Store[T]) Set(updater func(T) T) {
	s.mu.Lock()
	oldState := s.state
	newState := updater(oldState)
	s.state = newState

	// Copy subscribers under the lock to prevent modification during iteration
	listeners := make([]func(T, T), 0, len(s.subscribers))
	for _, l := range s.subscribers {
		listeners = append(listeners, l)
	}
	s.mu.Unlock()

	// Notify subscribers outside the lock
	for _, l := range listeners {
		l(newState, oldState)
	}
}

// Subscribe registers a listener callback to be invoked when the state changes.
// It returns an unsubscribe function that removes the listener.
func (s *Store[T]) Subscribe(listener func(newVal T, oldVal T)) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextSubID
	s.nextSubID++
	s.subscribers[id] = listener

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.subscribers, id)
	}
}
