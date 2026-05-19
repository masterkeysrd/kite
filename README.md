# Kite

Kite is a modern, DOM-like terminal UI framework for Go. It brings web-like development paradigms—such as a logical DOM tree, CSS-style flexbox layout, and a capture/target/bubble event model—to the terminal environment. 

## 🏗 Architecture Overview

Kite is built with a clean separation of concerns, divided into specialized packages that form the rendering pipeline:

*   **DOM (`/dom`)**: The logical node tree representing the user interface. It contains core entities like `Document`, `Element`, and `TextNode`. It implements strict lifecycle hooks (Connected/Disconnected), identity registration, and semantic state (e.g., Focus, Disabled).
*   **Style (`/style`)**: A CSS-like styling engine using an `Optional[T]` pattern to allow sparse style definitions. It supports flexbox, box model dimensions, and terminal-specific text formatting.
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
├── dom/        # Logical node tree, Element, Document, and TextNode
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

While Kite operates under the hood with a full render loop, manually constructing a UI looks similar to web DOM manipulation:

```go
package main

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func main() {
    // A concrete backend/render loop setup is required to draw this tree
    // However, the DOM structure is built like this:
	
	// Create a document and a root element
	// (Note: usually initialized by the engine)
	var doc dom.Document 
	
	container := element.NewBox(doc)
	container.SetID("main-container")
	
	// Create a list
	list := element.NewUnorderedList(doc)
	list.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Item 1")))
	list.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Item 2")))
	container.AppendChild(list)
	
	// Create a table
	table := element.NewTable(doc)
	row := element.NewTableRow(doc)
	cell1 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Cell 1"))
	cell2 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Cell 2"))
	row.AddChild(cell1).AddChild(cell2)
	table.AppendChild(row)
	container.AppendChild(table)

	text := element.NewText(doc, "Hello, Kite!")
	container.AppendChild(text)
	
	// Apply styles (Sparse assignment via Optional[T])
	myStyle := style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
	}
	
	_ = myStyle
}
```
