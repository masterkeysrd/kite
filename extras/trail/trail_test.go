package trail_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/trail"
	"github.com/masterkeysrd/kite/geom"
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

func TestTrail_Render_WaitUntilExit(t *testing.T) {
	b := mock.New(80, 24)
	App := trail.SimpleFC("App", func() kitex.Node {
		app := trail.UseApp()
		trail.UseInput(func(input string, k event.KeyEvent) {
			if input == "x" {
				app.Exit(nil)
			}
		})
		return trail.Box(trail.BoxProps{}, trail.Text("Running..."))
	})

	inst, err := trail.Render(context.Background(), App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to render trail: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	b.PostEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "x", Code: 'x'},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- inst.WaitUntilExit()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil error from WaitUntilExit, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for WaitUntilExit")
	}
}

func TestTrail_Rerender(t *testing.T) {
	b := mock.New(80, 24)
	App1 := trail.Box(trail.BoxProps{Style: style.S().Height(style.Cells(1))}, trail.Text("Hello"))
	inst, err := trail.New(App1, trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	frame1 := b.LastFrame().Surface
	if frame1.CellAt(0, 0).Content != "H" {
		t.Errorf("expected 'H', got %q", frame1.CellAt(0, 0).Content)
	}

	App2 := trail.Box(trail.BoxProps{Style: style.S().Height(style.Cells(1))}, trail.Text("World"))
	inst.Rerender(App2)
	inst.Frame()

	frame2 := b.LastFrame().Surface
	if frame2.CellAt(0, 0).Content != "W" {
		t.Errorf("expected 'W' after Rerender, got %q", frame2.CellAt(0, 0).Content)
	}
}

func TestTrail_Clear(t *testing.T) {
	b := mock.New(80, 24)
	App := trail.Box(trail.BoxProps{Style: style.S().Height(style.Cells(1))}, trail.Text("ClearMe"))
	inst, err := trail.New(App, trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	if b.LastFrame().Surface.Height != 1 {
		t.Fatalf("expected initial frame height 1, got %d", b.LastFrame().Surface.Height)
	}

	inst.Clear()
	inst.Frame()

	if b.LastFrame().Surface.Height != 0 {
		t.Errorf("expected frame height 0 after Clear, got %d", b.LastFrame().Surface.Height)
	}
}

func TestTrail_Unmount(t *testing.T) {
	b := mock.New(80, 24)
	var cleanupRan atomic.Bool

	App := trail.SimpleFC("App", func() kitex.Node {
		trail.UseEffectCleanup(func() func() {
			return func() {
				cleanupRan.Store(true)
			}
		}, nil)
		return trail.Box(trail.BoxProps{}, trail.Text("Active"))
	})

	inst, err := trail.Render(context.Background(), App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to render trail: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	inst.Unmount()

	errCh := make(chan error, 1)
	go func() {
		errCh <- inst.WaitUntilExit()
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected nil error on Unmount, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for WaitUntilExit after Unmount")
	}

	if !cleanupRan.Load() {
		t.Errorf("expected UseEffectCleanup to execute on Unmount")
	}
}

func TestTrail_UseInput_IsActive(t *testing.T) {
	b := mock.New(80, 24)
	var receivedInput string

	type AppProps struct {
		Active bool
	}

	Comp := trail.FC("Comp", func(props AppProps) kitex.Node {
		trail.UseInput(func(input string, k event.KeyEvent) {
			receivedInput = input
		}, trail.InputOptions{IsActive: props.Active})
		return trail.Box(trail.BoxProps{}, trail.Text("InputComp"))
	})

	inst, err := trail.New(Comp(AppProps{Active: false}), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	// Post key 'a' while inactive
	inst.Engine().ProcessRawEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "a", Code: 'a'},
	})

	if receivedInput != "" {
		t.Errorf("expected no input when IsActive is false, got %q", receivedInput)
	}

	// Rerender with Active = true
	inst.Rerender(Comp(AppProps{Active: true}))
	inst.Frame()

	// Post key 'b' while active
	inst.Engine().ProcessRawEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "b", Code: 'b'},
	})

	if receivedInput != "b" {
		t.Errorf("expected input 'b' when IsActive is true, got %q", receivedInput)
	}
}

