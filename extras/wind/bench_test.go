package wind

import (
	"context"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/promise"
)

func BenchmarkOptionalUnwrapOr(b *testing.B) {
	b.ReportAllocs()
	unsetBool := None[bool]()
	setBool := Some(false)
	unsetDur := None[time.Duration]()
	setDur := Some(time.Duration(0))

	var boolSink bool
	var durSink time.Duration
	for b.Loop() {
		boolSink = unsetBool.UnwrapOr(true) && !setBool.UnwrapOr(true)
		durSink = unsetDur.UnwrapOr(16*time.Millisecond) + setDur.UnwrapOr(16*time.Millisecond)
	}
	_ = boolSink
	_ = durSink
}

func BenchmarkUse_CacheHit(b *testing.B) {
	b.ReportAllocs()
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	defer client.Close()

	entry, _ := client.getOrCreateEntry("warm_key")
	entry.mu.Lock()
	entry.state.status = "success"
	entry.state.data = "cached_value"
	entry.state.hasData = true
	entry.state.updatedAt = time.Now()
	entry.version = 1
	entry.mu.Unlock()

	fetcher := func(ctx context.Context, key string) *promise.Promise[string] {
		return promise.Resolved("cached_value")
	}

	comp := kitex.FC("WarmQueryComp", func(_ kitex.NoProps) kitex.Node {
		res := Use("warm_key", fetcher, Options{StaleTime: time.Hour})
		return kitex.Box(kitex.BoxProps{ID: res.Data})
	})
	root := Provider(ProviderProps{Client: client}, comp())

	kitex.Render(root, container)

	for b.Loop() {
		kitex.Render(root, container)
	}
}

type benchBurstState struct {
	count int
	sum   int
}

func BenchmarkUseStream_BurstFold(b *testing.B) {
	b.ReportAllocs()
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	client := NewClient()
	defer client.Close()

	const burstSize = 64
	var mu sync.Mutex
	var doneCh chan struct{}

	source := func(ctx context.Context, _ benchBurstState) iter.Seq2[int, error] {
		return func(yield func(int, error) bool) {
			for i := 1; i <= burstSize; i++ {
				if !yield(i, nil) {
					return
				}
			}
			mu.Lock()
			ch := doneCh
			mu.Unlock()
			if ch != nil {
				close(ch)
			}
		}
	}

	var latest capture[StreamResult[benchBurstState]]
	comp := kitex.FC("BurstFoldComp", func(_ kitex.NoProps) kitex.Node {
		res := UseStream("bench_burst_reduce", Stream[int, benchBurstState]{
			Source: source,
			Reduce: func(prev benchBurstState, ev int) benchBurstState {
				return benchBurstState{
					count: prev.count + 1,
					sum:   prev.sum + ev,
				}
			},
			Enabled:  Some(false),
			Throttle: Some(time.Duration(0)),
		})
		latest.set(res)
		return kitex.Box(kitex.BoxProps{})
	})

	kitex.Render(Provider(ProviderProps{Client: client}, comp()), container)
	refetch := latest.get().Refetch

	for b.Loop() {
		ch := make(chan struct{})
		mu.Lock()
		doneCh = ch
		mu.Unlock()

		refetch()
		<-ch
	}
}
