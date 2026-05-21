# TSK-032: Implement Web-Based DOM Inspector (SSE)

## Overview
Build a lightweight, out-of-band debugging server that streams Kite's internal DOM and layout state to a web browser using Server-Sent Events (SSE).

## Requirements

1. **Inspector Server (`devtools/inspector`):**
   - Implement `Attach(eng *engine.Engine, addr string)` which starts an HTTP server alongside the Kite application.
   - Serve a minimal, static HTML/JS dashboard at `/`.

2. **SSE Streaming Endpoint:**
   - Expose `/stream` handling the SSE protocol (`text/event-stream`). This is a unidirectional stream where the server sends updates to the client (browser) automatically.
   - Hook into the Kite engine's frame lifecycle (or a dirty flag). When a frame finishes rendering, serialize the logical DOM tree and the associated physical bounding boxes into JSON.
   - Push the JSON payload to all connected SSE clients.

3. **Dashboard UI:**
   - The dashboard should parse the JSON and display an expandable/collapsible tree view of the DOM.
   - Display computed styles and layout coordinates (X, Y, W, H) for the currently selected node.

4. **Engine Hooks:**
   - You may need to expose an `OnFrameRendered` hook or observer pattern in the core `/engine` package to allow the inspector to trigger its SSE broadcast.

## Testing & Verifications
- Write integration tests verifying the HTTP server starts and the `/stream` endpoint successfully emits a JSON payload when the Kite app updates.