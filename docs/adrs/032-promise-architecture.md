# ADR 032: Promise and Scheduler Architecture

## Status
Accepted

## Context
Kite requires a mechanism for asynchronous operations (like network requests or heavy computations) that does not block the 60FPS UI thread, while ensuring that DOM mutations following those operations occur safely on the main thread.

Historically, this was handled via `engine.Job` (a low-level struct interface) and ad-hoc `go func()` calls in packages like `extras/wind`. This resulted in unbounded goroutine growth, complex boilerplate, and a lack of a unified concurrency model. Furthermore, the `engine.Engine` struct was heavily bloated by managing worker pools, mutexes, and task queues internally.

## Decision
We will replace `engine.Job` with a Go-idiomatic `promise` package backed by a decoupled `terminal.Scheduler`.

### 1. The `terminal.Scheduler` Interface
We will define a `Scheduler` capability within the `terminal` package to centralize concurrency limits and main-thread synchronization.
```go
type Scheduler interface {
    RunBackground(task func())     // Executes on a bounded worker pool
    QueueMicrotask(task func())    // Executes on the main UI thread (next drain)
    QueueMacrotask(task func())    // Executes on the main UI thread (budgeted)
}
```
The concrete implementation of this Scheduler will live in the `engine` package (e.g., `engine.NewScheduler(workers)`), effectively extracting the worker pool and queue management out of the core `Engine` struct. The Engine will simply coordinate by calling `DrainMicrotasks()` on the Scheduler during its frame loop.

### 2. Idiomatic Go Promises (`promise`)
We will introduce an `promise` package.
- **Global Context:** It will hold a global reference to the `terminal.Scheduler` (set at application boot) to avoid passing the scheduler into every promise creation.
- **Idiomatic Executor:** Instead of JavaScript-style `resolve`/`reject` callbacks, the `promise.New` executor will return a standard Go tuple `(T, error)`.
```go
promise.New(ctx, func(ctx context.Context) (User, error) {
    return fetchUser(ctx)
})
```
- **Thread-Safe Chaining:** The `.Then(onFulfilled, onRejected)` methods will automatically route their callbacks through `Scheduler.QueueMicrotask()`, guaranteeing that DOM mutations inside `.Then()` safely execute on the main UI thread.

## Consequences
### Positive
* **Bounded Concurrency:** The entire application (Promises, data fetchers) uses a single bounded worker pool. Unbounded `go func()` leaks are eliminated.
* **Go Idiomatic:** The `(T, error)` executor syntax is natural for Go developers, eliminating callback hell and boilerplate adapter structs (`JobFunc`).
* **Main-Thread Safety:** Developers can freely mutate the DOM inside `.Then()` without manually synchronizing threads.
* **Engine Purity:** The `Engine` struct is stripped of worker pool management, becoming a pure rendering coordinator.

### Negative
* Using a global variable for the `Scheduler` inside the `promise` package introduces hidden state, though this is an acceptable tradeoff for the massive improvement in developer ergonomics. Tests will need to ensure the global scheduler is initialized or mocked.
