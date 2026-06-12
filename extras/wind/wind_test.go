package wind

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

func TestContextCancellationOnUnmount(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)

	client := NewClient()

	ctxCancelled := make(chan struct{})
	fetcher := func(windCtx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(schedCtx context.Context) (string, error) {
			select {
			case <-windCtx.Done():
				close(ctxCancelled)
				return "", windCtx.Err()
			case <-schedCtx.Done():
				return "", schedCtx.Err()
			case <-time.After(2 * time.Second):
				return "done", nil
			}
		})
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("test_cancel", fetcher)
		return kitex.Box(kitex.BoxProps{ID: "box"}, kitex.Text(res.Data))
	})

	// Mount
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait a tiny bit to let the fetcher start in background
	time.Sleep(10 * time.Millisecond)

	// Unmount
	kitex.Render(nil, container)

	// The context should be cancelled immediately upon unmount
	select {
	case <-ctxCancelled:
		// Success!
	case <-time.After(1 * time.Second):
		t.Fatal("expected fetcher context to be cancelled on unmount")
	}
}

type ChildProps struct {
	Key     string
	Fetcher func(context.Context, string) *promise.Promise[string]
}

var ChildQueryComp = kitex.FC("ChildQueryComp", func(props ChildProps) kitex.Node {
	res := Use(props.Key, props.Fetcher)
	return kitex.Box(kitex.BoxProps{ID: "child"}, kitex.Text(res.Data))
})

func TestRequestDeduping(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	var fetchCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		mu.Lock()
		fetchCount++
		mu.Unlock()
		return promise.New(func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "data", nil
		})
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{},
			ChildQueryComp(ChildProps{Key: "same_key", Fetcher: fetcher}),
			ChildQueryComp(ChildProps{Key: "same_key", Fetcher: fetcher}),
		)
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait for the fetch to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := fetchCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("expected fetcher to be called exactly once (deduped), got %d", count)
	}
}

func TestQueryInvalidation(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	var fetchCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		mu.Lock()
		fetchCount++
		currentCount := fetchCount
		mu.Unlock()
		return promise.New(func(ctx context.Context) (string, error) {
			return "value_" + string(rune('0'+currentCount)), nil
		})
	}

	var result Result[string]
	app := kitex.SimpleFC("App", func() kitex.Node {
		result = Use("test_key", fetcher)
		return kitex.Box(kitex.BoxProps{ID: "box"}, kitex.Text(result.Data))
	})

	// Render initially
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait for fetcher to complete
	time.Sleep(20 * time.Millisecond)

	if result.Data != "value_1" {
		t.Fatalf("expected initial data to be value_1, got %q", result.Data)
	}
	if result.IsFetching {
		t.Fatal("expected isFetching to be false after completion")
	}

	// Invalidate the query
	client.InvalidateQueries("test_key")

	// The query should immediately transition to isFetching = true
	if !result.IsFetching {
		t.Error("expected IsFetching to be true immediately after InvalidateQueries")
	}

	// Wait for the second fetch to complete
	time.Sleep(20 * time.Millisecond)

	if result.Data != "value_2" {
		t.Errorf("expected updated data to be value_2, got %q", result.Data)
	}
	if result.IsFetching {
		t.Error("expected IsFetching to be false after invalidation fetch completes")
	}
}

func TestMutation(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	var successResult string
	var successVariables int
	var successClient *Client

	mutOpts := MutationOptions[int, string]{
		OnSuccess: func(res string, vars int, ctx MutationContext) {
			successResult = res
			successVariables = vars
			successClient = ctx.Client
		},
	}

	var mutation MutationResult[int]
	app := kitex.SimpleFC("App", func() kitex.Node {
		mutation = UseMutation(func(ctx context.Context, vars int) *promise.Promise[string] {
			return promise.New(func(ctx context.Context) (string, error) {
				return "mut_res", nil
			})
		}, mutOpts)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if mutation.IsPending {
		t.Fatal("expected mutation not to be pending initially")
	}

	mutation.Mutate(42)

	// Wait for background mutation to complete
	time.Sleep(20 * time.Millisecond)

	if successResult != "mut_res" {
		t.Errorf("expected onSuccess result 'mut_res', got %q", successResult)
	}
	if successVariables != 42 {
		t.Errorf("expected onSuccess variables 42, got %d", successVariables)
	}
	if successClient != client {
		t.Errorf("expected onSuccess client to match, got %v", successClient)
	}
}

func BenchmarkClient_InvalidateQueries(b *testing.B) {
	for _, numSubs := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("%d_subscribers", numSubs), func(b *testing.B) {
			client := NewClient()
			entry := &queryEntry{
				key: "key",
				state: queryState{
					status: "success",
				},
				subscribers: make(map[int]func()),
			}
			entry.refetch = func() {
				entry.mu.Lock()
				entry.state.isFetching = true
				entry.mu.Unlock()
				entry.notifySubscribers()
				entry.mu.Lock()
				entry.state.isFetching = false
				entry.mu.Unlock()
			}
			client.cache["key"] = entry

			entry.mu.Lock()
			for i := 0; i < numSubs; i++ {
				entry.subscribers[i] = func() {}
			}
			entry.mu.Unlock()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				client.InvalidateQueries("key")
			}
		})
	}
}

func BenchmarkUseQuery(b *testing.B) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return "data", nil
		})
	}

	comp := kitex.SimpleFC("Comp", func() kitex.Node {
		res := Use("key", fetcher)
		return kitex.Box(kitex.BoxProps{ID: res.Data})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kitex.Render(Provider(ProviderProps{Client: client}, comp()), container)
	}
}

func BenchmarkUseMutation(b *testing.B) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	mutationFn := func(ctx context.Context, val int) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return "ok", nil
		})
	}

	comp := kitex.SimpleFC("Comp", func() kitex.Node {
		res := UseMutation(mutationFn)
		var char rune
		if res.IsPending {
			char = 'Y'
		} else {
			char = 'N'
		}
		return kitex.Box(kitex.BoxProps{}, kitex.Text(string(char)))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kitex.Render(Provider(ProviderProps{Client: client}, comp()), container)
	}
}
