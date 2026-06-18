# Kitex Stage

The `stage` package provides an interactive component development playground and sandbox for Kitex applications. It allows you to build, test, and preview components in complete isolation without running your main application.

## Features

- 🎭 **Isolated Scenes**: Register components with different mock scenarios (e.g. Primary, Secondary, Disabled).
- 🛠️ **Dynamic Controls**: Add interactive knobs (`c.Text()`, `c.Bool()`, `c.Select()`) to tweak properties and styles in real-time.
- 📋 **Event Action Log**: Capture and view component actions and callback events locally.
- ⌨️ **Keyboard Navigation**: Use global hotkeys to switch scenes without leaving the keyboard.
- 🔍 **Devtools Integration**: Integrates directly with standard Kite web-inspector devtools (`devtools.Install`) for layout and DOM inspections.

## Usage

Create a standalone playground command file (e.g., `cmd/stage/main.go`):

```go
package main

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/stage"
)

func main() {
	stg := stage.New()

	stg.Register("Button", []stage.Scene{
		{
			Name: "Default",
			Render: func(c *stage.Context) kitex.Node {
				text := c.Text("Label", "Click Me")
				disabled := c.Bool("Disabled", false)

				return kitex.Button(kitex.ButtonProps{
					Text:     text,
					Disabled: disabled,
					OnClick: func(e event.Event) {
						c.Log("Button clicked!")
					},
				})
			},
		},
		{
			Name: "Primary Theme",
			Render: func(c *stage.Context) kitex.Node {
				return kitex.Button(kitex.ButtonProps{
					Text: "Primary",
				})
			},
		},
	})

	stg.Run()
}
```

Then run it in your terminal:
```bash
go run cmd/stage/main.go
```

## Hotkeys

- **Up / Down Arrow**: Select and switch between different scenes in the sidebar.
- **Tab**: Cycle focus to input fields inside the controls panel.
- **Ctrl+C**: Quit Stage.
- **F12**: Toggles the web-inspector devtools browser dashboard.
