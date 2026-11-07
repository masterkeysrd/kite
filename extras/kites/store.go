package kites

import (
	"reflect"
	"sync"
	"time"
)

// Store holds the state of type T and manages subscribers to changes.
// It is fully thread-safe and can be used both inside and outside the kitex VDOM tree.
type Store[T any] struct {
	mu          sync.RWMutex
	state       T
	needsClone  bool
	subscribers map[uint64]func(newVal T, oldVal T)
	nextSubID   uint64
}

// Create initializes a new Store with the given initial state.
func Create[T any](initial T) *Store[T] {
	needsClone := typeNeedsClone(reflect.TypeFor[T]())
	if needsClone {
		initial = cloneValue(initial)
	}
	return &Store[T]{
		state:       initial,
		needsClone:  needsClone,
		subscribers: make(map[uint64]func(T, T)),
	}
}

// Get returns the current state snapshot.
func (s *Store[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Set updates the state using the provided updater function.
// It passes an isolated copy-on-write draft of the current state to updater under
// a write lock so that in-place mutations to slices, maps, or data pointers inside
// updater never race with concurrent readers (such as the UI drawing thread) or
// corrupt previous snapshots. Subscribers are notified after unlocking.
func (s *Store[T]) Set(updater func(T) T) {
	s.mu.Lock()
	oldState := s.state
	draft := oldState
	if s.needsClone {
		draft = cloneValue(oldState)
	}
	newState := updater(draft)
	s.state = newState

	if len(s.subscribers) == 0 {
		s.mu.Unlock()
		return
	}

	// Copy subscribers under the lock to prevent modification during iteration.
	listeners := make([]func(T, T), 0, len(s.subscribers))
	for _, l := range s.subscribers {
		listeners = append(listeners, l)
	}
	s.mu.Unlock()

	// Notify subscribers outside the lock.
	for _, l := range listeners {
		l(newState, oldState)
	}
}

// Update mutates the state in place using a pointer to an isolated copy-on-write
// draft of the state and commits the result.
func (s *Store[T]) Update(mutator func(*T)) {
	s.Set(func(draft T) T {
		mutator(&draft)
		return draft
	})
}

// SetState replaces the store's state with next and notifies subscribers.
func (s *Store[T]) SetState(next T) {
	s.Set(func(_ T) T {
		return next
	})
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

var (
	needsCloneCache sync.Map // map[reflect.Type]bool
	timeType        = reflect.TypeFor[time.Time]()
)

func typeNeedsClone(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if cached, ok := needsCloneCache.Load(t); ok {
		return cached.(bool)
	}
	res := computeNeedsClone(t, make(map[reflect.Type]bool))
	needsCloneCache.Store(t, res)
	return res
}

func computeNeedsClone(t reflect.Type, visited map[reflect.Type]bool) bool {
	if t == nil || t == timeType || isSyncOrAtomicType(t) {
		return false
	}
	if visited[t] {
		return false
	}
	visited[t] = true

	switch t.Kind() {
	case reflect.Slice, reflect.Map, reflect.Interface:
		return true
	case reflect.Pointer:
		elem := t.Elem()
		if isOpaqueTarget(elem) {
			return false
		}
		return true
	case reflect.Array:
		return computeNeedsClone(t.Elem(), visited)
	case reflect.Struct:
		for f := range t.Fields() {
			if !f.IsExported() {
				continue
			}
			if computeNeedsClone(f.Type, visited) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func isSyncOrAtomicType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	pkg := t.PkgPath()
	return pkg == "sync" || pkg == "sync/atomic"
}

// isOpaqueTarget reports whether a pointer target type represents an opaque handle
// or synchronization primitive (such as *sync.Mutex, *http.Client, *os.File) that
// must be shared by reference rather than duplicated.
func isOpaqueTarget(t reflect.Type) bool {
	if t == nil || t == timeType || isSyncOrAtomicType(t) {
		return true
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for f := range t.Fields() {
		if isSyncOrAtomicType(f.Type) {
			return true
		}
		if !f.IsExported() && f.Type != timeType {
			return true
		}
		if f.Type.Kind() == reflect.Struct && f.Type != timeType && isOpaqueTarget(f.Type) {
			return true
		}
	}
	return false
}

func cloneValue[T any](v T) T {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return v
	}
	cloned := cloneReflectValue(rv, nil)
	return cloned.Interface().(T)
}

func cloneReflectValue(v reflect.Value, seen map[uintptr]reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	t := v.Type()
	if !typeNeedsClone(t) {
		return v
	}

	switch t.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		n := v.Len()
		cp := reflect.MakeSlice(t, n, n)
		if !typeNeedsClone(t.Elem()) {
			reflect.Copy(cp, v)
			return cp
		}
		for i := range n {
			cp.Index(i).Set(cloneReflectValue(v.Index(i), seen))
		}
		return cp

	case reflect.Map:
		if v.IsNil() {
			return v
		}
		cp := reflect.MakeMapWithSize(t, v.Len())
		cloneKey := typeNeedsClone(t.Key())
		cloneElem := typeNeedsClone(t.Elem())
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			val := iter.Value()
			if cloneKey {
				k = cloneReflectValue(k, seen)
			}
			if cloneElem {
				val = cloneReflectValue(val, seen)
			}
			cp.SetMapIndex(k, val)
		}
		return cp

	case reflect.Struct:
		cp := reflect.New(t).Elem()
		cp.Set(v)
		for i := range t.NumField() {
			f := t.Field(i)
			if f.IsExported() && typeNeedsClone(f.Type) {
				cp.Field(i).Set(cloneReflectValue(v.Field(i), seen))
			}
		}
		return cp

	case reflect.Pointer:
		if v.IsNil() || isOpaqueTarget(t.Elem()) {
			return v
		}
		ptr := v.Pointer()
		if seen == nil {
			seen = make(map[uintptr]reflect.Value)
		} else if existing, ok := seen[ptr]; ok {
			return existing
		}
		cp := reflect.New(t.Elem())
		seen[ptr] = cp
		cp.Elem().Set(cloneReflectValue(v.Elem(), seen))
		return cp

	case reflect.Array:
		cp := reflect.New(t).Elem()
		for i := range t.Len() {
			cp.Index(i).Set(cloneReflectValue(v.Index(i), seen))
		}
		return cp

	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		inner := cloneReflectValue(v.Elem(), seen)
		cp := reflect.New(t).Elem()
		cp.Set(inner)
		return cp

	default:
		return v
	}
}
