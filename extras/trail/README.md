# trail

`trail` is an Ink-like inline CLI framework built on [Kite](https://github.com/masterkeysrd/kite), its rendering `engine`, and the `kitex` reactive Virtual DOM framework.

<p align="center">
  <img width="960" height="540" alt="trail_demo" src="https://github.com/user-attachments/assets/2adc3681-c760-4707-84b4-896a47b298fe" />
</p>

## Features

- **Inline Execution**: Runs directly in the standard terminal scrollback without taking over the screen or entering an alternate screen buffer (`altscreen`).
- **Dynamic Content Height**: Automatically resizes the active terminal viewport to exactly match the rendered content height.
- **`<Static>` Output**: Permanently write completed logs, tasks, and items above the active inline frame into the terminal's native scrollback buffer while continuing to update live interactive components below.
- **Ink-Compatible Mental Model**: Declarative component tree with lifecycle controls (`Render`, `WaitUntilExit`, `Rerender`, `Unmount`, `Clear`), hooks (`UseApp`, `UseInput`, `UseFocus`, `UseFocusManager`, `UseStdout`, `UseStderr`), and primitives (`Box`, `Text`, `Span`, `Static`, `Newline`, `Spacer`, `Transform`).
- **Console & Log Interception**: Automatically redirect standard `log.Default()` or `slog` output (`WithInterceptLog`, `inst.LogWriter()`) cleanly into scrollback above the active frame.
- **Full kitex Interoperability**: Direct access to `kitex` state hooks (`UseState`, `UseReducer`), effect hooks (`UseEffect`, `UseEffectCleanup`, `UseLayoutEffect`, `UseInterval`, `UseTimeout`), and refs (`UseRef`, `CreateRef`).

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/trail"
)

func App() trail.Node {
	count, setCount := trail.UseState(0)
	app := trail.UseApp()

	trail.UseInput(func(input string, k event.KeyEvent) {
		if input == "q" {
			app.Exit(nil)
		}
	})

	trail.UseInterval(func() {
		setCount(count() + 1)
	}, time.Second, nil)

	return trail.Box(trail.BoxProps{},
		trail.Text(fmt.Sprintf("Counter: %d (press 'q' to exit)", count())),
	)
}

func main() {
	// Start with trail.Render and block on WaitUntilExit (Ink-style)
	inst, err := trail.Render(context.Background(), App())
	if err != nil {
		panic(err)
	}
	if err := inst.WaitUntilExit(); err != nil {
		panic(err)
	}
}
```

## Static Output

Use `trail.Static` to permanently insert completed items above the active live UI:

```go
func TaskList(tasks []string) trail.Node {
	return trail.Fragment(
		trail.Static(tasks, func(task string, index int) trail.Node {
			return trail.Box(trail.BoxProps{},
				trail.Text("✔ " + task),
			)
		}),
		trail.Box(trail.BoxProps{},
			trail.Text("Working on next task..."),
		),
	)
}
```

## Instance Lifecycle Controls

`trail.Render` returns an `*Instance` that gives you full programmatic control over the inline execution lifecycle:

- **`inst.WaitUntilExit() error`**: Blocks until the app exits and returns the recorded exit error.
- **`inst.Rerender(newApp Node)`**: Dynamically updates the root component tree with a new node.
- **`inst.Clear()` / `app.Clear()`**: Clears the current inline terminal output without exiting.
- **`inst.Unmount()`**: Executes all `UseEffectCleanup` hooks across the component tree and cleanly halts the engine.
- **`inst.LogWriter() io.Writer`**: Returns an `io.Writer` that formats log messages and inserts them cleanly into scrollback above the live frame.

```go
inst, _ := trail.Render(ctx, App())

// Later: update root component
inst.Rerender(UpdatedApp())

// Clear inline frame
inst.Clear()

// Unmount and exit
inst.Unmount()
```

## Log Interception (`WithInterceptLog`)

Pass `trail.WithInterceptLog(true)` to automatically patch the standard Go `log.Default()` output while the inline application is running. Log calls are routed above the active inline viewport and restored when the application exits:

```go
inst, err := trail.Render(ctx, App(), trail.WithInterceptLog(true))
// log.Println(...) now writes cleanly above the active inline frame!
```

## Hooks & Controls

### Keyboard Input (`UseInput`)

```go
trail.UseInput(func(input string, k event.KeyEvent) {
    if input == "p" {
        togglePause()
    }
}, trail.InputOptions{IsActive: isInputEnabled})
```

### Focus Management (`UseFocus`, `UseFocusManager`)

```go
btnRef := trail.CreateRef[dom.Element]()
isFocused := trail.UseFocus(btnRef, trail.FocusOptions{AutoFocus: true})

fm := trail.UseFocusManager()
// fm.FocusNext(), fm.FocusPrevious(), fm.Focus("element-id"), fm.Blur()
```

### Stdout & Stderr Routing (`UseStdout`, `UseStderr`)

```go
stdout := trail.UseStdout()
stdout.Write("Task completed!")
size := stdout.Size() // current viewport size

stderr := trail.UseStderr()
stderr.Write("Warning: slow connection")
```

### Element Measurement (`MeasureElement`)

```go
size := trail.MeasureElement(containerRef)
// size.Width, size.Height in terminal cells
```

### Text Line Transformation (`Transform`)

```go
trail.Transform(func(line string, idx int) string {
    return fmt.Sprintf("[%02d] %s", idx+1, line)
}, multilineText)
```