func TestTrail_UseFocus_And_FocusManager(t *testing.T) {
	b := mock.New(80, 24)
	var fm trail.FocusManager

	btnRef := kitex.CreateRef[dom.Element]()
	inpRef := kitex.CreateRef[dom.Element]()

	App := trail.SimpleFC("FocusApp", func() kitex.Node {
		fm = trail.UseFocusManager()
		trail.UseFocus(btnRef, trail.FocusOptions{AutoFocus: true})
		trail.UseFocus(inpRef)

		return trail.Box(trail.BoxProps{},
			kitex.Button(kitex.ButtonProps{ID: "btn-1", Ref: btnRef}, kitex.Text("Button 1")),
			kitex.Input(kitex.InputProps{ID: "inp-1", Ref: inpRef}),
		)
	})

	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	// Verify btnRef was auto-focused
	if btnRef.Current == nil {
		t.Fatalf("expected btnRef.Current to be non-nil")
	}
	if !inst.Engine().Document().IsFocused(btnRef.Current) {
		t.Errorf("expected btnRef to be focused via AutoFocus")
	}

	// Test Focus by ID
	focused := fm.Focus("inp-1")
	if !focused {
		t.Errorf("expected fm.Focus(inp-1) to succeed")
	}
	if !inst.Engine().Document().IsFocused(inpRef.Current) {
		t.Errorf("expected inpRef to be focused")
	}

	// Test Blur
	fm.Blur()
	if inst.Engine().Document().CurrentFocus() != nil {
		t.Errorf("expected no focus after Blur, got %v", inst.Engine().Document().CurrentFocus())
	}

	// Test FocusNext
	fm.FocusNext()
	if inst.Engine().Document().CurrentFocus() == nil {
		t.Errorf("expected element to be focused after FocusNext")
	}
}

func TestTrail_MeasureElement(t *testing.T) {
	b := mock.New(80, 24)
	boxRef := kitex.CreateRef[dom.Element]()

	App := trail.SimpleFC("MeasureApp", func() kitex.Node {
		return trail.Box(trail.BoxProps{
			Ref:   boxRef,
			Style: style.S().Width(style.Cells(40)).Height(style.Cells(5)),
		}, trail.Text("Measuring box"))
	})

	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	sz := trail.MeasureElement(boxRef)
	if sz.Width != 40 || sz.Height != 5 {
		t.Errorf("expected measured size 40x5, got %dx%d", sz.Width, sz.Height)
	}

	// Test with nil ref
	nilSize := trail.MeasureElement(nil)
	if nilSize != (geom.Size{}) {
		t.Errorf("expected empty size for nil ref, got %v", nilSize)
	}
}

