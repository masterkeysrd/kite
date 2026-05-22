# Kite DOM Inspector

The Kite DOM Inspector is a web-based debugging tool that allows you to inspect the internal DOM tree, computed styles, and layout coordinates of a running Kite application.

## Usage

To use the inspector, call `inspector.Attach` after creating your engine:

```go
import (
    "github.com/masterkeysrd/kite/devtools/inspector"
    "github.com/masterkeysrd/kite/engine"
)

func main() {
    eng := engine.New(backend, opts)
    
    // Start the inspector on port 8080.
    if err := inspector.Attach(eng, "127.0.0.1:8080", inspector.Options{}); err != nil {
        log.Fatalf("failed to attach inspector: %v", err)
    }
    
    // ... rest of your application setup ...
    eng.Run(ctx)
}
```

Once your application is running, the inspector is available via the **Ctrl+I** hotkey (by default). When pressed, it will automatically attempt to open your default web browser to the dashboard URL. If the browser does not open, navigate to the address printed in the logs (typically `http://127.0.0.1:8080`).

### Port Handling

If the requested port is already in use, the inspector will automatically find an available random port and log the new address.

## Features

- **Live DOM Tree**: View the logical DOM hierarchy in real-time. The tree updates automatically via Server-Sent Events (SSE) whenever a frame is rendered.
- **Hot-key Access**: Press **Ctrl+I** to open the inspector browser window.
- **Dynamic Port Selection**: Automatically fallback to an available port if the default one is occupied.
- **Node Selection**: Click on any node in the tree to view its details.
- **Computed Styles**: Inspect the fully resolved computed styles for the selected node.
- **Layout Coordinates**: View the absolute bounding box (X, Y, Width, Height) of the selected node in terminal cells.

## How it Works

The inspector hooks into the Kite engine's `OnFrameRendered` lifecycle hook. Every time a frame is committed, the inspector:
1. Traverses the current DOM tree.
2. Calculates absolute bounding boxes by walking the fragment tree.
3. Serializes the state to JSON.
4. Broadcasts the update to all connected web clients via SSE.
