package kites

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type BenchState struct {
	ValueA int
	ValueB string
}

func BenchmarkStoreGet(b *testing.B) {
	store := Create(BenchState{ValueA: 42, ValueB: "hello"})

	for b.Loop() {
		_ = store.Get()
	}
}

func BenchmarkStoreSet(b *testing.B) {
	store := Create(BenchState{ValueA: 42, ValueB: "hello"})

	for i := 0; b.Loop(); i++ {
		store.Set(func(s BenchState) BenchState {
			s.ValueA = i
			return s
		})
	}
}

func BenchmarkStoreNotify(b *testing.B) {
	for _, numSubs := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("%d_subscribers", numSubs), func(b *testing.B) {
			store := Create(BenchState{ValueA: 42, ValueB: "hello"})
			var dummy int
			for range numSubs {
				store.Subscribe(func(newVal, oldVal BenchState) {
					dummy = newVal.ValueA
				})
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.Set(func(s BenchState) BenchState {
					s.ValueA = i
					return s
				})
			}
			_ = dummy
		})
	}
}

func BenchmarkUseHook(b *testing.B) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	store := Create(BenchState{ValueA: 1, ValueB: "hello"})

	comp := kitex.SimpleFC("Comp", func() kitex.Node {
		val := Use(store, func(s BenchState) int {
			return s.ValueA
		})
		return kitex.Box(kitex.BoxProps{ID: fmt.Sprintf("%d", val)})
	})

	kitex.Render(comp(), container)

	b.Run("Bailout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Updating ValueB, ValueA is unchanged. Hook should bailout and not re-render component.
			store.Set(func(s BenchState) BenchState {
				s.ValueB = fmt.Sprintf("changed_%d", i)
				return s
			})
		}
	})

	b.Run("ReRender", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Updating ValueA. Hook should NOT bailout and component should re-render.
			store.Set(func(s BenchState) BenchState {
				s.ValueA = i
				return s
			})
		}
	})
}
