package wind

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

type boundedPoolScheduler struct {
	jobs      chan func(context.Context)
	micros    chan func()
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func newBoundedPoolScheduler(workers int) *boundedPoolScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &boundedPoolScheduler{
		jobs:   make(chan func(context.Context), 64),
		micros: make(chan func(), 256),
		ctx:    ctx,
		cancel: cancel,
	}
	for range workers {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-s.jobs:
					job(ctx)
				}
			}
		}()
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-s.micros:
				task()
			}
		}
	}()
	return s
}

func (s *boundedPoolScheduler) RunBackground(task func(context.Context)) {
	select {
	case <-s.ctx.Done():
	case s.jobs <- task:
	}
}

func (s *boundedPoolScheduler) Spawn(task func(context.Context)) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		task(s.ctx)
	}()
}

func (s *boundedPoolScheduler) QueueMicrotask(task func()) {
	select {
	case <-s.ctx.Done():
	case s.micros <- task:
	}
}

func (s *boundedPoolScheduler) QueueMacrotask(task func()) {
	s.QueueMicrotask(task)
}

func (s *boundedPoolScheduler) Close() {
	s.closeOnce.Do(func() {
		s.cancel()
		s.wg.Wait()
	})
}

func TestOptional(t *testing.T) {
	none := None[bool]()
	if none.IsSet() {
		t.Fatal("expected None().IsSet() to be false")
	}
	if none.Value() {
		t.Fatal("expected None().Value() to be zero value false")
	}
	if v, ok := none.Get(); ok || v {
		t.Fatalf("expected None().Get() == (false, false), got (%v, %v)", v, ok)
	}
	if !none.UnwrapOr(true) {
		t.Fatal("expected None().UnwrapOr(true) to return true")
	}

	someFalse := Some(false)
	if !someFalse.IsSet() {
		t.Fatal("expected Some(false).IsSet() to be true")
	}
	if someFalse.Value() {
		t.Fatal("expected Some(false).Value() to be false")
	}
	if v, ok := someFalse.Get(); !ok || v {
		t.Fatalf("expected Some(false).Get() == (false, true), got (%v, %v)", v, ok)
	}
	if someFalse.UnwrapOr(true) {
		t.Fatal("expected Some(false).UnwrapOr(true) to return false")
	}

	someZeroDur := Some(time.Duration(0))
	if !someZeroDur.IsSet() {
		t.Fatal("expected Some(0).IsSet() to be true")
	}
	if got := someZeroDur.UnwrapOr(16 * time.Millisecond); got != 0 {
		t.Fatalf("expected Some(0).UnwrapOr(16ms) == 0, got %v", got)
	}
}

func TestOptions_StaleTimeDoesNotDisable(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	var fetches counter
	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		fetches.inc()
		return promise.Resolved("ok")
	}

	var res capture[Result[string]]
	app := kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
		r := Use("stale_opts_key", fetcher, Options{StaleTime: time.Minute})
		res.set(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool { return res.get().Data == "ok" }) {
		t.Fatalf("expected Options{StaleTime: time.Minute} to stay enabled and resolve, got %+v", res.get())
	}
	if fetches.get() != 1 {
		t.Fatalf("expected 1 fetch, got %d", fetches.get())
	}
}

func TestStreamOptions_ThrottleZeroKeepsEnabled(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	var connects counter
	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			connects.inc()
			yield("instant", nil)
		}
	}

	var res capture[StreamResult[string]]
	app := kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
		r := UseStream("throttle_zero_key", fetcher, StreamOptions{
			Throttle: Some(time.Duration(0)),
		})
		res.set(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool { return res.get().Data == "instant" && res.get().Status == "closed" }) {
		t.Fatalf("expected StreamOptions{Throttle: Some(0)} to remain enabled and receive data, got %+v", res.get())
	}
	if connects.get() != 1 {
		t.Fatalf("expected 1 connection, got %d", connects.get())
	}
}

