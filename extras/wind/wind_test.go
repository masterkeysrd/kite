package wind

import (
	"context"
	"fmt"
	"iter"
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

func TestQueryInitialLoadingState(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "data", nil
		})
	}

	var capturedIsLoading bool
	var capturedData string
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("test_loading_key", fetcher)
		capturedIsLoading = res.IsLoading
		capturedData = res.Data
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	// Render initially. On this very first render, the hook should report IsLoading as true.
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !capturedIsLoading {
		t.Error("expected IsLoading to be true on the first render pass")
	}
	if capturedData != "" {
		t.Errorf("expected initial data to be empty, got %q", capturedData)
	}

	// Wait for the fetch to complete
	time.Sleep(20 * time.Millisecond)

	// After completion, the state should update and report IsLoading as false.
	if capturedIsLoading {
		t.Error("expected IsLoading to be false after query settles")
	}
	if capturedData != "data" {
		t.Errorf("expected settled data to be 'data', got %q", capturedData)
	}
}

func TestStream_UnmountCancellation(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)

	client := NewClient()
	ctxCancelled := make(chan struct{})

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			select {
			case <-ctx.Done():
				close(ctxCancelled)
				return
			case <-time.After(1 * time.Second):
				yield("data", nil)
			}
		}
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		res := UseStream("stream_cancel", fetcher)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	// Mount
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	time.Sleep(10 * time.Millisecond)

	// Unmount
	kitex.Render(nil, container)

	select {
	case <-ctxCancelled:
		// Success!
	case <-time.After(1 * time.Second):
		t.Fatal("expected stream context to be cancelled on unmount")
	}
}

func TestStream_Deduplication(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()
	var streamCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			mu.Lock()
			streamCount++
			mu.Unlock()
			yield("initial", nil)
		}
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{},
			ChildStreamComp(ChildStreamProps{Key: "same_key", Fetcher: fetcher}),
			ChildStreamComp(ChildStreamProps{Key: "same_key", Fetcher: fetcher}),
		)
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	count := streamCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("expected stream to be created once, got %d", count)
	}
}

type ChildStreamProps struct {
	Key     string
	Fetcher func(context.Context, string) iter.Seq2[string, error]
}

var ChildStreamComp = kitex.FC("ChildStreamComp", func(props ChildStreamProps) kitex.Node {
	res := UseStream(props.Key, props.Fetcher)
	return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
})

func TestStream_Coalescing(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	// Set test scheduler
	sched := &testScheduler{}
	promise.SetScheduler(sched)
	defer promise.SetScheduler(nil)

	client := NewClient()

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			// Yield 10 values in a loop as fast as possible
			for i := 0; i < 10; i++ {
				yield(fmt.Sprintf("val_%d", i), nil)
			}
		}
	}

	var renderCount int
	var mu sync.Mutex
	var lastData string

	app := kitex.SimpleFC("App", func() kitex.Node {
		res := UseStream("coalesce_key", fetcher)
		mu.Lock()
		renderCount++
		lastData = res.Data
		mu.Unlock()
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait for background events to queue their microtasks
	time.Sleep(30 * time.Millisecond)

	// Flush queued microtasks (coalesced pass)
	sched.Flush()

	// Wait a tiny bit more for layout effects if any
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	count := renderCount
	data := lastData
	mu.Unlock()

	// 1 for initial render (connecting/loading state), and up to 3 for update and cleanup passes.
	if count > 4 {
		t.Errorf("expected render count to be coalesced (<= 4 passes), got %d", count)
	}
	if data != "val_9" {
		t.Errorf("expected last data to be val_9, got %q", data)
	}
}

func TestStream_InvalidationCoalescing(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	// Set test scheduler
	sched := &testScheduler{}
	promise.SetScheduler(sched)
	defer promise.SetScheduler(nil)

	client := NewClient()
	var startCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			mu.Lock()
			startCount++
			mu.Unlock()
			yield("value", nil)
		}
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		res := UseStream("invalidate_key", fetcher)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait and flush initial connection
	time.Sleep(20 * time.Millisecond)
	sched.Flush()

	mu.Lock()
	initialStarts := startCount
	mu.Unlock()

	if initialStarts != 1 {
		t.Fatalf("expected stream to start once initially, got %d", initialStarts)
	}

	// Trigger invalidations 10 times in a tight loop
	for i := 0; i < 10; i++ {
		client.InvalidateQueries("invalidate_key")
	}

	// Flush the coalesced invalidation microtask to trigger the reconnect/executeStream call
	sched.Flush()

	// Wait for the reconnected background stream to start and increment startCount
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	finalStarts := startCount
	mu.Unlock()

	// It should have connected exactly once initially, and exactly once for the coalesced invalidations.
	if finalStarts != 2 {
		t.Errorf("expected exactly 2 starts total (1 initial + 1 coalesced invalidation), got %d", finalStarts)
	}
}

