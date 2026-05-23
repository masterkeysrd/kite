# Kite Devtools

The `devtools` package provides essential debugging and inspection tools for Kite applications. It bridges the gap between terminal UI rendering and standard web-inspired development workflows.

## Features

- **DOM Inspector**: A web-based inspector that allows real-time visualization of the DOM tree, computed styles, and layout geometry.
- **X-Ray Mode**: A visual debugging mode that highlights layout nodes directly in the terminal, useful for diagnosing padding, margins, and border issues.
- **Floating Inspector Window**: Automatic launching of a Chromium-based browser in app mode to act as an external inspector dashboard.

## Usage

To enable devtools in your Kite application, use the `devtools.Install` function during your application setup:

```go
import (
    "github.com/masterkeysrd/kite/devtools"
    "github.com/masterkeysrd/kite/engine"
)

func main() {
    eng := engine.New(backend, opts)
    
    // Configure and install devtools
    err := devtools.Install(eng, devtools.Options{
        InspectorAddr: "127.0.0.1:8080",
    })
    if err != nil {
        log.Fatalf("failed to install devtools: %v", err)
    }
    
    // ... run engine ...
}
```

### Hotkeys

- **F12**: Toggles the web inspector (opens browser).
- **Ctrl+D**: Toggles X-Ray mode.

These can be configured via `devtools.Options`.

## Inspector Dashboard

When enabled, the inspector dashboard provides:
1. **DOM Tree**: Navigate the logical tree structure.
2. **Computed Styles**: Inspect box models and resolved CSS-style properties.
3. **Fragments**: View physical layout output.
4. **Layout Visualizer**: Interactive box model diagrams for selected elements.
