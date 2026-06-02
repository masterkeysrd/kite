---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: kite-testing
  description: Guidelines and examples for using the testenv package for ergonomic, headless component testing.
  displayName: Kite Testing
---

# Kite Testing Skill

This skill provides comprehensive guidelines and patterns for testing Kite components using the `testenv` package. The test environment allows for high-level, ergonomic testing of UI components by simulating user input and inspecting the logical DOM without requiring a physical terminal.

## Core Concepts

*   **`testenv.Environment`**: A wrapper around the Kite engine and a mock backend that provides a high-level API for test simulation.
*   **Headless Execution**: Tests run entirely in memory using `mock.Backend`, making them fast and suitable for CI.
*   **Declarative Setup**: Components can be declared and mounted in a single line using the fluent `element` API.
*   **DOM Piercing**: Query APIs (`QuerySelector`, `GetNodeByID`) automatically traverse into UA shadow subtrees to allow testing of internal widget state.
*   **Golden Testing**: Visual regression testing by comparing current framebuffer state against stored `.golden` snapshots. Shows color-supported side-by-side terminal diffs on failure.
*   **Visual Dumps**: Diagnostic tools to export the current framebuffer as ANSI text, plain text, or HTML for debugging.

## Implementation Guidelines

### 1. Environment Setup
Always use `testenv.Default(width, height)` for standard tests. This initializes the mock backend and engine automatically. Use `defer env.Close()` to ensure proper cleanup.

```go
env := testenv.Default(80, 24)
defer env.Close()
```

### 2. Declarative Mounting
Use the declarative `element` API to build and mount your UI tree. Use `WithID()` to make elements easily accessible for assertions. Note that `env.Mount()` replaces the document root element, so multiple elements should be wrapped in a container box.

```go
container := element.Box()
container.AddChild(element.Input("initial value").WithID("my-input"))
container.AddChild(element.Button("Submit").WithID("btn"))
env.Mount(container)
```

### 3. Simulating Interaction
The environment provides simple methods to simulate user behavior:
*   **`env.Play(actions...)`**: Simulates key presses. Special keys are enclosed in brackets (e.g. `"<Tab>"`, `"<Enter>"`, `"<Esc>"`, `"<Ctrl+c>"`), while other strings are typed literally.
*   **`env.Click(x, y)`**: Simulates a mouse click.
*   **`env.DoubleClick(x, y)`**: Simulates a mouse double-click.
*   **`env.DragAndDrop(x1, y1, x2, y2)`**: Simulates dragging a mouse from `(x1, y1)` to `(x2, y2)`.
*   **`env.Wheel(x, y, dx, dy)`**: Simulates a mouse wheel event.
*   **`env.ScrollTo(el, x, y)` / `env.ScrollBy(el, dx, dy)`**: Directly manipulate element scroll offsets.

### 4. Frame Synchronization
Use `env.Flush()` to block until the engine processes pending tasks, updates styles, runs layout, and "paints" a frame. Call `Flush()`:
*   After mounting (to establish initial state and auto-focus).
*   After every interaction (to ensure the effects are processed).

### 5. Fluent DOM State Assertions
Use `testenv.Expect(t, node)` for fluent, chainable DOM assertions.
*   `ToHaveID(expected)`: Asserts element ID.
*   `ToHaveClass(expected)`: Asserts element class name.
*   `ToHaveTextContent(expected)`: Asserts node subtree text content.
*   `ToHaveValue(expected)`: Asserts form control value.
*   `ToBeChecked(expected)`: Asserts checkbox or radio checked status.
*   `ToBeDisabled(expected)`: Asserts element disabled status.

```go
testenv.Expect(t, env.GetNodeByID("my-input")).
    ToHaveID("my-input").
    ToHaveValue("initial value").
    ToBeDisabled(false)
```

### 6. Event Spying & Async Conditions
*   **`testenv.Eventually(t, checkFunc, timeout)`**: Periodically polls a condition until it returns true or the timeout expires.
*   **`testenv.SpyEvents(t, target, eventType)`**: Attaches an `EventSpy` to record occurrences of events (e.g., `spy.AssertFired()`, `spy.AssertFiredCount(2)`).

```go
spy := testenv.SpyEvents(t, btn, event.EventClick)
env.Click(0, 0)
spy.AssertFired()
```

### 7. Golden Testing
Use `env.MatchGolden(t, "filename")` to perform visual regression testing.
*   If the `.golden` file doesn't exist, it will be created.
*   Use `go test -update` to refresh existing golden files.
*   Golden files are stored in `testdata/*.golden`. On failure, the actual state is saved to `testdata/*.actual` and a side-by-side terminal diff is printed.

```go
env.MatchGolden(t, "my-component-state")
```

### 8. Visual Dumps
When debugging CI failures or complex layouts, use dump utilities to inspect the exact state of the framebuffer:
*   **`env.DumpANSI()`**: Returns a string with ANSI escape codes for local terminal inspection.
*   **`env.DumpHTML()`**: Returns a standalone HTML file with CSS styles (useful for CI artifacts).
*   **`env.DumpText()`**: Returns a plain text grid (useful for quick diffs).

```go
fmt.Println(env.DumpANSI())
os.WriteFile("dump.html", []byte(env.DumpHTML()), 0644)
```

## Example Test Case

```go
func TestButtonCounter(t *testing.T) {
    env := testenv.Default(80, 24)
    defer env.Close()

    count := 0
    btn := element.Button("Count: 0").WithID("btn").OnEvent(event.EventClick, func(e event.Event) {
        count++
    })

    env.Mount(btn)
    env.Flush() // Initial sync

    spy := testenv.SpyEvents(t, btn, event.EventClick)
    env.Click(0, 0)
    env.Flush()

    spy.AssertFiredCount(1)
    if count != 1 {
        t.Errorf("expected count 1, got %d", count)
    }
}
```

## Best Practices
*   **Use IDs**: Prefer `GetNodeByID` for reliable element retrieval in tests.
*   **Check Focus**: If typing simulation isn't working, verify the correct element is focused.
*   **Wrap Mounted Children**: Remember that `Mount()` replaces the document root; always wrap multiple elements in a single container Box.

