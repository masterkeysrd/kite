# TSK-033: Implement Terminal X-Ray Mode

## Status
In Progress

## Overview
Add visual layout debugging directly into the core `paint` engine, toggleable via the devtools package, to overlay bounding boxes on the terminal interface.

## Requirements

1. **Engine Flag:**
   - Add a developer flag to the `engine.Engine` or `paint.Context` (e.g., `DebugXRay bool`).

2. **Paint Phase Interception:**
   - Modify the `paint` phase traversal. When `DebugXRay` is true, after a `render.Box` paints its background, text, and standard borders, draw the X-Ray overlays.
   - Use distinct colors:
     - **Content Box:** Blue border/tint.
     - **Padding Box:** Green border/tint.
     - **Margin Box:** Red border/tint.
   - *Constraint:* Ensure X-Ray lines do not permanently alter the real DOM layout or text flow. They are purely visual post-paint additions.

3. **DevTools Toggle:**
   - Provide a hook in `kite/devtools` to bind a hotkey (e.g., `Ctrl+D`) that toggles this `DebugXRay` flag on the running engine.

## Testing & Verifications
- Add unit tests in the `/paint` package ensuring that when `DebugXRay` is true, the resulting framebuffer contains the expected colored border cells.
- Verify that standard clipping (overflow hidden) still applies correctly to the X-Ray boxes.