# ADR 033: Color Blending and Opacity Simulation

## Status
Accepted

## Context
Terminals do not natively support transparency or alpha channels for text or box background colors. To support modern UI features like semi-transparent dialog overlays, hover state tints, and glassmorphism-like effects in a TUI, the engine must simulate opacity by mathematically blending overlapping background colors.

Initial attempts to resolve color blending at the `PaintEngine` level (by propagating the parent background color down the fragment tree recursion) failed when rendering overlays because overlays reside in a completely separate root-level tree and do not inherit the parent block's background color. Furthermore, overlapping elements with semi-transparent backgrounds could trigger double-blending if their child components (such as text nodes or borders) re-rendered the background color on top of already-filled framebuffer cells.

## Decision
We decided to implement color blending directly inside the framebuffer layer (`FrameBuffer.Set`).
1. **At-The-Glass Blending**: When writing a cell via `FrameBuffer.Set`, if `c.Bg` is a non-transparent color with an alpha channel < 255, we blend it directly with whatever color is currently stored in that cell's background in the framebuffer.
2. **Double-Blending Prevention**:
   - We propagate the background color of the nearest block-level parent (`blockBg`) down the `paintFragment` recursion.
   - If an element's background matches `blockBg`, we skip `fillRect` to prevent writing the background a second time.
   - If a text node's background matches `blockBg`, we paint it with `Bg: color.Transparent` to overlay characters directly on top of the already-blended cell background.
   - Borders and scrollbars are painted with `Bg: color.Transparent` for the same reason.
3. **Zero-Allocation Cache**: The `FrameBuffer` hosts its own `blendCache map[blendKey]color.Color` to memoize the linear interpolation calculations, ensuring O(1) rendering time with zero heap allocations on hot paths. The cache is cleared when it exceeds 1024 entries.

## Consequences
### Positive
- Overlays, dialogs, and nested elements blend correctly against the actual content underneath them.
- High performance (~19 ns per write) with 0 memory allocations on cached hits, maintaining 60FPS target.
- Correctly handles text spans and inline elements with custom backgrounds without double-blending.

### Negative / Trade-offs
- Blending requires that elements be painted in strict back-to-front order (which Kite already enforces natively).
- The default terminal color (`style.TerminalDefault`) is treated as transparent for blending purposes, so nested elements inherit parent block colors rather than punching holes to show the terminal emulator background.
