package promise

import "context"

// Wrap converts a synchronous function that takes a context into an asynchronous
// function that returns a Promise. The context provided when calling the returned
// function supplies its values, deadline, and cancellation. Scheduler shutdown
// also cancels the execution context. The wrapped function must honor cancellation.
func Wrap[T any](fn func(context.Context) (T, error)) func(context.Context) *Promise[T] {
	return func(ctx context.Context) *Promise[T] {
		return newWithContext(ctx, fn)
	}
}

// WrapWithProps converts a synchronous function that takes a context and a properties
// payload into an asynchronous function that returns a Promise.
// It preserves the caller's context and scheduler cancellation in the same way as Wrap.
func WrapWithProps[P any, T any](fn func(context.Context, P) (T, error)) func(context.Context, P) *Promise[T] {
	return func(ctx context.Context, props P) *Promise[T] {
		return newWithContext(ctx, func(innerCtx context.Context) (T, error) {
			return fn(innerCtx, props)
		})
	}
}

func newWithContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Promise[T] {
	return New(func(schedulerCtx context.Context) (T, error) {
		executionCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(nil)
		stop := context.AfterFunc(schedulerCtx, func() {
			cancel(context.Cause(schedulerCtx))
		})
		defer stop()
		// AfterFunc runs asynchronously, even if the scheduler is already stopped.
		if schedulerCtx.Err() != nil {
			cancel(context.Cause(schedulerCtx))
		}
		if err := executionCtx.Err(); err != nil {
			var zero T
			return zero, err
		}
		return fn(executionCtx)
	})
}