func TestStream_ManualReconnection(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()
	var startCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			mu.Lock()
			startCount++
			mu.Unlock()
			yield("val", nil)
		}
	}

	var capturedResult StreamResult[string]
	app := kitex.SimpleFC("App", func() kitex.Node {
		capturedResult = UseStream("reconnect_key", fetcher)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	count1 := startCount
	mu.Unlock()

	if count1 != 1 {
		t.Fatalf("expected stream to connect initially, got %d", count1)
	}

	// Trigger manual refetch (reconnection)
	capturedResult.Refetch()

	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	count2 := startCount
	mu.Unlock()

	if count2 != 2 {
		t.Errorf("expected stream to reconnect, total starts should be 2, got %d", count2)
	}
}

type testScheduler struct {
	mu         sync.Mutex
	microtasks []func()
}

func (s *testScheduler) RunBackground(task func(ctx context.Context)) {
	go task(context.Background())
}

func (s *testScheduler) QueueMicrotask(task func()) {
	s.mu.Lock()
	s.microtasks = append(s.microtasks, task)
	s.mu.Unlock()
}

func (s *testScheduler) QueueMacrotask(task func()) {
	s.QueueMicrotask(task)
}

func (s *testScheduler) Flush() {
	for {
		s.mu.Lock()
		if len(s.microtasks) == 0 {
			s.mu.Unlock()
			break
		}
		tasks := s.microtasks
		s.microtasks = nil
		s.mu.Unlock()

		for _, t := range tasks {
			t()
		}
	}
}

func TestCacheEviction(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()
	client.GcTime = 10 * time.Millisecond

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return "data", nil
		})
	}

	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("evict_key", fetcher)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	// Mount & fetch
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
	time.Sleep(20 * time.Millisecond)

	// Verify it's in the cache
	client.mu.Lock()
	_, foundBefore := client.cache["evict_key"]
	client.mu.Unlock()
	if !foundBefore {
		t.Fatal("expected query entry to be in cache after mount")
	}

	// Unmount
	kitex.Render(nil, container)

	// Verify that the entry is NOT removed immediately if GcTime is not yet elapsed
	client.mu.Lock()
	_, foundAfterImmediate := client.cache["evict_key"]
	client.mu.Unlock()
	if !foundAfterImmediate {
		t.Fatal("expected query entry to still be in cache immediately after unmount before GcTime")
	}

	// Wait for eviction timer to elapse
	time.Sleep(30 * time.Millisecond)

	// Verify it's evicted from the cache
	client.mu.Lock()
	_, foundAfterGc := client.cache["evict_key"]
	client.mu.Unlock()
	if foundAfterGc {
		t.Error("expected query entry to be evicted from cache after GcTime has passed")
	}
}

func TestInvalidateQueries_PrefixAndPredicateMatching(t *testing.T) {
	client := NewClient()

	var refetchCount1, refetchCount2, refetchCount3 int
	var mu sync.Mutex

	entry1 := &queryEntry{
		key:         [2]string{"todos", "list"},
		state:       queryState{status: "success"},
		subscribers: map[int]func(){1: func() {}},
	}
	entry1.refetch = func() {
		mu.Lock()
		refetchCount1++
		mu.Unlock()
	}

	entry2 := &queryEntry{
		key:         [2]string{"todos", "detail"},
		state:       queryState{status: "success"},
		subscribers: map[int]func(){1: func() {}},
	}
	entry2.refetch = func() {
		mu.Lock()
		refetchCount2++
		mu.Unlock()
	}

	entry3 := &queryEntry{
		key:         [2]string{"users", "list"},
		state:       queryState{status: "success"},
		subscribers: map[int]func(){1: func() {}},
	}
	entry3.refetch = func() {
		mu.Lock()
		refetchCount3++
		mu.Unlock()
	}

	client.cache[entry1.key] = entry1
	client.cache[entry2.key] = entry2
	client.cache[entry3.key] = entry3

	// 1. Invalidate with Prefix []string{"todos"}
	client.InvalidateQueries([]string{"todos"})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	c1, c2, c3 := refetchCount1, refetchCount2, refetchCount3
	mu.Unlock()

	if c1 != 1 || c2 != 1 {
		t.Errorf("expected entry1 and entry2 to be refetched once, got c1=%d, c2=%d", c1, c2)
	}
	if c3 != 0 {
		t.Errorf("expected entry3 to not be refetched, got %d", c3)
	}

	// 2. Invalidate with Predicate
	var predCount1, predCount2, predCount3 int
	entry1.refetch = func() { mu.Lock(); predCount1++; mu.Unlock() }
	entry2.refetch = func() { mu.Lock(); predCount2++; mu.Unlock() }
	entry3.refetch = func() { mu.Lock(); predCount3++; mu.Unlock() }

	pred := func(k any) bool {
		arr, ok := k.([2]string)
		return ok && (arr[1] == "list")
	}

	client.InvalidateQueries(pred)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	p1, p2, p3 := predCount1, predCount2, predCount3
	mu.Unlock()

	if p1 != 1 || p3 != 1 {
		t.Errorf("expected entry1 and entry3 (list keys) to be refetched once, got p1=%d, p3=%d", p1, p3)
	}
	if p2 != 0 {
		t.Errorf("expected entry2 (detail key) to not be refetched, got %d", p2)
	}
}

func TestInvalidateQueries_SequencesActiveFetch(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	var fetchStarted = make(chan struct{}, 5)
	var fetchCount int
	var mu sync.Mutex

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		mu.Lock()
		fetchCount++
		currentCount := fetchCount
		mu.Unlock()
		fetchStarted <- struct{}{}
		return promise.New(func(ctx context.Context) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(15 * time.Millisecond):
				return fmt.Sprintf("val_%d", currentCount), nil
			}
		})
	}

	var result Result[string]
	app := kitex.SimpleFC("App", func() kitex.Node {
		result = Use("test_cancel_key", fetcher)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(result.Data))
	})

	// Mount initially
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait for the first fetch to start in background
	<-fetchStarted

	// Call InvalidateQueries while the first fetch is in progress
	client.InvalidateQueries("test_cancel_key")

	// Wait for the second fetch to start (starts after the first fetch finishes)
	<-fetchStarted

	// Wait for the second fetch to complete
	time.Sleep(25 * time.Millisecond)

	mu.Lock()
	count := fetchCount
	mu.Unlock()

	if count != 2 {
		t.Fatalf("expected fetch to be triggered exactly 2 times, got %d", count)
	}

	if result.Data != "val_2" {
		t.Errorf("expected final data to be 'val_2' from the second fetch, got %q", result.Data)
	}
}
