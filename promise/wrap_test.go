package promise

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func contextWrappers() map[string]func(context.Context, func(context.Context) (int, error)) *Promise[int] {
	return map[string]func(context.Context, func(context.Context) (int, error)) *Promise[int]{
		"Wrap": func(ctx context.Context, fn func(context.Context) (int, error)) *Promise[int] {
			return Wrap(fn)(ctx)
		},
		"WrapWithProps": func(ctx context.Context, fn func(context.Context) (int, error)) *Promise[int] {
			return WrapWithProps(func(ctx context.Context, props int) (int, error) {
				if props != 42 {
					return 0, errors.New("properties not forwarded")
				}
				return fn(ctx)
			})(ctx, 42)
		},
	}
}

func TestWrap_PreservesCallerContext(t *testing.T) {
	for name, wrap := range contextWrappers() {
		for _, scheduled := range []bool{false, true} {
			mode := "fallback"
			if scheduled {
				mode = "scheduled"
			}
			t.Run(name+"/"+mode, func(t *testing.T) {
				sched := &mockScheduler{}
				SetScheduler(nil)
				if scheduled {
					SetScheduler(sched)
				}
				t.Cleanup(func() { SetScheduler(nil) })
				type contextKey struct{}
				deadline := time.Now().Add(time.Minute)
				ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), contextKey{}, "caller"), deadline)
				defer cancel()
				p := wrap(ctx, func(got context.Context) (int, error) {
					if got.Value(contextKey{}) != "caller" {
						t.Error("wrapped function lost caller context value")
					}
					if actual, ok := got.Deadline(); !ok || !actual.Equal(deadline) {
						t.Errorf("wrapped function lost caller deadline: %v, %t", actual, ok)
					}
					return 42, nil
				})
				if scheduled {
					sched.flushBackground()
					sched.flushMicrotasks()
				}
				wait, stop := context.WithTimeout(context.Background(), time.Second)
				defer stop()
				if value, err := p.Await(wait); err != nil || value != 42 {
					t.Fatalf("expected 42, nil; got %d, %v", value, err)
				}
			})
		}
	}
}

func TestWrap_Cancellation(t *testing.T) {
	for name, wrap := range contextWrappers() {
		for _, source := range []string{"caller", "scheduler"} {
			for _, beforeStart := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/beforeStart=%t", name, source, beforeStart), func(t *testing.T) {
					sched := &mockScheduler{}
					SetScheduler(sched)
					t.Cleanup(func() { SetScheduler(nil) })
					callerCtx, cancelCaller := context.WithCancelCause(context.Background())
					defer cancelCaller(nil)
					schedulerCtx, cancelScheduler := context.WithCancelCause(context.Background())
					defer cancelScheduler(nil)
					cause := errors.New("request stopped")
					cancel := func() { cancelCaller(cause) }
					if source == "scheduler" {
						cancel = func() { cancelScheduler(cause) }
					}
					if beforeStart {
						cancel()
					}
					called := false
					p := wrap(callerCtx, func(ctx context.Context) (int, error) {
						called = true
						cancel()
						select {
						case <-ctx.Done():
							if !errors.Is(context.Cause(ctx), cause) {
								t.Errorf("lost cancellation cause: %v", context.Cause(ctx))
							}
							return 0, ctx.Err()
						case <-time.After(time.Second):
							return 0, errors.New("wrapped context was not canceled")
						}
					})
					sched.backgroundTasks[0](schedulerCtx)
					sched.flushMicrotasks()
					if _, err := p.Await(context.Background()); !errors.Is(err, context.Canceled) {
						t.Errorf("expected cancellation, got %v", err)
					}
					if called == beforeStart {
						t.Errorf("wrapped function called=%t, beforeStart=%t", called, beforeStart)
					}
				})
			}
		}
	}
}

func TestWrap_ExpiredDeadline(t *testing.T) {
	for name, wrap := range contextWrappers() {
		t.Run(name, func(t *testing.T) {
			SetScheduler(nil)
			t.Cleanup(func() { SetScheduler(nil) })
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			p := wrap(ctx, func(context.Context) (int, error) {
				t.Error("wrapped function executed after its deadline")
				return 42, nil
			})
			if _, err := p.Await(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("expected deadline exceeded, got %v", err)
			}
		})
	}
}

func TestWrap_CallerCancellationWithoutScheduler(t *testing.T) {
	for name, wrap := range contextWrappers() {
		t.Run(name, func(t *testing.T) {
			SetScheduler(nil)
			t.Cleanup(func() { SetScheduler(nil) })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := wrap(ctx, func(ctx context.Context) (int, error) {
				cancel()
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(time.Second):
					return 0, errors.New("caller cancellation was lost")
				}
			})
			if _, err := p.Await(context.Background()); !errors.Is(err, context.Canceled) {
				t.Errorf("expected cancellation, got %v", err)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	SetScheduler(nil)

	syncFn := func(ctx context.Context) (string, error) {
		return "wrapped", nil
	}

	asyncFn := Wrap(syncFn)
	p := asyncFn(context.Background())

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val != "wrapped" {
		t.Errorf("expected 'wrapped', got %q", val)
	}
}

func TestWrapWithProps(t *testing.T) {
	SetScheduler(nil)

	type Props struct {
		ID int
	}

	syncFn := func(ctx context.Context, p Props) (int, error) {
		return p.ID * 2, nil
	}

	asyncFn := WrapWithProps(syncFn)
	p := asyncFn(context.Background(), Props{ID: 21})

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}
