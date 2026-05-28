# Task: Implement Idiomatic Promises

## Objective
Introduce the `extras/promise` package to replace ad-hoc `go func()` calls and the deprecated `engine.Job` interface with a Go-idiomatic, chainable async primitive.

## Requirements
1. **Create `extras/promise` Package:**
   - Create `extras/promise/promise.go`.
   - Define the global state:
     ```go
     var globalScheduler terminal.Scheduler
     func SetScheduler(s terminal.Scheduler) { globalScheduler = s }
     ```

2. **Implement `Promise[T]` API:**
   - Define the constructor: `func New[T any](ctx context.Context, executor func(context.Context) (T, error)) *Promise[T]`
   - Inside `New`, call `globalScheduler.RunBackground(...)` to execute the user's function.
   - Implement `Then(onFulfilled func(T), onRejected func(error)) *Promise[T]`.
   - Implement `Catch(onRejected func(error)) *Promise[T]`.
   - Implement `Finally(onFinally func()) *Promise[T]`.
   - Implement `Await(ctx context.Context) (T, error)` to block the caller until resolution.

3. **Main-Thread Safety:**
   - Ensure the internal state machine locks via `sync.Mutex`.
   - When a promise settles, push the registered `.Then` / `.Catch` callbacks to `globalScheduler.QueueMicrotask(func() { ... })`. This guarantees DOM mutations happen safely on the main thread.

4. **Refactor `extras/wind`:**
   - Update `extras/wind/query.go` to use `promise.New` (or directly use the global scheduler) instead of spawning an unbounded `go func()` for every fetch.

## Tests to Verify
- Write `promise_test.go` using a mock `terminal.Scheduler` that executes tasks synchronously to verify resolution, rejection, and chaining logic without race conditions.
- Run `go test ./extras/wind/...` to ensure data fetching still works over the new bounded pool.

## Documentation Updates
- Create a `README.md` or `doc.go` in `extras/promise` explaining the `(T, error)` executor pattern and the main-thread safety guarantees of `.Then()`.