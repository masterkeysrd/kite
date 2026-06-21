# Kitex Stage

The `stage` package provides an interactive component development playground and sandbox for Kitex applications. It allows you to build, test, and preview components in complete isolation without running your main application.

## Features

- 🎭 **Isolated Scenes**: Register components with different mock scenarios (e.g. Primary, Secondary, Disabled).
- 🛠️ **Dynamic Controls**: Add interactive knobs (`c.Text()`, `c.Bool()`, `c.Select()`, `c.Int()`) to tweak properties and styles in real-time.
- 🌍 **Global Controls**: Register toolbar controls (e.g. Theme, Locale) that affect all components and persist across scene selections.
- 🎨 **Context-Aware Decorators**: Inject global providers (like Theme contexts) or layout wrappers around all scenes, driven by active global settings.
- 📋 **Event Action Log**: Capture and view component actions and callback events locally.
- ⌨️ **Keyboard Navigation**: Use global hotkeys to switch scenes without leaving the keyboard.
- 🔍 **Devtools Integration**: Integrates directly with standard Kite web-inspector devtools (`devtools.Install`) for layout and DOM inspections.

## Usage

Create a standalone playground command file (e.g., `cmd/stage/main.go`):

```go
package main

import (
	"image/color"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/stage"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	stg := stage.New()

	// 1. Register global controls (toolbar controls)
	stg.GlobalSelect("Theme", []string{"Light", "Dark"}, "Dark")
	stg.GlobalBool("Show Border", true)

	// 2. Wrap all scenes with a context-aware decorator
	stg.WithContextDecorator(func(c *stage.Context, node kitex.Node) kitex.Node {
		theme := c.GlobalString("Theme", "Dark")
		showBorder := c.GlobalBool("Show Border", true)

		var bgColor color.RGBA
		if theme == "Light" {
			bgColor = color.RGBA{R: 243, G: 244, B: 246, A: 255}
		} else {
			bgColor = color.RGBA{R: 17, G: 24, B: 39, A: 255}
		}

		wrapperStyle := style.S().Background(bgColor).Padding(2)
		if showBorder {
			wrapperStyle = wrapperStyle.Border(true, style.BorderSingle, color.RGBA{R: 79, G: 70, B: 229, A: 255})
		}

		return kitex.Box(kitex.BoxProps{Style: wrapperStyle}, node)
	})

	// 3. Register components and scenes
	stg.Register("Button", []stage.Scene{
		{
			Name: "Default",
			Render: func(c *stage.Context) kitex.Node {
				text := c.Text("Label", "Click Me")
				disabled := c.Bool("Disabled", false)

				return kitex.Button(kitex.ButtonProps{
					Disabled: disabled,
					OnClick: func(e event.Event) {
						c.Log("Button clicked!")
					},
				}, kitex.Text(text))
			},
		},
	})

	stg.Register("Counter", []stage.Scene{
		{
			Name: "Basic",
			Render: func(c *stage.Context) kitex.Node {
				// Integer controls render as steppers (+ / -) in the panel
				count := c.Int("Limit", 10)

				return kitex.Box(
					kitex.BoxProps{},
					kitex.Text(fmt.Sprintf("Limit: %d", count)),
				)
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
