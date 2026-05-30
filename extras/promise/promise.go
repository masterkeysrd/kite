package promise

import (
	"context"
	"sync"

	"github.com/masterkeysrd/kite/terminal"
)

var (
	globalScheduler terminal.Scheduler
	schedMu         sync.RWMutex
)

// SetScheduler sets the global scheduler used by all promises.
func SetScheduler(s terminal.Scheduler) {
	schedMu.Lock()
	defer schedMu.Unlock()
	globalScheduler = s
}

func getScheduler() terminal.Scheduler {
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
func New[T any](ctx context.Context, executor func(context.Context) (T, error)) *Promise[T] {
	p := &Promise[T]{
		state: Pending,
		done:  make(chan struct{}),
	}

	sched := getScheduler()
	if sched == nil {
		// Fallback for tests or uninitialized environments
		go p.run(ctx, executor)
		return p
	}

	sched.RunBackground(func() {
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

	if state == Fulfilled {
		for _, fn := range onFulfilled {
			fn(val)
		}
	} else if state == Rejected {
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
