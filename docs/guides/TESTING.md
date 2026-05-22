# Testing Guide

Run tests across the repo with a timeout to prevent hangs:

```bash
go test -timeout 30s ./...
```

Run a single package's tests with verbose output:

```bash
go test -v -timeout 30s ./layout
```

Run benchmarks (example for `layout`):

```bash
go test -bench . -run ^$ -benchmem ./layout
```

### Headless Integration Testing

For high-level component testing, use the `devtools/testenv` package. It allows you to simulate user interactions and verify DOM state without a terminal.

```go
func TestComponent(t *testing.T) {
    env := testenv.Default(80, 24)
    defer env.Close()

    // Declarative setup
    env.Mount(element.Input("initial").WithID("my-input"))
    env.Flush()

    // Simulation
    env.Type("hello")
    env.Flush()

    // Assertion
    input := env.GetNodeByID("my-input").(*element.InputElement)
    if input.Value() != "hello" {
        t.Errorf("unexpected value")
    }
}
```

### Golden Testing

Catch visual regressions by snapshotting the framebuffer state.

```go
func TestVisual(t *testing.T) {
    env := testenv.Default(20, 5)
    defer env.Close()

    env.Mount(element.Box().Style(style.Style{
        Background: style.Some(color.Color(color.RGBA{R: 255, G: 0, B: 0, A: 255})),
    }))
    env.Flush()

    // Compares against testdata/my-box.golden
    // Use 'go test -update' to refresh snapshots
    env.MatchGolden(t, "my-box")
}
```

### Visual Dumps

For debugging complex layouts or CI failures, you can dump the current state in various formats:

```go
// Print to your local terminal with full colors
fmt.Print(env.DumpANSI())

// Save as HTML to view in a browser
os.WriteFile("diff.html", []byte(env.DumpHTML()), 0644)

// Plain text for quick diffing
fmt.Print(env.DumpText())
```

### Terminal X-Ray Mode

When running interactive tests or manual verification, you can toggle the visual layout debugging overlay:

1. Call `devtools.Install(eng, devtools.Options{...})` in your setup.
2. Press the configured hotkey (default `Ctrl+D`) while the application is running to overlay the Margin (Red), Padding (Green), and Content (Blue) boxes.

See the `kite-testing` skill for more detailed guidelines.

### Assertion helpers (devtools/testenv)

Two new helper families make headless assertions ergonomic:

- `Expect(t, node)` — fluent assertions on logical DOM nodes. Located in `devtools/testenv/assertions.go`.
    - Example:

```go
env := testenv.Default(10,5)
defer env.Close()

tree := element.Box(
        element.Box("a"),
        element.Box("b"),
        element.Box("c"),
)
env.Mount(tree)
env.Flush()

testenv.Expect(t, tree).
    ToHaveChildCount(3).
    ToHaveChildrenText([]string{"a", "b", "c"})
```

- `ExpectScreen(t, env)` — fluent assertions on the rendered framebuffer. Located in `devtools/testenv/screen_assertions.go`.
    - Example:

```go
env := testenv.Default(20,10)
defer env.Close()

// ... mount and render ...
env.Flush()

testenv.ExpectScreen(t, env).
        CellAt(1,3).ToHaveContent("├").
        CellAt(10,3).ToHaveContent("┤")
```

Both helpers use `t.Helper()` and call `t.Fatalf`/`t.Errorf` as appropriate.
