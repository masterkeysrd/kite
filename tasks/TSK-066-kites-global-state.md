# TSK-066: Implement `extras/kites` Global State Management

## Objective
Implement `extras/kites`, a lightweight, thread-safe global state management package for the Kite framework. The design follows an "External Store" model (similar to Zustand or Redux), providing reactive state outside of the `kitex` VDOM tree with optimized, selector-based re-rendering.

## Requirements

### 1. Core Store (`extras/kites/store.go`)
Create the foundational `kites.Store[T]` type. This must be entirely independent of `kitex` and safe for concurrent use.

- **Types:**
  - `type Store[T any] struct { ... }`
- **Functions:**
  - `func Create[T any](initial T) *Store[T]`
    - Initializes and returns a new store holding the `initial` state.
  - `func (s *Store[T]) Get() T`
    - Safely returns the current state using an `RLock()`.
  - `func (s *Store[T]) Set(updater func(T) T)`
    - Safely updates the state using an atomic `Lock()`.
    - `updater` receives the old state and must return the new state.
    - Notifies all registered subscribers with the `(new T, old T)` values *after* releasing the lock to prevent deadlocks.
  - `func (s *Store[T]) Subscribe(listener func(new T, old T)) (unsubscribe func())`
    - Registers a callback to be fired on state changes.
    - Returns a function that safely removes the listener from the subscriber list.

### 2. Kitex Integration (`extras/kites/hooks.go`)
Provide the bridge between the external store and the `kitex` reconciler.

- **Hook:**
  - `func Use[T any, U comparable](s *Store[T], selector func(T) U) U`
- **Behavior:**
  - Retrieves the initial slice using `selector(s.Get())` and registers it in local component state via `kitex.UseState`.
  - Subscribes to the store using `kitex.UseLayoutEffectCleanup` (or similar effect hook) to ensure the subscription is established on mount and cleaned up on unmount.
  - **Optimization (Bailout):** Inside the subscriber callback, compute `newSlice := selector(newState)`. *Only* invoke the `kitex` state setter if `newSlice != selector(oldState)`. This relies on `U comparable` and is critical for performance.

### 3. Documentation & Examples
- Add a package-level `doc.go` to `extras/kites` explaining the "Kite Store" pattern.
- Create an example application in `examples/kites_demo/main.go` that demonstrates:
  - Creating a global store with a struct containing multiple fields.
  - Updating the store from an event handler.
  - Two separate components reading different slices of the store, proving that updating slice A does not cause the component tracking slice B to re-render.
  - Updating the store asynchronously (e.g., from a goroutine mimicking a network fetch) to prove thread safety.

## Testing Requirements
- Unit tests for `Store[T]` in `store_test.go` verifying concurrent `Get`/`Set` operations (run with `-race`).
- Unit tests for subscriber registration and unsubscription.
- Integration test for `Use` verifying that a component re-renders when its selected slice changes, and *does not* re-render when an unselected part of the state changes.

## Related Documentation Updates
- Update `README.md` to mention `kites` alongside `kitex` as the official global state solution.