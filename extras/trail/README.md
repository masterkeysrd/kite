# trail

`trail` is an Ink-like inline CLI framework built on [Kite](https://github.com/masterkeysrd/kite), its rendering `engine`, and the `kitex` reactive Virtual DOM framework.

## Features

- **Inline Execution**: Runs directly in the standard terminal scrollback without taking over the screen or entering an alternate screen buffer (`altscreen`).
- **Dynamic Content Height**: Automatically resizes the active terminal viewport to exactly match the rendered content height.
- **`<Static>` Output**: Permanently write completed logs, tasks, and items above the active inline frame into the terminal's native scrollback buffer while continuing to update live interactive components below.
- **Ink-Compatible Mental Model**: Declarative component tree with hooks like `UseApp()`, `UseInput()`, and primitives like `Box`, `Text`, `Static`, `Newline`, and `Spacer`.
- **Full kitex Interoperability**: Seamlessly use existing `kitex` components, state hooks (`UseState`, `UseReducer`), effect hooks (`UseEffect`, `UseLayoutEffect`), and refs (`UseRef`).

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/trail"
)

func App() kitex.Node {
	count, setCount := kitex.UseState(0)
	app := trail.UseApp()

	trail.UseInput(func(input string, k event.KeyEvent) {
		if input == "q" {
			app.Exit(nil)
		}
	})

	kitex.UseEffect(func() {
		ticker := time.NewTicker(time.Second)
		go func() {
			for range ticker.C {
				setCount(func(prev int) int { return prev + 1 })
			}
		}()
	}, nil)

	return trail.Box(trail.BoxProps{},
		trail.Text(fmt.Sprintf("Counter: %d (press 'q' to exit)", count)),
	)
}

func main() {
	if err := trail.Run(context.Background(), App()); err != nil {
		panic(err)
	}
}
```

## Static Output

Use `trail.Static` to permanently insert completed items above the active live UI:

```go
func TaskList(tasks []string) kitex.Node {
	return trail.Fragment(
		trail.Static(tasks, func(task string, index int) kitex.Node {
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
