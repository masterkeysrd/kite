package wind

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

type queuedScheduler struct {
	testScheduler
	background []func(context.Context)
}

func (s *queuedScheduler) RunBackground(task func(context.Context)) {
	s.background = append(s.background, task)
}

func (s *queuedScheduler) startBackground() <-chan struct{} {
	done := make(chan struct{})
	var wg sync.WaitGroup
	for _, task := range s.background {
		wg.Go(func() { task(context.Background()) })
	}
	s.background = nil
	go func() {
		wg.Wait()
		close(done)
	}()
	return done
}

func TestMutation_CancelsWrappedRequestsOnUnmount(t *testing.T) {
	sched := &queuedScheduler{}
	promise.SetScheduler(sched)
	defer promise.SetScheduler(nil)
	doc := dom.NewDocument()
	container := kitex.Box(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)
	started := make(chan context.Context, 2)
	var requests []*promise.Promise[int]
	callbacks := 0
	mutate := promise.WrapWithProps(func(ctx context.Context, variables int) (int, error) {
		started <- ctx
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Second):
			return 0, errors.New("mutation was not canceled")
		}
	})
	var mutation MutationResult[int]
	app := kitex.SimpleFC("App", func() kitex.Node {
		mutation = UseMutation(func(ctx context.Context, variables int) *promise.Promise[int] {
			p := mutate(ctx, variables)
			requests = append(requests, p)
			return p
		}, MutationOptions[int, int]{
			OnSuccess: func(int, int, MutationContext) { callbacks++ },
			OnError:   func(error, int, MutationContext) { callbacks++ },
		})
		return kitex.Box(kitex.BoxProps{})
	})
	client := NewClient()
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
	mutation.Mutate(1)
	mutation.Mutate(2)
	done := sched.startBackground()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var requestContexts []context.Context
	for range 2 {
		select {
		case requestCtx := <-started:
			requestContexts = append(requestContexts, requestCtx)
		case <-ctx.Done():
			t.Fatal("mutations did not start")
		}
	}
	// Rerendering must not replace or cancel the component lifetime.
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
	for _, requestCtx := range requestContexts {
		if requestCtx.Err() != nil {
			t.Error("rerender canceled an active mutation")
		}
	}
	kitex.Render(nil, container)
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("mutations did not stop on unmount")
	}
	for _, p := range requests {
		if _, err := p.Await(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("expected canceled wrapped request, got %v", err)
		}
	}
	sched.Flush()
	if callbacks != 0 {
		t.Errorf("expected no callbacks after unmount, got %d", callbacks)
	}
	mutation.Mutate(3)
	if len(requests) != 2 {
		t.Error("retained Mutate started work after unmount")
	}
}

func TestMutation_SuppressesSettledCallbacksAfterUnmount(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		name := "success"
		if rejected {
			name = "error"
		}
		t.Run(name, func(t *testing.T) {
			sched := &queuedScheduler{}
			promise.SetScheduler(sched)
			defer promise.SetScheduler(nil)
			doc := dom.NewDocument()
			container := kitex.Box(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
			defer kitex.Render(nil, container)
			callbacks := 0
			var mutation MutationResult[int]
			app := kitex.SimpleFC("App", func() kitex.Node {
				mutation = UseMutation(func(context.Context, int) *promise.Promise[int] {
					if rejected {
						return promise.RejectedPromise[int](errors.New("request failed"))
					}
					return promise.Resolved(42)
				}, MutationOptions[int, int]{
					OnSuccess: func(int, int, MutationContext) { callbacks++ },
					OnError:   func(error, int, MutationContext) { callbacks++ },
				})
				return kitex.Box(kitex.BoxProps{})
			})
			kitex.Render(Provider(ProviderProps{Client: NewClient()}, app()), container)
			mutation.Mutate(1)
			sched.Flush()
			if callbacks != 1 || mutation.IsPending || mutation.IsError != rejected {
				t.Fatalf("expected normal settlement while mounted; callbacks=%d result=%+v", callbacks, mutation)
			}
			mutation.Mutate(2)
			kitex.Render(nil, container)
			sched.Flush()
			if callbacks != 1 {
				t.Errorf("callback ran after unmount: %d calls", callbacks)
			}
		})
	}
}

func TestQuery_WrappedFetcherCancellationOnUnmount(t *testing.T) {
	sched := &queuedScheduler{}
	promise.SetScheduler(sched)
	defer promise.SetScheduler(nil)
	doc := dom.NewDocument()
	container := kitex.Box(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)
	started := make(chan context.Context, 1)
	fetch := promise.WrapWithProps(func(ctx context.Context, key string) (string, error) {
		started <- ctx
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
			return "", errors.New("query was not canceled")
		}
	})
	app := kitex.SimpleFC("App", func() kitex.Node {
		Use("key", fetch)
		return kitex.Box(kitex.BoxProps{})
	})
	kitex.Render(Provider(ProviderProps{Client: NewClient()}, app()), container)
	done := sched.startBackground()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var requestCtx context.Context
	select {
	case requestCtx = <-started:
	case <-ctx.Done():
		t.Fatal("wrapped query did not start")
	}
	kitex.Render(nil, container)
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("wrapped query did not stop on unmount")
	}
	if !errors.Is(requestCtx.Err(), context.Canceled) {
		t.Errorf("expected query context cancellation, got %v", requestCtx.Err())
	}
	sched.Flush()
}
