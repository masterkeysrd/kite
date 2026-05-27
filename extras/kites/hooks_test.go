package kites

import (
	"sync/atomic"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type TestState struct {
	CountA int
	CountB int
}

func TestUseHookIntegration(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc).(dom.Element)
	defer kitex.Render(nil, container) // Clean up on exit

	store := Create(TestState{CountA: 10, CountB: 20})

	var renderCountA int32
	var renderCountB int32

	compA := kitex.SimpleFC("CompA", func() kitex.Node {
		atomic.AddInt32(&renderCountA, 1)
		val := Use(store, func(s TestState) int {
			return s.CountA
		})
		// ID must be a string. We can use a simple formatted string to represent value.
		return kitex.Box(kitex.BoxProps{ID: string(rune(val))})
	})

	compB := kitex.SimpleFC("CompB", func() kitex.Node {
		atomic.AddInt32(&renderCountB, 1)
		val := Use(store, func(s TestState) int {
			return s.CountB
		})
		return kitex.Box(kitex.BoxProps{ID: string(rune(val))})
	})

	parent := kitex.SimpleFC("Parent", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{}, compA(), compB())
	})

	// 1. Initial Render
	kitex.Render(parent(), container)

	if atomic.LoadInt32(&renderCountA) != 1 {
		t.Errorf("expected CompA to render 1 time initially, got %d", renderCountA)
	}
	if atomic.LoadInt32(&renderCountB) != 1 {
		t.Errorf("expected CompB to render 1 time initially, got %d", renderCountB)
	}

	// 2. Update CountA: only CompA should re-render
	store.Set(func(s TestState) TestState {
		s.CountA = 11
		return s
	})

	if atomic.LoadInt32(&renderCountA) != 2 {
		t.Errorf("expected CompA to re-render after updating CountA, got %d", renderCountA)
	}
	if atomic.LoadInt32(&renderCountB) != 1 {
		t.Errorf("expected CompB not to re-render, got %d", renderCountB)
	}

	// 3. Update CountB: only CompB should re-render
	store.Set(func(s TestState) TestState {
		s.CountB = 25
		return s
	})

	if atomic.LoadInt32(&renderCountA) != 2 {
		t.Errorf("expected CompA not to re-render, got %d", renderCountA)
	}
	if atomic.LoadInt32(&renderCountB) != 2 {
		t.Errorf("expected CompB to re-render after updating CountB, got %d", renderCountB)
	}

	// 4. Update unrelated (or update with same values): neither should re-render
	store.Set(func(s TestState) TestState {
		return s
	})

	if atomic.LoadInt32(&renderCountA) != 2 {
		t.Errorf("expected CompA not to re-render when state is unchanged, got %d", renderCountA)
	}
	if atomic.LoadInt32(&renderCountB) != 2 {
		t.Errorf("expected CompB not to re-render when state is unchanged, got %d", renderCountB)
	}
}
