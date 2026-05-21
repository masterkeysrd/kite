# Kite

Kite is a modern, DOM-like terminal UI framework for Go. It brings web-like development paradigms—such as a logical DOM tree, CSS-style flexbox layout, standard event propagation, and scrolling—to the terminal environment. 

## 🏗 Architecture Overview

Kite is built with a clean separation of concerns, divided into specialized packages that form the rendering pipeline:

*   **DOM (`/dom`)**: The logical node tree representing the user interface. It contains core entities like `Document`, `Element`, and `TextNode`. It implements strict lifecycle hooks (Connected/Disconnected), identity registration, semantic state (e.g., Focus, Disabled), and a **closed UA Shadow Subtree** primitive (ADR-009) that allows replaced and compound widgets to compose their visuals as a private, author-invisible DOM subtree.
*   **Style (`/style`)**: A CSS-like styling engine using an `Optional[T]` pattern to allow sparse style definitions. It supports flexbox, box model dimensions, and terminal-specific text formatting. The resolver applies a **four-layer cascade** (weakest → strongest): inherited values, element-type defaults (`DefaultStyle()`), author styles (`RawStyle()`), and UA-intrinsic styles (`IntrinsicStyle()`). The intrinsic layer lets replaced elements (e.g. `<input>`) enforce UA-mandated properties (like `display: inline-block`) that author code cannot override (ADR-010).
*   **Layout (`/layout`)**: The high-performance, LayoutNG-inspired engine responsible for computing geometry. It takes computed styles and constraints, and returns immutable `Fragment` trees.
*   **Paint (`/paint`) & Backend (`/backend`)**: The drawing layer. The `paint` package interfaces with a framebuffer to draw absolute coordinates with clipping, while `backend` decouples the engine from the actual terminal output (using Charmbracelet's `ultraviolet` or a mock backend for tests).
*   **Render (`/render`)**: The visual bridge. It holds a unified `render.Box` or `render.Text` tree that perfectly mirrors the DOM, carrying lifecycle dirty-flags (`NeedsSync`, `DirtyStyle`, `DirtyLayout`) without doing actual math.
*   **Engine (`/engine`)**: The central nervous system. It orchestrates the 6-phase pipeline (Task Draining -> Sync -> Style -> Layout -> Paint -> Commit) at 60FPS on the main thread, while managing concurrent asynchronous Jobs.
*   **Event (`/event`)**: An advanced event dispatcher supporting capture, target, and bubble phases. It includes synthesizers to translate raw terminal input into semantic events (e.g., clicks).

## 🚀 Getting Started

### Prerequisites

*   **Go 1.26.1** or higher.

### Installation

Add Kite to your project using `go get`:

```bash
go get github.com/masterkeysrd/kite
```

### Local Development & Testing

Run the standard test suite:

```bash
go test ./...
```

Run the benchmarks to verify rendering and layout performance:

```bash
go test -bench=. ./...
```

## 📁 Project Structure

```text
github.com/masterkeysrd/kite
├── backend/    # Terminal decoupling, mock, and ultraviolet implementations
├── dom/        # Logical node tree, Element, Document, TextNode, and TextArea
├── editor/     # Text editing buffers and Unicode-safe string mutation
├── event/      # Event dispatching, synthetic events, and keystroke helpers
├── focus/      # Focus management and spatial navigation
├── key/        # Key codes and modifiers
├── layout/     # Geometry calculations and layout engine
├── paint/      # Drawing interfaces and framebuffer management
├── render/     # The rendering pipeline tying DOM and Layout together
├── style/      # Sparse styling, computed values, and resolvers
└── text/       # Text shaping and grapheme cluster management
```

## 💻 Usage Example

Kite provides a declarative, SwiftUI-inspired API for constructing UI trees. Thanks to **Implicit DOM Adoption**, you can build complex structures without threading a document reference:

```go
package main

import (
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	// Build a UI tree declaratively
	ui := element.Box(
		element.Box(
			"Welcome to Kite!",
		).Style(style.Style{
			Padding: style.Some(style.EdgeValues[int]{Top: 1, Bottom: 1}),
		}),

		element.UL(
			element.LI("High Performance (60FPS)"),
			element.LI("Declarative Syntax"),
			element.LI("Flexbox Layout"),
		),

		element.Table(
			element.TR(
				element.TD("Cell 1"),
				element.TD("Cell 2"),
			),
		),
	)

	// In a real application, you would mount this to the engine:
	// eng.Mount(ui)
	_ = ui
}
```
