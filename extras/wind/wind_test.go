package wind

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

// capture hands a value written during a component render to the test
// goroutine. Renders run on whichever goroutine flushed the dirty queue, so
// promise callbacks and stream events re-render off the test goroutine and a
// plain variable would be a data race.
type capture[T any] struct {
	mu    sync.Mutex
	value T
}

func (c *capture[T]) set(v T) {
	c.mu.Lock()
	c.value = v
	c.mu.Unlock()
}

func (c *capture[T]) get() T {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// waitFor polls cond until it holds or the timeout elapses. It replaces fixed
// sleeps, which are both flaky under -race and a source of false positives.
func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return cond()
}

type mutationSuccess struct {
	result    string
	variables int
	client    *Client
}

// counter tallies fetch or connection attempts that happen on worker
// goroutines.
type counter struct {
	n atomic.Int64
}

func (c *counter) inc() int { return int(c.n.Add(1)) }
func (c *counter) get() int { return int(c.n.Load()) }

// recorder collects every value a component renders. Tests that need to assert
// on an early render pass (for example the initial loading state) cannot read a
// single "latest" value, because a settled fetch may already have re-rendered
// with new data by the time the test looks.
type recorder[T any] struct {
	mu     sync.Mutex
	values []T
}

func (r *recorder[T]) add(v T) {
	r.mu.Lock()
	r.values = append(r.values, v)
	r.mu.Unlock()
}

func (r *recorder[T]) all() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.values)
}

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

	var result capture[Result[string]]
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("test_key", fetcher)
		result.set(res)
		return kitex.Box(kitex.BoxProps{ID: "box"}, kitex.Text(res.Data))
	})

	// Render initially
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool { return result.get().Data == "value_1" }) {
		t.Fatalf("expected initial data to be value_1, got %q", result.get().Data)
	}
	if result.get().IsFetching {
		t.Fatal("expected isFetching to be false after completion")
	}

	// Invalidate the query
	client.InvalidateQueries("test_key")

	// The query should transition to isFetching = true
	if !waitFor(t, func() bool { return result.get().IsFetching }) {
		t.Error("expected IsFetching to be true after InvalidateQueries")
	}

	// Wait for the second fetch to complete
	if !waitFor(t, func() bool { return result.get().Data == "value_2" }) {
		t.Errorf("expected updated data to be value_2, got %q", result.get().Data)
	}
	if result.get().IsFetching {
		t.Error("expected IsFetching to be false after invalidation fetch completes")
	}
}

func TestMutation(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	client := NewClient()

	var success capture[mutationSuccess]

	mutOpts := MutationOptions[int, string]{
		OnSuccess: func(res string, vars int, ctx MutationContext) {
			success.set(mutationSuccess{result: res, variables: vars, client: ctx.Client})
		},
	}

	var mutation capture[MutationResult[int]]
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := UseMutation(func(ctx context.Context, vars int) *promise.Promise[string] {
			return promise.New(func(ctx context.Context) (string, error) {
				return "mut_res", nil
			})
		}, mutOpts)
		mutation.set(res)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if mutation.get().IsPending {
		t.Fatal("expected mutation not to be pending initially")
	}

	mutation.get().Mutate(42)

	if !waitFor(t, func() bool { return success.get().client != nil }) {
		t.Fatal("expected onSuccess to fire")
	}
	if success.get().result != "mut_res" {
		t.Errorf("expected onSuccess result 'mut_res', got %q", success.get().result)
	}
	if success.get().variables != 42 {
		t.Errorf("expected onSuccess variables 42, got %d", success.get().variables)
	}
	if success.get().client != client {
		t.Errorf("expected onSuccess client to match, got %v", success.get().client)
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
			for i := range numSubs {
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

	for b.Loop() {
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

	for b.Loop() {
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

	var observed recorder[Result[string]]
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("test_loading_key", fetcher)
		observed.add(res)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	// Render initially. On this very first render, the hook should report IsLoading as true.
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	first := observed.all()[0]
	if !first.IsLoading {
		t.Error("expected IsLoading to be true on the first render pass")
	}
	if first.Data != "" {
		t.Errorf("expected initial data to be empty, got %q", first.Data)
	}

	// After the fetch settles, a later render pass should report it complete.
	if !waitFor(t, func() bool {
		for _, r := range observed.all() {
			if r.Data == "data" {
				return true
			}
		}
		return false
	}) {
		t.Error("expected a render pass with settled data 'data'")
	}

	latestAll := observed.all()
	latest := latestAll[len(latestAll)-1]
	if latest.IsLoading {
		t.Error("expected IsLoading to be false after query settles")
	}
	if latest.Data != "data" {
		t.Errorf("expected settled data to be 'data', got %q", latest.Data)
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
			for i := range 10 {
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
	for range 10 {
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
	var startCount counter

	starts := startCount.get

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			startCount.inc()
			yield("val", nil)
		}
	}

	var capturedResult capture[StreamResult[string]]
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := UseStream("reconnect_key", fetcher)
		capturedResult.set(res)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool { return starts() == 1 }) {
		t.Fatalf("expected stream to connect initially, got %d", starts())
	}

	// Trigger manual refetch (reconnection)
	capturedResult.get().Refetch()

	if !waitFor(t, func() bool { return starts() == 2 }) {
		t.Errorf("expected stream to reconnect, total starts should be 2, got %d", starts())
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
	var fetchCount counter

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		currentCount := fetchCount.inc()
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

	var result capture[Result[string]]
	app := kitex.SimpleFC("App", func() kitex.Node {
		res := Use("test_cancel_key", fetcher)
		result.set(res)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(res.Data))
	})

	// Mount initially
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	// Wait for the first fetch to start in background
	<-fetchStarted

	// Call InvalidateQueries while the first fetch is in progress
	client.InvalidateQueries("test_cancel_key")

	// Wait for the second fetch to start (starts after the first fetch finishes)
	<-fetchStarted

	// Wait for the second fetch to land in the rendered result
	if !waitFor(t, func() bool { return result.get().Data == "val_2" }) {
		t.Errorf("expected final data to be 'val_2' from the second fetch, got %q", result.get().Data)
	}

	if count := fetchCount.get(); count != 2 {
		t.Fatalf("expected fetch to be triggered exactly 2 times, got %d", count)
	}
}
