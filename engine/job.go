package engine

import "context"

// Job is the unit of async work submitted to the engine's worker pool. The
// Run method executes on a worker goroutine; OnComplete is called on the main
// thread via the microtask queue after Run returns, ensuring it observes a
// consistent main-thread state.
//
// If Run panics, the worker goroutine recovers the panic, logs it via the
// engine's Logger, and calls OnComplete with a nil result and the recovered
// error. The engine remains operational after a single bad job.
type Job interface {
	// Run is called on a worker goroutine. The provided context is cancelled
	// when the engine is stopping. Run must not access engine state directly;
	// any state mutation must be deferred to OnComplete.
	Run(ctx context.Context) error

	// OnComplete is called on the main goroutine (via the microtask queue)
	// after Run returns. result is the value returned by Run (always nil for
	// Job implementations that return errors only); err is any error Run
	// returned or, if Run panicked, a recovered error.
	OnComplete(result any, err error)
}

// JobFunc is a thin adapter that lets callers submit a Job without defining a
// named struct.
type JobFunc struct {
	// RunFn is invoked by the worker goroutine.
	RunFn func(ctx context.Context) error
	// DoneFn is invoked on the main thread after Run completes.
	DoneFn func(result any, err error)
}

// Run implements Job by delegating to RunFn. If RunFn is nil Run is a no-op.
func (j *JobFunc) Run(ctx context.Context) error {
	if j.RunFn == nil {
		return nil
	}
	return j.RunFn(ctx)
}

// OnComplete implements Job by delegating to DoneFn. If DoneFn is nil
// OnComplete is a no-op.
func (j *JobFunc) OnComplete(result any, err error) {
	if j.DoneFn == nil {
		return
	}
	j.DoneFn(result, err)
}
