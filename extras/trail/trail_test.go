package trail_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/trail"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

func TestTrail_New_DynamicFrames(t *testing.T) {
	var updateCount func(int)

	App := trail.SimpleFC("App", func() kitex.Node {
		count, setCount := kitex.UseState(2)
		updateCount = setCount

		c := count()
		boxes := make([]kitex.Node, c)
		for i := 0; i < c; i++ {
			boxes[i] = trail.Box(trail.BoxProps{
				Style: style.S().Height(style.Cells(1)),
			}, trail.Text("Line"))
		}
		return trail.Box(trail.BoxProps{}, boxes...)
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	inst.Frame()

	frame1 := b.LastFrame().Surface
	if frame1 == nil {
		t.Fatalf("expected a rendered frame")
	}
	if frame1.Height != 2 {
		t.Errorf("expected 2-row initial frame, got %d", frame1.Height)
	}

	// Dynamically grow to 4 rows
	updateCount(4)
	inst.Frame()

	frame2 := b.LastFrame().Surface
	if frame2.Height != 4 {
		t.Errorf("expected 4-row frame after growing, got %d", frame2.Height)
	}

	// Dynamically shrink to 1 row
	updateCount(1)
	inst.Frame()

	frame3 := b.LastFrame().Surface
	if frame3.Height != 1 {
		t.Errorf("expected 1-row frame after shrinking, got %d", frame3.Height)
	}
}

func TestTrail_Static(t *testing.T) {
	var addItems func(...string)
	var updateProgress func(string)

	App := trail.SimpleFC("App", func() kitex.Node {
		items, setItems := kitex.UseState([]string{"task 1", "task 2"})
		addItems = func(newItems ...string) {
			prev := items()
			cp := make([]string, len(prev)+len(newItems))
			copy(cp, prev)
			copy(cp[len(prev):], newItems)
			setItems(cp)
		}
		progress, setProgress := kitex.UseState("Progress: 50%")
		updateProgress = func(p string) {
			setProgress(p)
		}

		return trail.Fragment(
			trail.Static(items(), func(item string, index int) kitex.Node {
				return trail.Box(trail.BoxProps{
					Style: style.S().Height(style.Cells(1)),
				}, trail.Text(item))
			}),
			trail.Box(trail.BoxProps{
				Style: style.S().Height(style.Cells(1)),
			}, trail.Text(progress())),
		)
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	inst.Frame()

	// Static initial items (2) must have been flushed to InsertedAbove
	if len(b.InsertedAbove) != 2 {
		t.Fatalf("expected 2 items flushed to InsertedAbove, got %d", len(b.InsertedAbove))
	}
	if b.InsertedAbove[0].Height != 1 {
		t.Errorf("expected first inserted buffer height 1, got %d", b.InsertedAbove[0].Height)
	}
	if b.InsertedAbove[0].CellAt(0, 0).Content != "t" {
		t.Errorf("expected 't' in first inserted buffer, got %q", b.InsertedAbove[0].CellAt(0, 0).Content)
	}

	// Live inline frame only contains the progress box (height 1)
	frame1 := b.LastFrame().Surface
	if frame1.Height != 1 {
		t.Errorf("expected live frame height 1, got %d", frame1.Height)
	}
	if frame1.CellAt(0, 0).Content != "P" {
		t.Errorf("expected live frame to show 'Progress...', got %q", frame1.CellAt(0, 0).Content)
	}

	// Append a new item "task 3"
	addItems("task 3")
	updateProgress("Progress: 75%")
	inst.Frame()

	// Only task 3 should have been appended (total 3 in InsertedAbove)
	if len(b.InsertedAbove) != 3 {
		t.Fatalf("expected 3 items flushed to InsertedAbove after adding task 3, got %d", len(b.InsertedAbove))
	}
	if b.InsertedAbove[2].CellAt(0, 0).Content != "t" {
		t.Errorf("expected 't' in third inserted buffer, got %q", b.InsertedAbove[2].CellAt(0, 0).Content)
	}

	// Live frame still occupies only its own height (1 row)
	frame2 := b.LastFrame().Surface
	if frame2.Height != 1 {
		t.Errorf("expected live frame height to remain 1, got %d", frame2.Height)
	}

	// Update progress only (no new static items)
	updateProgress("Progress: 100%")
	inst.Frame()

	// No additional items flushed
	if len(b.InsertedAbove) != 3 {
		t.Fatalf("expected InsertedAbove count to remain 3, got %d", len(b.InsertedAbove))
	}
	frame3 := b.LastFrame().Surface
	if frame3.Height != 1 {
		t.Errorf("expected live frame height to remain 1, got %d", frame3.Height)
	}
}

func TestTrail_UseApp_Exit(t *testing.T) {
	expectedErr := errors.New("custom exit error")

	App := trail.SimpleFC("App", func() kitex.Node {
		app := trail.UseApp()
		trail.UseInput(func(input string, k event.KeyEvent) {
			if input == "x" {
				app.Exit(expectedErr)
			}
		})
		return trail.Box(trail.BoxProps{}, trail.Text("running"))
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- inst.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	b.PostEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "x", Code: 'x'},
	})

	select {
	case runErr := <-errCh:
		if !errors.Is(runErr, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for app to exit")
	}
}

func TestTrail_Spacer_And_Newline(t *testing.T) {
	doc := dom.NewDocument()

	// Newline default (1)
	nl1 := trail.Newline()
	nodes1 := nl1.Instantiate(doc)
	if len(nodes1) != 1 {
		t.Fatalf("expected 1 DOM node for Newline(), got %d", len(nodes1))
	}

	// Newline with count (3)
	nl3 := trail.Newline(3)
	nodes3 := nl3.Instantiate(doc)
	if len(nodes3) != 1 {
		t.Fatalf("expected 1 DOM node for Newline(3), got %d", len(nodes3))
	}

	// Spacer
	sp := trail.Spacer()
	nodesSp := sp.Instantiate(doc)
	if len(nodesSp) != 1 {
		t.Fatalf("expected 1 DOM node for Spacer(), got %d", len(nodesSp))
	}
}

func TestTrail_WithCtrlCExit(t *testing.T) {
	App := trail.SimpleFC("App", func() kitex.Node {
		return trail.Box(trail.BoxProps{}, trail.Text("app"))
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b), trail.WithCtrlCExit(true))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- inst.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	// Send Ctrl+C
	b.PostEvent(&backend.RawKeyEvent{
		Key: key.Key{Code: 'c', Mod: key.ModCtrl},
	})

	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Errorf("expected clean exit on Ctrl+C, got error %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for app to exit on Ctrl+C")
	}
}

func TestTrail_UseInput_And_Println(t *testing.T) {
	var receivedInput string
	var receivedKey event.KeyEvent

	App := trail.SimpleFC("App", func() kitex.Node {
		app := trail.UseApp()
		trail.UseInput(func(input string, k event.KeyEvent) {
			receivedInput = input
			receivedKey = k
			if input == "p" {
				app.Println("printed log line")
			}
		})
		return trail.Box(trail.BoxProps{}, trail.Text("interactive"))
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	inst.Frame()

	// Send key 'p'
	inst.Engine().ProcessRawEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "p", Code: 'p'},
	})

	if receivedInput != "p" {
		t.Errorf("expected received input 'p', got %q", receivedInput)
	}
	if receivedKey.Code != 'p' {
		t.Errorf("expected key code 'p', got %v", receivedKey.Code)
	}

	// Verify Println wrote to InsertedAbove
	if len(b.InsertedAbove) != 1 {
		t.Fatalf("expected 1 buffer inserted above from Println, got %d", len(b.InsertedAbove))
	}
	if b.InsertedAbove[0].CellAt(0, 0).Content != "p" {
		t.Errorf("expected 'p' in inserted buffer, got %q", b.InsertedAbove[0].CellAt(0, 0).Content)
	}
}

func TestTrail_Exit_FlushesFinalStateAndStatic(t *testing.T) {
	App := trail.SimpleFC("App", func() kitex.Node {
		app := trail.UseApp()
		status, setStatus := kitex.UseState("in progress")
		items, setItems := kitex.UseState([]string{"step 1"})

		trail.UseInput(func(input string, k event.KeyEvent) {
			if input == "done" {
				setItems([]string{"step 1", "step 2"})
				setStatus("complete")
				app.Exit(nil)
			}
		})

		return trail.Fragment(
			trail.Static(items(), func(item string, index int) kitex.Node {
				return trail.Box(trail.BoxProps{
					Style: style.S().Height(style.Cells(1)),
				}, trail.Text(item))
			}),
			trail.Box(trail.BoxProps{
				Style: style.S().Height(style.Cells(1)),
			}, trail.Text(status())),
		)
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- inst.Run(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	// Send key 'done' (matching input "done")
	b.PostEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "done"},
	})

	select {
	case runErr := <-errCh:
		if runErr != nil {
			t.Fatalf("expected nil error on exit, got %v", runErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to exit")
	}

	// Verify step 2 was flushed to InsertedAbove
	if len(b.InsertedAbove) != 2 {
		t.Fatalf("expected 2 items flushed to InsertedAbove, got %d", len(b.InsertedAbove))
	}
	if b.InsertedAbove[1].CellAt(5, 0).Content != "2" {
		t.Errorf("expected '2' in second inserted buffer, got %q", b.InsertedAbove[1].CellAt(5, 0).Content)
	}

	// Verify final live frame has "complete"
	frame := b.LastFrame().Surface
	if frame == nil {
		t.Fatalf("expected last frame to exist")
	}
	if frame.CellAt(0, 0).Content != "c" {
		t.Errorf("expected final frame to display 'complete', got %q", frame.CellAt(0, 0).Content)
	}
}
