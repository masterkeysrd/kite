package promise

import "context"

// Wrap converts a synchronous function that takes a context into an asynchronous
// function that returns a Promise. The context provided when calling the returned
// function is passed directly to promise.New, allowing the scheduler to inject
// its own execution context if necessary.
func Wrap[T any](fn func(context.Context) (T, error)) func(context.Context) *Promise[T] {
	return func(ctx context.Context) *Promise[T] {
		return New(func(innerCtx context.Context) (T, error) {
			return fn(innerCtx)
		})
	}
}

// WrapWithProps converts a synchronous function that takes a context and a properties
// payload into an asynchronous function that returns a Promise.
func WrapWithProps[P any, T any](fn func(context.Context, P) (T, error)) func(context.Context, P) *Promise[T] {
	return func(ctx context.Context, props P) *Promise[T] {
		return New(func(innerCtx context.Context) (T, error) {
			return fn(innerCtx, props)
		})
	}
}