func TestStream_RemountWithinGcTime(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	client.GcTime = time.Second
	t.Cleanup(client.Close)

	var connects counter
	releaseSecond := make(chan struct{})

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			n := connects.inc()
			if n == 1 {
				if !yield("msg1", nil) {
					return
				}
				<-ctx.Done()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-releaseSecond:
			}
			yield("msg2", nil)
		}
	}

	var observed recorder[StreamResult[string]]
	app := kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
		r := UseStream("remount_key", fetcher)
		observed.add(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	// First mount
	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
	if !waitFor(t, func() bool {
		vals := observed.all()
		return len(vals) > 0 && vals[len(vals)-1].Data == "msg1"
	}) {
		t.Fatal("expected initial mount to receive msg1")
	}

	// Unmount within GcTime
	kitex.Render(nil, container)

	// Remount before GcTime expires
	var remountObserved recorder[StreamResult[string]]
	app2 := kitex.FC("App2", func(_ kitex.NoProps) kitex.Node {
		r := UseStream("remount_key", fetcher)
		remountObserved.add(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app2()), container)

	firstRemount := remountObserved.all()[0]
	if firstRemount.Data != "msg1" {
		t.Fatalf("expected remount within GcTime to immediately return stale Data 'msg1', got %q", firstRemount.Data)
	}
	if !waitFor(t, func() bool { return connects.get() == 2 }) {
		t.Fatalf("expected stream to reconnect on remount, connects=%d", connects.get())
	}

	close(releaseSecond)
	if !waitFor(t, func() bool {
		vals := remountObserved.all()
		return len(vals) > 0 && vals[len(vals)-1].Data == "msg2"
	}) {
		t.Fatal("expected reconnected stream to deliver msg2")
	}
}

func TestStream_EnabledToggleCancellation(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	var connects counter
	var cancelled1 = make(chan struct{})

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			n := connects.inc()
			if n == 1 {
				if !yield("first", nil) {
					return
				}
				<-ctx.Done()
				close(cancelled1)
				return
			}
			if !yield("second", nil) {
				return
			}
			<-ctx.Done()
		}
	}

	type props struct {
		Enabled bool
	}

	var res capture[StreamResult[string]]
	comp := kitex.FC("ToggleStream", func(p props) kitex.Node {
		r := UseStream("toggle_key", fetcher, StreamOptions{
			Enabled: Some(p.Enabled),
		})
		res.set(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	// 1. Mount with Enabled=true
	kitex.Render(Provider(ProviderProps{Client: client}, comp.Props(props{Enabled: true})), container)
	if !waitFor(t, func() bool { return res.get().Data == "first" && res.get().IsFetching && res.get().Status == "open" }) {
		t.Fatalf("expected initial stream open with 'first', got %+v", res.get())
	}

	// 2. Toggle Enabled=false: cancels active stream and notifies passive subscriber
	kitex.Render(Provider(ProviderProps{Client: client}, comp.Props(props{Enabled: false})), container)

	select {
	case <-cancelled1:
	case <-time.After(time.Second):
		t.Fatal("expected stream context to cancel when Enabled toggles true -> false")
	}

	if !waitFor(t, func() bool { return !res.get().IsFetching && res.get().Status == "closed" }) {
		t.Fatalf("expected passive subscriber to observe IsFetching=false / Status=closed, got %+v", res.get())
	}

	// 3. Toggle Enabled=true: reconnects and receives new data
	kitex.Render(Provider(ProviderProps{Client: client}, comp.Props(props{Enabled: true})), container)
	if !waitFor(t, func() bool { return connects.get() == 2 && res.get().Data == "second" }) {
		t.Fatalf("expected reconnect on false -> true, connects=%d res=%+v", connects.get(), res.get())
	}
}

func TestStream_MultiConsumerEnabledToggleAndFiniteDedupe(t *testing.T) {
	t.Run("second enabled consumer prevents cancellation", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		var cancelled atomic.Bool
		var connects counter

		fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
			return func(yield func(string, error) bool) {
				connects.inc()
				yield("live", nil)
				<-ctx.Done()
				cancelled.Store(true)
			}
		}

		sub := kitex.FC("Sub", func(p struct{ Enabled bool }) kitex.Node {
			r := UseStream("shared_toggle_key", fetcher, StreamOptions{
				Enabled: Some(p.Enabled),
			})
			return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
		})

		app := kitex.FC("App", func(p struct{ E1, E2 bool }) kitex.Node {
			return kitex.Box(kitex.BoxProps{},
				sub.Props(struct{ Enabled bool }{Enabled: p.E1}),
				sub.Props(struct{ Enabled bool }{Enabled: p.E2}),
			)
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app.Props(struct{ E1, E2 bool }{true, true})), container)
		if !waitFor(t, func() bool { return connects.get() == 1 }) {
			t.Fatal("expected stream to start once")
		}

		// Disable first consumer while second stays enabled
		kitex.Render(Provider(ProviderProps{Client: client}, app.Props(struct{ E1, E2 bool }{false, true})), container)
		time.Sleep(25 * time.Millisecond)
		if cancelled.Load() {
			t.Fatal("stream must NOT cancel while another enabled subscriber remains")
		}
		if connects.get() != 1 {
			t.Fatalf("expected 1 connection, got %d", connects.get())
		}

		// Disable second consumer too -> now stream cancels
		kitex.Render(Provider(ProviderProps{Client: client}, app.Props(struct{ E1, E2 bool }{false, false})), container)
		if !waitFor(t, cancelled.Load) {
			t.Fatal("expected stream to cancel when last enabled subscriber disables")
		}
	})

	t.Run("finite stream EOF deduplicates second mount", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		var connects counter
		fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
			return func(yield func(string, error) bool) {
				connects.inc()
				yield("done", nil)
			}
		}

		var res1, res2 capture[StreamResult[string]]
		sub1 := kitex.FC("Sub1", func(_ kitex.NoProps) kitex.Node {
			r := UseStream("finite_dedupe", fetcher)
			res1.set(r)
			return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
		})
		sub2 := kitex.FC("Sub2", func(_ kitex.NoProps) kitex.Node {
			r := UseStream("finite_dedupe", fetcher)
			res2.set(r)
			return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
		})

		app := kitex.FC("App", func(p struct{ ShowSecond bool }) kitex.Node {
			if p.ShowSecond {
				return kitex.Box(kitex.BoxProps{}, sub1(), sub2())
			}
			return kitex.Box(kitex.BoxProps{}, sub1())
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app.Props(struct{ ShowSecond bool }{false})), container)
		if !waitFor(t, func() bool { return res1.get().Status == "closed" && res1.get().Data == "done" }) {
			t.Fatalf("expected finite stream to complete and close, got %+v", res1.get())
		}

		// Mount second subscriber while first is still mounted & enabled
		kitex.Render(Provider(ProviderProps{Client: client}, app.Props(struct{ ShowSecond bool }{true})), container)
		time.Sleep(25 * time.Millisecond)

		if connects.get() != 1 {
			t.Fatalf("expected finite stream NOT to restart on 1->2 transition, got %d connects", connects.get())
		}
		if res2.get().Data != "done" {
			t.Fatalf("expected second subscriber to observe cached 'done', got %q", res2.get().Data)
		}
	})
}

