# 🪁 Kite

Kite is a modern, DOM-like terminal UI framework for Go. It brings web-inspired development paradigms—such as a logical DOM tree, CSS-style flexbox layout, standard event propagation, and scrolling—to the terminal environment.

## ✨ Features

- 🌳 **Logical DOM Tree**: Core entities like `Document`, `Element`, and `TextNode` with strict lifecycle hooks.
- 🎨 **Style Engine**: CSS-like styling using an `Optional[T]` pattern for sparse definitions.
- 📐 **Layout Engine**: High-performance, LayoutNG-inspired engine responsible for computing geometry.
- 🏎️ **60FPS Pipeline**: Orchestrated 6-phase pipeline (Sync -> Style -> Layout -> Paint -> Commit).
- 🖱️ **Advanced Events**: Support for capture, target, and bubble phases, plus synthetic event management.
- 🧪 **Headless Testing**: Simulate user input and assert on DOM state or visual regression (golden files).
- ✨ **Animations**: Imperative property interpolation and tweening system for smooth transitions.
- 🛠️ **Developer Tools**: Web-based DOM inspector, in-terminal X-Ray mode, and performance profiler.
- ⚛️ **Reactive Primitives**: Lightweight Virtual DOM (VDOM) for declarative, type-safe API (via [`extras/kitex`](extras/kitex/)).

## 🏗 Architecture Overview

Kite is built with a clean separation of concerns, divided into specialized packages that form the rendering pipeline:

- **`dom`**: The logical node tree representing the user interface. It implements strict lifecycle hooks, identity registration, semantic state, and a top-layer overlay API for out-of-flow elements.
- **`style`**: CSS-like styling engine using an `Optional[T]` pattern. It supports a four-layer cascade: inherited values, element-type defaults, author styles, and intrinsic styles.
- **`layout`**: The high-performance engine responsible for computing geometry, returning immutable fragment trees.
- **`paint` & `backend`**: The drawing layer and terminal decoupling, allowing for varied output backends.
- **`render`**: The visual bridge mirroring the DOM to the layout engine, carrying lifecycle dirty-flags.
- **`engine`**: The central nervous system orchestrating the pipeline at 60FPS while managing concurrent jobs.
- **`event`**: Advanced event dispatcher supporting capture, target, and bubble phases.
- **`animation`**: Imperative property interpolation and tweening system for smooth transitions.

## 🚀 Getting Started

### Prerequisites

- **Go 1.26.1** or higher.

### Installation

```bash
go get github.com/masterkeysrd/kite
```

### Usage Example

Kite provides a declarative, type-safe API for constructing UI trees.

```go
package main

import (
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/event"
)

func main() {
	// Build a UI tree declaratively
	ui := element.Box(
		element.Box(
			"Welcome to Kite!",
		).Style(style.Style{
			Padding: style.Some(style.Edges(1, 0)),
			Bold:    style.Some(true),
		}),

		element.UL(
			element.LI("High Performance"),
			element.LI("Flexbox & Grid Layouts"),
		),

		element.Button("Click Me").OnEvent(event.EventClick, func(e event.Event) {
			// Handle click
		}),
	)

	// In a real application, you would mount this to the engine:
	// eng.Mount(ui)
	_ = ui
}
```

## 🧪 Headless Testing

Kite provides a headless testing environment via the `devtools/testenv` package. This allows you to simulate user input and assert on the DOM state without a physical terminal.

```go
func TestMyApp(t *testing.T) {
    env := testenv.Default(80, 24)
    defer env.Close()

    env.Mount(element.Input("").WithID("my-input"))
    env.Flush()

    env.Type("hello")
    env.Flush()

    input := env.GetNodeByID("my-input").(*element.InputElement)
    if input.Value() != "hello" {
        t.Errorf("expected 'hello', got %q", input.Value())
    }

    // Visual Regression Testing (Golden Files)
    env.MatchGolden(t, "input-state")
}
```

## 🛠 Developer Tools

Kite includes a unified developer tools package that provides a web-based DOM inspector, an in-terminal X-Ray mode, and a performance profiler.

```go
import "github.com/masterkeysrd/kite/devtools"

// ... after creating your engine
insp, _ := devtools.Install(eng, devtools.Options{
    InspectorAddr: "127.0.0.1:8080",
})
```

### Web-Based DOM Inspector
Open the inspector window in your browser to debug your application's logical tree, computed styles, and layout box model in real-time. It features a live DOM tree, style layer inspection, and a visual box model representation.

### Terminal X-Ray Mode
Toggle colored bounding boxes directly on your running application for immediate layout debugging.
- **Red**: Margin Box
- **Green**: Padding Box
- **Blue**: Content Box

### Performance Profiler
Analyze execution durations for the rendering pipeline phases and asynchronous background jobs with interactive flamecharts and Chrome Trace export.

## 📖 Documentation & Guides

For more detailed information, please refer to the following guides in the `docs/` directory:

- [Quickstart Guide](docs/guides/QUICKSTART.md)
- [API Overview](docs/guides/API_OVERVIEW.md)
- [Architecture Deep Dive](docs/architecture.md)
- [Testing Guide](docs/guides/TESTING.md)
- [Examples](examples/)

## 🤝 Contributing

Contributions are welcome! Please check our [Developing Guide](docs/guides/DEVELOPING.md) to get started.
