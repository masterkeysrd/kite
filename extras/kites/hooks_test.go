package kites

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/testenv"
)

type TestState struct {
	CountA int
	CountB int
}

func TestUseHookIntegration(t *testing.T) {
	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
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

func TestUseHook_SliceSelectionAndCustomEquality(t *testing.T) {
	type Message struct {
		ID   string
		Text string
	}
	type ChatState struct {
		Status   string
		Messages []Message
	}

	doc := dom.NewDocument()
	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer kitex.Render(nil, container)

	store := Create(ChatState{
		Status:   "idle",
		Messages: []Message{{ID: "1", Text: "Hello"}},
	})

	var transcriptRenders int32
	var countOnlyRenders int32
	var lastRenderedText string

	transcript := kitex.SimpleFC("Transcript", func() kitex.Node {
		atomic.AddInt32(&transcriptRenders, 1)
		msgs := Use(store, func(s ChatState) []Message {
			return s.Messages
		})
		lastRenderedText = msgs[len(msgs)-1].Text
		return kitex.Box(kitex.BoxProps{}, kitex.Text(lastRenderedText))
	})

	countBadge := kitex.SimpleFC("CountBadge", func() kitex.Node {
		atomic.AddInt32(&countOnlyRenders, 1)
		msgs := UseWith(store, func(s ChatState) []Message {
			return s.Messages
		}, func(a, b []Message) bool {
			return len(a) == len(b)
		})
		return kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("%d", len(msgs))))
	})

	app := kitex.SimpleFC("App", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{}, transcript(), countBadge())
	})

	kitex.Render(app(), container)

	if atomic.LoadInt32(&transcriptRenders) != 1 || atomic.LoadInt32(&countOnlyRenders) != 1 {
		t.Fatalf("expected 1 initial render each, got transcript=%d count=%d", transcriptRenders, countOnlyRenders)
	}

	// Stream a token by mutating the last message in-place inside Update:
	// Transcript should re-render (content changed), CountBadge should bail out (len unchanged).
	store.Update(func(s *ChatState) {
		s.Messages[len(s.Messages)-1].Text += " world"
	})

	if atomic.LoadInt32(&transcriptRenders) != 2 {
		t.Fatalf("expected Transcript to re-render on in-place slice mutation, got %d", transcriptRenders)
	}
	if lastRenderedText != "Hello world" {
		t.Fatalf("expected lastRenderedText='Hello world', got %q", lastRenderedText)
	}
	if atomic.LoadInt32(&countOnlyRenders) != 1 {
		t.Fatalf("expected CountBadge to bail out when length is unchanged, got %d", countOnlyRenders)
	}

	// Append a new message: both Transcript and CountBadge should re-render.
	store.Update(func(s *ChatState) {
		s.Messages = append(s.Messages, Message{ID: "2", Text: "Done"})
	})

	if atomic.LoadInt32(&transcriptRenders) != 3 || atomic.LoadInt32(&countOnlyRenders) != 2 {
		t.Fatalf("expected transcript=3 and count=2 renders, got transcript=%d count=%d", transcriptRenders, countOnlyRenders)
	}
}

func TestUseHook_ConcurrentStreamingAndRendering(t *testing.T) {
	type Message struct {
		ID      string
		Content string
	}
	type AppState struct {
		AuthStatus string
		Messages   []Message
		Tokens     int
	}

	env := testenv.New(t, 80, 24)
	store := Create(AppState{
		AuthStatus: "checking",
		Messages:   []Message{{ID: "m1", Content: ""}},
	})

	var latestAuth string
	var latestTokens int
	var latestMsgCount int

	app := kitex.SimpleFC("ConcurrentApp", func() kitex.Node {
		state := UseStore(store)
		msgs := Use(store, func(s AppState) []Message {
			return s.Messages
		})
		latestAuth = state.AuthStatus
		latestTokens = state.Tokens
		latestMsgCount = len(msgs)

		children := make([]kitex.Node, 0, len(msgs)+1)
		children = append(children, kitex.Text(fmt.Sprintf("auth:%s tokens:%d", state.AuthStatus, state.Tokens)))
		for _, m := range msgs {
			children = append(children, kitex.Box(kitex.BoxProps{Key: m.ID}, kitex.Text(m.Content)))
		}
		return kitex.Box(kitex.BoxProps{}, children...)
	})

	root := kitex.Div(kitex.BoxProps{}).Instantiate(env.Document())[0].(dom.Element)
	env.Mount(root)
	kitex.Render(app(), root)
	env.Engine.Frame()

	const numWriters = 8
	const updatesPerWriter = 25
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Simulate background OAuth check + server event writers + token stream mutating store concurrently.
	for w := range numWriters {
		wg.Go(func() {
			for i := range updatesPerWriter {
				store.Update(func(s *AppState) {
					s.Tokens++
					if w == 0 && i == updatesPerWriter-1 {
						s.AuthStatus = "authenticated"
					}
					last := len(s.Messages) - 1
					s.Messages[last].Content += "x"
					if i%10 == 0 && len(s.Messages) < 10 {
						s.Messages = append(s.Messages, Message{
							ID:      fmt.Sprintf("m-%d-%d", w, i),
							Content: "tok",
						})
					}
				})
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	// Drive frames on the drawing thread while background goroutines mutate the store.
	for {
		select {
		case <-done:
			env.Engine.Frame()
			env.Engine.Frame()
			goto finished
		default:
			env.Engine.Frame()
		}
	}

finished:
	expected := store.Get()
	if latestTokens != expected.Tokens {
		t.Fatalf("expected rendered tokens %d to match store %d", latestTokens, expected.Tokens)
	}
	if latestAuth != "authenticated" {
		t.Fatalf("expected rendered auth 'authenticated', got %q", latestAuth)
	}
	if latestMsgCount != len(expected.Messages) {
		t.Fatalf("expected rendered msg count %d, got %d", len(expected.Messages), latestMsgCount)
	}
}