func TestUseStreamReduce_BurstAndRefetchModes(t *testing.T) {
	t.Run("folds rapid burst without dropping throttled events", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		const total = 50
		fetcher := func(ctx context.Context, key string) iter.Seq2[int, error] {
			return func(yield func(int, error) bool) {
				for i := 1; i <= total; i++ {
					if !yield(i, nil) {
						return
					}
				}
			}
		}

		var res capture[StreamResult[int]]
		app := kitex.FC("BurstReduce", func(_ kitex.NoProps) kitex.Node {
			r := UseStreamReduce("burst_reduce", 0, fetcher, func(prev, ev int) int {
				return prev + ev
			})
			res.set(r)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

		expectedSum := total * (total + 1) / 2
		if !waitFor(t, func() bool { return res.get().Data == expectedSum && res.get().Status == "closed" }) {
			t.Fatalf("expected sum=%d and closed, got %+v", expectedSum, res.get())
		}
	})

	t.Run("RefetchReplace keeps cached data until first event", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		var connects counter
		gate := make(chan struct{})
		fetcher := func(ctx context.Context, key string) iter.Seq2[int, error] {
			return func(yield func(int, error) bool) {
				n := connects.inc()
				if n == 1 {
					yield(1, nil)
					yield(2, nil)
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-gate:
				}
				yield(10, nil)
			}
		}

		var res capture[StreamResult[int]]
		app := kitex.FC("ReplaceReduce", func(_ kitex.NoProps) kitex.Node {
			r := UseStreamReduce("replace_reduce", 0, fetcher, func(prev, ev int) int {
				return prev + ev
			}, StreamOptions{RefetchMode: RefetchReplace})
			res.set(r)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
		if !waitFor(t, func() bool { return res.get().Data == 3 && res.get().Status == "closed" }) {
			t.Fatalf("expected initial folded data 3, got %+v", res.get())
		}

		res.get().Refetch()
		if !waitFor(t, func() bool { return res.get().IsFetching }) {
			t.Fatal("expected IsFetching=true while reconnecting")
		}
		if cur := res.get(); cur.Data != 3 || cur.IsLoading {
			t.Fatalf("RefetchReplace must keep cached Data=3 and IsLoading=false while reconnecting, got %+v", cur)
		}

		close(gate)
		if !waitFor(t, func() bool { return res.get().Data == 10 && res.get().Status == "closed" }) {
			t.Fatalf("expected RefetchReplace to swap to 10, got %+v", res.get())
		}
	})

	t.Run("RefetchReset clears cached data immediately", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		var connects counter
		gate := make(chan struct{})
		fetcher := func(ctx context.Context, key string) iter.Seq2[int, error] {
			return func(yield func(int, error) bool) {
				n := connects.inc()
				if n == 1 {
					yield(5, nil)
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-gate:
				}
				yield(20, nil)
			}
		}

		var res capture[StreamResult[int]]
		app := kitex.FC("ResetReduce", func(_ kitex.NoProps) kitex.Node {
			r := UseStreamReduce("reset_reduce", 100, fetcher, func(prev, ev int) int {
				return prev + ev
			}, StreamOptions{RefetchMode: RefetchReset})
			res.set(r)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
		if !waitFor(t, func() bool { return res.get().Data == 105 && res.get().Status == "closed" }) {
			t.Fatalf("expected initial folded data 105, got %+v", res.get())
		}

		res.get().Refetch()
		if !waitFor(t, func() bool {
			cur := res.get()
			return cur.IsFetching && cur.Data == 100 && cur.IsLoading && cur.Status == "connecting"
		}) {
			t.Fatalf("RefetchReset must reset Data to initial=100, IsLoading=true, Status=connecting; got %+v", res.get())
		}

		close(gate)
		if !waitFor(t, func() bool { return res.get().Data == 120 && res.get().Status == "closed" }) {
			t.Fatalf("expected RefetchReset to fold onto initial=100 -> 120, got %+v", res.get())
		}
	})

	t.Run("RefetchAppend folds onto cached data", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		var connects counter
		fetcher := func(ctx context.Context, key string) iter.Seq2[int, error] {
			return func(yield func(int, error) bool) {
				n := connects.inc()
				if n == 1 {
					yield(1, nil)
					yield(2, nil)
					return
				}
				yield(10, nil)
				yield(20, nil)
			}
		}

		var res capture[StreamResult[int]]
		app := kitex.FC("AppendReduce", func(_ kitex.NoProps) kitex.Node {
			r := UseStreamReduce("append_reduce", 0, fetcher, func(prev, ev int) int {
				return prev + ev
			}, StreamOptions{RefetchMode: RefetchAppend})
			res.set(r)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
		if !waitFor(t, func() bool { return res.get().Data == 3 && res.get().Status == "closed" }) {
			t.Fatalf("expected initial folded data 3, got %+v", res.get())
		}

		res.get().Refetch()
		if !waitFor(t, func() bool { return res.get().Data == 33 && res.get().Status == "closed" }) {
			t.Fatalf("expected RefetchAppend to fold onto cached 3 -> 33, got %+v", res.get())
		}
	})

	t.Run("zero value transitions hasData and loading", func(t *testing.T) {
		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		client := NewClient()
		t.Cleanup(client.Close)

		fetcher := func(ctx context.Context, key string) iter.Seq2[int, error] {
			return func(yield func(int, error) bool) {
				yield(0, nil)
			}
		}

		var res capture[StreamResult[int]]
		app := kitex.FC("ZeroEvent", func(_ kitex.NoProps) kitex.Node {
			r := UseStream("zero_event", fetcher)
			res.set(r)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: client}, app()), container)
		if !waitFor(t, func() bool { return !res.get().IsLoading && res.get().Status == "closed" && res.get().Data == 0 }) {
			t.Fatalf("expected zero-value event to mark hasData=true and finish loading, got %+v", res.get())
		}
	})
}

func TestStream_BoundedWorkerPoolNonStarvation(t *testing.T) {
	sched := newBoundedPoolScheduler(4)
	promise.SetScheduler(sched)
	defer func() {
		promise.ClearSchedulerIf(sched)
		sched.Close()
	}()

	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	var openStreams atomic.Int32
	streamFetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			openStreams.Add(1)
			yield("open:"+key, nil)
			<-ctx.Done()
		}
	}

	queryFetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return "query_resolved", nil
		})
	}

	var queryRes capture[Result[string]]
	app := kitex.FC("NonStarvationApp", func(_ kitex.NoProps) kitex.Node {
		s1 := UseStream("s1", streamFetcher)
		s2 := UseStream("s2", streamFetcher)
		s3 := UseStream("s3", streamFetcher)
		s4 := UseStream("s4", streamFetcher)
		s5 := UseStream("s5", streamFetcher)
		s6 := UseStream("s6", streamFetcher)
		q := Use("non_starved_query", queryFetcher)
		queryRes.set(q)
		return kitex.Box(kitex.BoxProps{},
			kitex.Text(s1.Data+s2.Data+s3.Data+s4.Data+s5.Data+s6.Data+q.Data),
		)
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool { return openStreams.Load() == 6 }) {
		t.Fatalf("expected all 6 streams to open concurrently, got %d", openStreams.Load())
	}

	if !waitFor(t, func() bool { return queryRes.get().Data == "query_resolved" }) {
		t.Fatalf("expected wind.Use query to resolve without starving behind 6 open streams, got %+v", queryRes.get())
	}
}

