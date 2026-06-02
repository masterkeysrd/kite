package testenv_test

import (
	"testing"
	"time"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestEnvironment_Type(t *testing.T) {
	// 1. Setup default environment.
	env := testenv.Default(80, 24)
	defer env.Close()

	// 2. Mount an input field declaratively.
	env.Mount(element.Input("").WithID("my-input"))
	input := env.GetNodeByID("my-input").(*element.InputElement)

	// 3. Simulation.
	env.Flush() // Initial render and auto-focus
	env.Type("hello")
	env.Flush() // Flush after typing

	// 4. Assertions.
	got := input.Value()
	if got != "hello" {
		t.Errorf("expected input value 'hello', got %q", got)
	}
}

func TestEnvironment_QuerySelector(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	env.Mount(element.Box().WithID("target").WithClass("my-class"))
	env.Flush()

	// Test ID selector
	found := env.QuerySelector("#target")
	if found == nil || found.ID() != "target" {
		t.Errorf("expected to find element with ID 'target'")
	}

	// Test Tag selector
	found = env.QuerySelector("box")
	if found == nil || found.TagName() != "box" {
		t.Errorf("expected to find element with tag 'box', got %v", found)
	}

	// Test Class selector
	found = env.QuerySelector(".my-class")
	if found == nil {
		t.Errorf("expected to find element with class 'my-class'")
	}
}

func TestEnvironment_Wheel(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	box := element.Box().WithID("scroller")
	// Make it scrollable by giving it fixed size and content that exceeds it
	box.Style(style.Style{
		Width:     style.Some(style.Cells(5)),
		Height:    style.Some(style.Cells(5)),
		OverflowY: style.Some(style.OverflowScroll),
	})
	// Add a tall child to ensure there is something to scroll
	box.AddChild(element.Box().Style(style.Style{Height: style.Some(style.Cells(20))}))

	env.Mount(box)
	env.Flush()

	// Simulate wheel at (0,0) which should hit the box
	env.Wheel(0, 0, 0, 2)
	env.Flush()

	// Verify scroll offset
	_, gotY := box.Scroll()
	if gotY == 0 {
		t.Errorf("expected Y scroll offset > 0, got 0")
	}
	if gotY != 2 {
		t.Errorf("expected Y scroll offset 2, got %d", gotY)
	}
}

func TestNewAssertions(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	box := element.Box().WithID("my-box").WithClass("active")
	box.AddChild(element.Text("Hello World"))

	checkbox := element.Checkbox(true).WithName("check-me").Disabled(true)

	env.Mount(box)
	env.Mount(checkbox)
	env.Flush()

	// Use assertions
	testenv.Expect(t, box).
		ToHaveID("my-box").
		ToHaveClass("active").
		ToHaveTextContent("Hello World")

	testenv.Expect(t, checkbox).
		ToBeChecked(true).
		ToBeDisabled(true).
		ToHaveValue(true)
}

func TestMoreNewFeatures(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	box := element.Box().WithID("my-box").Style(style.Style{
		Width:  style.Some(style.Cells(5)),
		Height: style.Some(style.Cells(5)),
	})
	input := element.Input("").WithID("input-field")

	container := element.Box()
	container.AddChild(box)
	container.AddChild(input)

	env.Mount(container)
	env.Flush()

	// 1. Test Play (Option A)
	input.Focus()
	env.Play("hello", "<Tab>", "<Shift+Tab>", " world")
	env.Flush()

	testenv.Expect(t, input).ToHaveValue("hello world")

	// 2. Test Eventually
	counter := 0
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(5 * time.Millisecond)
			counter++
		}
	}()

	testenv.Eventually(t, func() bool {
		return counter == 3
	}, 100*time.Millisecond)

	// 3. Test EventSpy / SpyEvents
	spy := testenv.SpyEvents(t, box, "click")
	env.Click(0, 0)
	spy.AssertFired()
	spy.AssertFiredCount(1)

	// 4. DoubleClick
	spy2 := testenv.SpyEvents(t, box, "click")
	env.DoubleClick(0, 0)
	spy2.AssertFiredCount(2)
}
