// Package promise provides a lightweight, generic Promise<T> implementation
// for asynchronous operations in the Kite terminal UI engine.
//
// # Promise
//
// Promise[T any] represents the eventual completion or failure of an
// asynchronous operation. It is created via New(), which runs the
// executor on a background scheduler. The scheduler (set via
// SetScheduler) controls background execution, microtask queues, and
// macrotask queues.
//
// # Methods
//
//   - Then(fn, err) registers callbacks for fulfillment and rejection.
//   - Catch(err) registers only a rejection handler.
//   - Finally(fn) registers a handler for both settle paths.
//   - Await(ctx) blocks until the promise settles or the context is cancelled.
//
// # Factory Functions
//
//   - Resolved[T](val) returns an immediately fulfilled promise.
//   - RejectedPromise[T](err) returns an immediately rejected promise.
//
// # Scheduler
//
// The Scheduler interface controls where and how executor functions run.
// A global scheduler can be set via SetScheduler; if none is set,
// promises fall back to a simple goroutine and synchronous dispatch.
//
// Example:
//
//	p := promise.New(func(ctx context.Context) (string, error) {
//	     return "hello", nil
//	})
//	p.Then(func(s string) {
//	     fmt.Println(s)
//	})
package promise