func TestClient_Close(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	client.GcTime = 15 * time.Millisecond

	streamCancelled := make(chan struct{})
	fetchCancelled := make(chan struct{})

	streamFetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			yield("first", nil)
			// Second event triggers throttleTimer (default 16ms) before ctx is cancelled
			yield("second", nil)
			<-ctx.Done()
			close(streamCancelled)
		}
	}

	queryFetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.New(func(schedCtx context.Context) (string, error) {
			select {
			case <-ctx.Done():
				close(fetchCancelled)
				return "", ctx.Err()
			case <-schedCtx.Done():
				return "", schedCtx.Err()
			case <-time.After(2 * time.Second):
				return "done", nil
			}
		})
	}

	unmountQuery := kitex.FC("UnmountQuery", func(_ kitex.NoProps) kitex.Node {
		Use("gc_pending_key", func(ctx context.Context, key string) *promise.Promise[string] {
			return promise.Resolved("cached")
		})
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, unmountQuery()), container)
	time.Sleep(10 * time.Millisecond)
	// Unmount so gcTimer is armed
	kitex.Render(nil, container)

	app := kitex.FC("ActiveApp", func(_ kitex.NoProps) kitex.Node {
		UseStream("active_stream", streamFetcher)
		Use("active_fetch", queryFetcher)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	select {
	case <-time.After(time.Second):
		// wait until stream & fetch are active
	default:
		if !waitFor(t, func() bool {
			client.mu.Lock()
			_, hasStream := client.cache["active_stream"]
			_, hasFetch := client.cache["active_fetch"]
			client.mu.Unlock()
			return hasStream && hasFetch
		}) {
			t.Fatal("expected active entries in cache")
		}
	}
	time.Sleep(10 * time.Millisecond)

	// Close client (should cancel active ops, stop timers, and wait for wg)
	client.Close()
	// Idempotent second call
	client.Close()

	select {
	case <-streamCancelled:
	default:
		t.Fatal("expected Close() to cancel active stream and wait for goroutine exit")
	}

	select {
	case <-fetchCancelled:
	default:
		t.Fatal("expected Close() to cancel active fetch and wait for completion")
	}

	// Wait past GcTime & throttle interval to verify timers do not fire or panic
	time.Sleep(35 * time.Millisecond)

	client.mu.Lock()
	cacheLen := len(client.cache)
	client.mu.Unlock()
	if cacheLen != 0 {
		t.Fatalf("expected cache to be cleared after Close(), got %d entries", cacheLen)
	}

	t.Run("does not deadlock when scheduler microtasks are never flushed", func(t *testing.T) {
		sched := &testScheduler{}
		promise.SetScheduler(sched)
		defer promise.ClearSchedulerIf(sched)

		doc := dom.NewDocument()
		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
		defer kitex.Render(nil, container)

		c := NewClient()
		started := make(chan struct{})
		cancelled := make(chan struct{})

		fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
			return promise.New(func(schedCtx context.Context) (string, error) {
				close(started)
				select {
				case <-ctx.Done():
					close(cancelled)
					return "", ctx.Err()
				case <-schedCtx.Done():
					return "", schedCtx.Err()
				}
			})
		}

		app := kitex.FC("UnflushedCloseApp", func(_ kitex.NoProps) kitex.Node {
			Use("unflushed_key", fetcher)
			return kitex.Box(kitex.BoxProps{})
		})

		kitex.Render(Provider(ProviderProps{Client: c}, app()), container)

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected query fetcher to start")
		}

		closed := make(chan struct{})
		go func() {
			c.Close()
			close(closed)
		}()

		select {
		case <-closed:
		case <-time.After(time.Second):
			t.Fatal("Client.Close() deadlocked waiting on unflushed scheduler microtasks")
		}

		select {
		case <-cancelled:
		default:
			t.Fatal("expected in-flight query to be cancelled before Client.Close() returned")
		}
	})
}

