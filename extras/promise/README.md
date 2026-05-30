# Promise Package

The `extras/promise` package provides a Go-idiomatic, chainable async primitive for Kite applications. It replaces ad-hoc goroutines and ensures that callbacks are executed safely on the main UI thread.

## Usage

Promises are created using `promise.New`, which takes a context and an executor function returning `(T, error)`.

```go
p := promise.New(ctx, func(ctx context.Context) (string, error) {
    // This runs in a background worker pool.
    return fetchData()
})

p.Then(func(data string) {
    // This runs on the main UI thread.
    element.SetText(data)
}, func(err error) {
    // This also runs on the main UI thread.
    log.Error(err)
})
```

## Thread Safety

- **Background Execution**: The executor function runs in the engine's background worker pool, preventing UI freezes during expensive operations.
- **Main-Thread Callbacks**: `.Then()`, `.Catch()`, and `.Finally()` callbacks are automatically queued as microtasks on the main thread. This makes it safe to mutate the DOM or component state directly within these callbacks.
- **State Management**: The promise state machine is internally synchronized using a `sync.Mutex`, making it safe to register callbacks from any goroutine.

## Integration

The `promise` package is automatically integrated with the Kite engine. When you call `kitex.Render`, the scheduler is automatically wired, and any promises created thereafter will use the engine's bounded worker pool.