func TestTrail_WithInterceptLog_And_LogWriter(t *testing.T) {
	b := mock.New(80, 24)

	// Test LogWriter directly
	inst, err := trail.New(trail.Box(trail.BoxProps{}), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	lw := inst.LogWriter()
	_, _ = lw.Write([]byte("log message 1\nlog message 2\n"))

	if len(b.InsertedAbove) != 2 {
		t.Fatalf("expected 2 buffers in InsertedAbove from LogWriter, got %d", len(b.InsertedAbove))
	}

	// Test WithInterceptLog option in Run
	b2 := mock.New(80, 24)
	App2 := trail.SimpleFC("App2", func() kitex.Node {
		app := trail.UseApp()
		trail.UseInput(func(input string, k event.KeyEvent) {
			if input == "q" {
				app.Exit(nil)
			}
		})
		return trail.Box(trail.BoxProps{}, trail.Text("LoggingApp"))
	})

	inst2, err := trail.Render(context.Background(), App2(), trail.WithBackend(b2), trail.WithInterceptLog(true))
	if err != nil {
		t.Fatalf("failed to render trail: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	log.Println("intercepted message via log.Println")

	time.Sleep(20 * time.Millisecond)

	b2.PostEvent(&backend.RawKeyEvent{
		Key: key.Key{Text: "q", Code: 'q'},
	})

	_ = inst2.WaitUntilExit()

	if len(b2.InsertedAbove) < 1 {
		t.Fatalf("expected at least 1 intercepted buffer in InsertedAbove, got %d", len(b2.InsertedAbove))
	}
}

func TestTrail_UseStdout_And_UseStderr(t *testing.T) {
	b := mock.New(80, 24)
	var viewportSz geom.Size

	App := trail.SimpleFC("StreamsApp", func() kitex.Node {
		stdout := trail.UseStdout()
		stderr := trail.UseStderr()

		stdout.Write("stdout message line")
		stderr.Write("stderr message line\n")
		viewportSz = stdout.Size()

		return trail.Box(trail.BoxProps{}, trail.Text("Streams"))
	})

	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}
	inst.Frame()

	if len(b.InsertedAbove) != 2 {
		t.Fatalf("expected 2 buffers inserted above from stdout and stderr, got %d", len(b.InsertedAbove))
	}

	if viewportSz.Width != 80 || viewportSz.Height != 24 {
		t.Errorf("expected viewport size 80x24, got %v", viewportSz)
	}
}

func TestTrail_Transform(t *testing.T) {
	node := trail.Transform(func(line string, idx int) string {
		return fmt.Sprintf("[%d] %s", idx, strings.ToUpper(line))
	}, "first\nsecond")

	doc := dom.NewDocument()
	nodes := node.Instantiate(doc)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	textNode, ok := nodes[0].(dom.TextNode)
	if !ok {
		t.Fatalf("expected dom.TextNode, got %T", nodes[0])
	}

	expected := "[0] FIRST\n[1] SECOND"
	if textNode.Data() != expected {
		t.Errorf("expected %q, got %q", expected, textNode.Data())
	}
}

func TestTrail_ResumeSession_ManyStaticItemsWithActiveBottomUI(t *testing.T) {
	totalStatic := 60
	initialItems := make([]string, totalStatic)
	for i := 0; i < totalStatic; i++ {
		initialItems[i] = fmt.Sprintf("historical log %02d", i)
	}

	App := trail.SimpleFC("App", func() kitex.Node {
		items, _ := kitex.UseState(initialItems)

		return trail.Fragment(
			trail.Static(items(), func(item string, index int) kitex.Node {
				return trail.Box(trail.BoxProps{
					Style: style.S().Height(style.Cells(1)),
				}, trail.Text(item))
			}),
			trail.Box(trail.BoxProps{
				Style: style.S().Height(style.Cells(2)),
			},
				trail.Box(trail.BoxProps{
					Style: style.S().Height(style.Cells(1)),
				}, trail.Text("Composer: input message")),
				trail.Box(trail.BoxProps{
					Style: style.S().Height(style.Cells(1)),
				}, trail.Text("Status: Online")),
			),
		)
	})

	b := mock.New(80, 24)
	inst, err := trail.New(App(), trail.WithBackend(b))
	if err != nil {
		t.Fatalf("failed to create trail instance: %v", err)
	}

	inst.Frame()

	// Verify all 60 static items are flushed into InsertedAbove.
	if len(b.InsertedAbove) != totalStatic {
		t.Fatalf("expected %d static items flushed into InsertedAbove, got %d", totalStatic, len(b.InsertedAbove))
	}
	for i := 0; i < totalStatic; i++ {
		firstChar := b.InsertedAbove[i].CellAt(0, 0).Content
		if firstChar != "h" {
			t.Errorf("item %d: expected 'h' at (0, 0), got %q", i, firstChar)
		}
	}

	// Verify the active bottom UI (composer/status bar) is rendered in the inline frame.
	frame := b.LastFrame().Surface
	if frame == nil {
		t.Fatalf("expected active frame to be rendered")
	}
	if frame.Height != 2 {
		t.Errorf("expected active inline frame height 2, got %d", frame.Height)
	}
	if frame.CellAt(0, 0).Content != "C" {
		t.Errorf("expected 'C' at (0, 0) for composer, got %q", frame.CellAt(0, 0).Content)
	}
	if frame.CellAt(0, 1).Content != "S" {
		t.Errorf("expected 'S' at (0, 1) for status, got %q", frame.CellAt(0, 1).Content)
	}
}