func TestTearingWindow_ClosedOnSubscribe(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	// Pre-populate entry and simulate an asynchronous state bump between
	// render snapshot and UseLayoutEffectCleanup subscription.
	entry, _ := client.getOrCreateEntry("tearing_key")
	entry.mu.Lock()
	entry.state.status = "success"
	entry.state.data = "v1"
	entry.state.hasData = true
	entry.state.updatedAt = time.Now()
	entry.version = 1
	entry.mu.Unlock()

	var observed recorder[string]
	app := kitex.FC("TearingApp", func(_ kitex.NoProps) kitex.Node {
		r := Use("tearing_key", func(ctx context.Context, k string) *promise.Promise[string] {
			return promise.Resolved("v1")
		}, Options{StaleTime: time.Hour})
		observed.add(r.Data)

		// On first render only, mutate entry state before layout effects run.
		if r.Data == "v1" {
			entry.mu.Lock()
			if entry.version == 1 {
				entry.state.data = "v2"
				entry.version = 2
			}
			entry.mu.Unlock()
		}

		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	if !waitFor(t, func() bool {
		vals := observed.all()
		return len(vals) > 0 && vals[len(vals)-1] == "v2"
	}) {
		t.Fatalf("expected tearing window check to catch v2 update, observed=%v", observed.all())
	}
}

func TestStream_SchedulerStopCancelsBeforeUnmount(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{NumWorkers: 2})
	promise.SetScheduler(e.Scheduler())
	defer promise.ClearSchedulerIf(e.Scheduler())

	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	t.Cleanup(client.Close)

	started := make(chan struct{})
	cancelled := make(chan struct{})
	var exited atomic.Bool

	fetcher := func(ctx context.Context, key string) iter.Seq2[string, error] {
		return func(yield func(string, error) bool) {
			close(started)
			yield("open", nil)
			<-ctx.Done()
			time.Sleep(5 * time.Millisecond)
			exited.Store(true)
			close(cancelled)
		}
	}

	var res capture[StreamResult[string]]
	app := kitex.FC("SchedStopStream", func(_ kitex.NoProps) kitex.Node {
		r := UseStream("sched_stop_stream", fetcher)
		res.set(r)
		return kitex.Box(kitex.BoxProps{}, kitex.Text(r.Data))
	})

	kitex.Render(Provider(ProviderProps{Client: client}, app()), container)

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("expected stream to start via scheduler.Spawn")
	}

	// Drain microtasks until StreamResult reaches open state.
	if !waitFor(t, func() bool {
		e.Frame()
		return res.get().Data == "open" && res.get().Status == "open"
	}) {
		t.Fatalf("expected stream to deliver 'open', got %+v", res.get())
	}

	// Stop the engine scheduler BEFORE unmounting or calling client.Close().
	e.Stop()

	select {
	case <-cancelled:
	default:
		t.Fatal("expected e.Stop() to cancel active stream and wait for goroutine drain")
	}
	if !exited.Load() {
		t.Fatal("expected stream goroutine to finish before e.Stop() returned")
	}
}
