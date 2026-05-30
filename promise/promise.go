package promise

import (
	"context"
	"sync"
)

var (
	globalScheduler Scheduler
	schedMu         sync.RWMutex
)

type Scheduler interface {
	// RunBackground executes a task on a background worker pool.
	// The provided context is managed by the scheduler.
	RunBackground(task func(ctx context.Context))
	// QueueMicrotask schedules a task to run as a microtask on the main thread.
	QueueMicrotask(task func())
	// QueueMacrotask schedules a task to run as a macrotask on the main thread.
	QueueMacrotask(task func())
}

// SetScheduler sets the global scheduler used by all promises.
func SetScheduler(s Scheduler) {
	schedMu.Lock()
	defer schedMu.Unlock()
	globalScheduler = s
}

func getScheduler() Scheduler {
	schedMu.RLock()
	defer schedMu.RUnlock()
	return globalScheduler
}

type State int

const (
	Pending State = iota
	Fulfilled
	Rejected
)

// Promise represents the eventual completion (or failure) of an asynchronous operation
// and its resulting value.
type Promise[T any] struct {
	mu sync.Mutex

	state State
	value T
	err   error

	onFulfilled []func(T)
	onRejected  []func(error)
	onFinally   []func()

	done chan struct{}
}

// New creates a new Promise and executes the executor in the background.
// The context provided to the executor is managed by the scheduler.
func New[T any](executor func(context.Context) (T, error)) *Promise[T] {
	p := &Promise[T]{
		state: Pending,
		done:  make(chan struct{}),
	}

	sched := getScheduler()
	if sched == nil {
		// Fallback for tests or uninitialized environments
		go p.run(context.Background(), executor)
		return p
	}

	sched.RunBackground(func(ctx context.Context) {
		p.run(ctx, executor)
	})

	return p
}

func (p *Promise[T]) run(ctx context.Context, executor func(context.Context) (T, error)) {
	val, err := executor(ctx)

	p.mu.Lock()
	if err != nil {
		p.state = Rejected
		p.err = err
	} else {
		p.state = Fulfilled
		p.value = val
	}
	p.mu.Unlock()

	close(p.done)

	sched := getScheduler()
	if sched == nil {
		p.dispatchSync()
		return
	}

	sched.QueueMicrotask(func() {
		p.dispatchSync()
	})
}

func (p *Promise[T]) dispatchSync() {
	p.mu.Lock()
	state := p.state
	val := p.value
	err := p.err
	onFulfilled := p.onFulfilled
	onRejected := p.onRejected
	onFinally := p.onFinally

	// Clear callbacks after dispatch to prevent leaks
	p.onFulfilled = nil
	p.onRejected = nil
	p.onFinally = nil
	p.mu.Unlock()

	switch state {
	case Fulfilled:
		for _, fn := range onFulfilled {
			fn(val)
		}
	case Rejected:
		for _, fn := range onRejected {
			fn(err)
		}
	}

	for _, fn := range onFinally {
		fn()
	}
}

// Then registers callbacks for when the promise is fulfilled or rejected.
// If the promise is already settled, the callback is queued immediately.
func (p *Promise[T]) Then(onFulfilled func(T), onRejected func(error)) *Promise[T] {
	p.mu.Lock()
	state := p.state
	val := p.value
	err := p.err

	if state == Pending {
		if onFulfilled != nil {
			p.onFulfilled = append(p.onFulfilled, onFulfilled)
		}
		if onRejected != nil {
			p.onRejected = append(p.onRejected, onRejected)
		}
		p.mu.Unlock()
		return p
	}
	p.mu.Unlock()

	sched := getScheduler()
	if sched == nil {
		if state == Fulfilled && onFulfilled != nil {
			onFulfilled(val)
		} else if state == Rejected && onRejected != nil {
			onRejected(err)
		}
		return p
	}

	sched.QueueMicrotask(func() {
		if state == Fulfilled && onFulfilled != nil {
			onFulfilled(val)
		} else if state == Rejected && onRejected != nil {
			onRejected(err)
		}
	})

	return p
}

// Catch registers a callback for when the promise is rejected.
func (p *Promise[T]) Catch(onRejected func(error)) *Promise[T] {
	return p.Then(nil, onRejected)
}

// Finally registers a callback that is executed when the promise is settled.
func (p *Promise[T]) Finally(onFinally func()) *Promise[T] {
	p.mu.Lock()
	state := p.state

	if state == Pending {
		p.onFinally = append(p.onFinally, onFinally)
		p.mu.Unlock()
		return p
	}
	p.mu.Unlock()

	sched := getScheduler()
	if sched == nil {
		onFinally()
		return p
	}

	sched.QueueMicrotask(func() {
		onFinally()
	})

	return p
}

// Await blocks the caller until the promise is settled or the context is cancelled.
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-p.done:
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.value, p.err
	}
}

// Resolved returns a new Promise that is already fulfilled with the given value.
func Resolved[T any](val T) *Promise[T] {
	p := &Promise[T]{
		state: Fulfilled,
		value: val,
		done:  make(chan struct{}),
	}
	close(p.done)
	return p
}

// Rejected returns a new Promise that is already rejected with the given error.
func RejectedPromise[T any](err error) *Promise[T] {
	p := &Promise[T]{
		state: Rejected,
		err:   err,
		done:  make(chan struct{}),
	}
	close(p.done)
	return p
}
