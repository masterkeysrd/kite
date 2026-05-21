---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: kite-testing
  description: Guidelines and examples for using the devtools/testenv package for ergonomic, headless component testing.
  displayName: Kite Testing
---

# Kite Testing Skill

This skill provides comprehensive guidelines and patterns for testing Kite components using the `devtools/testenv` package. The test environment allows for high-level, ergonomic testing of UI components by simulating user input and inspecting the logical DOM without requiring a physical terminal.

## Core Concepts

*   **`testenv.Environment`**: A wrapper around the Kite engine and a mock backend that provides a high-level API for test simulation.
*   **Headless Execution**: Tests run entirely in memory using `mock.Backend`, making them fast and suitable for CI.
*   **Declarative Setup**: Components can be declared and mounted in a single line using the fluent `element` API.
*   **DOM Piercing**: Query APIs (`QuerySelector`, `GetNodeByID`) automatically traverse into UA shadow subtrees to allow testing of internal widget state.
*   **Golden Testing**: Visual regression testing by comparing current framebuffer state against stored `.golden` snapshots.
*   **Visual Dumps**: Diagnostic tools to export the current framebuffer as ANSI text, plain text, or HTML for debugging.

## Implementation Guidelines

### 1. Environment Setup
Always use `testenv.Default(width, height)` for standard tests. This initializes the mock backend and engine automatically. Use `defer env.Close()` to ensure proper cleanup.

```go
env := testenv.Default(80, 24)
defer env.Close()
```

### 2. Declarative Mounting
Use the declarative `element` API to build and mount your UI tree. Use `WithID()` to make elements easily accessible for assertions.

```go
env.Mount(element.Box(
    element.Input("initial value").WithID("my-input"),
    element.Button("Submit").WithID("btn"),
))
```

### 3. Simulating Interaction
The environment provides simple methods to simulate user behavior:
*   **`env.Type(text)`**: Simulates a user typing characters. Ensure the target element is focused.
*   **`env.Click(x, y)`**: Simulates a mouse click at specific coordinates.
*   **`env.Wheel(x, y, dx, dy)`**: Simulates a mouse wheel event at specific coordinates.
*   **`env.ScrollTo(el, x, y)` / `env.ScrollBy(el, dx, dy)`**: Directly manipulate element scroll offsets.
*   **`env.SendKey(key)`**: Sends a specific `key.Key` event.

### 4. Frame Synchronization
Use `env.Flush()` to block until the engine processes pending tasks, updates styles, runs layout, and "paints" a frame. Call `Flush()`:
*   After mounting (to establish initial state and auto-focus).
*   After every interaction (to ensure the effects are processed).

### 5. Assertions
Retrieve elements via `env.GetNodeByID(id)` or `env.QuerySelector(selector)` to assert on their logical state.

```go
input := env.GetNodeByID("my-input").(*element.InputElement)
if input.Value() != "new value" {
    t.Errorf("unexpected value: %s", input.Value())
}
```

### 6. Golden Testing
Use `env.MatchGolden(t, "filename")` to perform visual regression testing.
*   If the `.golden` file doesn't exist, it will be created.
*   Use `go test -update` to refresh existing golden files.
*   Golden files are stored in `testdata/*.golden`.
*   On failure, the actual state is saved to `testdata/*.actual`.

```go
env.MatchGolden(t, "my-component-state")
```

### 7. Visual Dumps
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
        // Note: in a real app, you'd update the button text here
    })

    env.Mount(btn)
    env.Flush() // Initial sync

    // Buttons are usually hit-tested via click coordinates
    // For this example, we assume we know the button is at 0,0
    env.Click(0, 0)
    env.Flush()

    if count != 1 {
        t.Errorf("expected count 1, got %d", count)
    }
}
```

## Best Practices
*   **Use IDs**: Prefer `GetNodeByID` for reliable element retrieval in tests.
*   **Check Focus**: If `Type()` isn't working, verify the correct element is focused using `env.Engine.FocusManager().Current()`.
*   **No UAT Piercing**: `QuerySelector` and `GetNodeByID` only see public DOM nodes. They do not cross UA shadow boundaries. Interact with components via their public API (e.g., `input.Value()`) or by simulating raw input (clicking, typing).
*   **Mock Backend Inspection**: If you need to verify visual output (cells, colors), access the backend via `env.Backend.LastFrame()`.
