package testenv_test

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
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
