# ADR-011 - Paint-Phase Overflow Clipping

## Status
Accepted.

## Context
The `style.Overflow` enum has been part of the codebase for some time, with values `Visible`, `Hidden`, `Scroll`, and `Clip` (see `style/enums.go`). The two-axis fields `OverflowX` and `OverflowY` are present on both `style.Style` and `style.Computed` and are resolved correctly by the cascade.

However, the paint phase (`paint/engine.go::paintFragment`) **never reads** these properties. The `Surface.Clip(rect)` API and the `clippedSurface` wrapper exist (`paint/framebuffer.go:74-77, 116-146`) but have zero callers in the codebase. The framebuffer's only clipping today is its silent drop of writes outside the terminal viewport (`framebuffer.go:43-45`).

This gap has been latent because most existing widgets do not set `Overflow != Visible`. Input and textarea (ADR-007, deprecated) "worked around" the gap by placing scroll-shifted text at negative offsets and relying on terminal-viewport clamping — which only happens when the input is small enough and far enough from other widgets that the spill happens to fall off-screen. ADR-009 (UA Shadow Subtree) raises the stakes: replaced form controls now declare `OverflowX/Y: clip` through their intrinsic style (ADR-010) and *expect that promise to be enforced*.

We considered:
- **Per-fragment per-cell `clip` flags written by paint:** too granular, would require revisiting every paint helper.
- **A separate "clip pass" after paint:** repaints would have to discard cells that paint placed outside content boxes, doubling work.
- **A `frag.ClipRect` field set during layout:** mixes concerns; layout would need to know painter intent.
- **Wire up the existing `Surface.Clip()` plumbing inside `paintFragment`:** smallest, most local change that connects two already-correct subsystems.

## Decision
The paint engine will apply per-fragment clipping by composing `Surface.Clip()` calls during the recursion in `paintFragment`. The mechanism is:

### 1. Scope of `OverflowX` / `OverflowY` Treated by Paint
| Value | Paint behavior |
|---|---|
| `OverflowVisible` (spec default) | No clip — children may paint anywhere. |
| `OverflowHidden` | Clip descendants to the content box. |
| `OverflowClip` | Clip descendants to the content box. Identical to `Hidden` in paint terms; the semantic difference (scrollability) is irrelevant here. |
| `OverflowScroll` | Clip descendants to the content box; the scroll translation itself is the concern of ADR-012 (Generic Scroll Offset). |
| `OverflowAuto` | Clip descendants to the content box (added by ADR-012). |

### 2. Clip Rect = Content Box
The clip applied to descendants equals the fragment's **content box** — the border-box minus border widths minus padding. The fragment's own background (border-box) and border (outer edges) paint **before** the clip is established and are therefore **not** clipped by the parent's own `overflow`. This matches CSS: an element with `overflow: hidden` does not clip its own border.

### 3. Composition with Nested Overflow Boxes
`clippedSurface.Clip` already composes by intersection (`paint/framebuffer.go:135-137`). Paint passes the resulting `Surface` down the recursion so nested overflow boxes naturally intersect their clip rects. No state needs to live outside the recursion call stack.

### 4. Border Resolver Invariant
The global border post-processor (`paint/engine.go::resolveBorders`) runs **once on the root surface** after all painting completes. It must never be invoked on a clipped sub-surface. This is the current behavior; ADR-011 codifies it as an invariant so future changes do not regress it.

### 5. Independence from Scrolling
This ADR does **not** introduce scroll offsets, scroll containers, or wheel-event wiring. Those belong to ADR-012. ADR-011 only makes the existing `clip`/`hidden` styles actually clip.

### 6. Relation to the UA Shadow Subtree Work
Without this ADR, the `IntrinsicStyle().OverflowX(Clip).OverflowY(Clip)` declared by `<input>` and `<textarea>` (ADR-009/ADR-010) is silently ignored by paint. TSK-027 implements this ADR and is a hard prerequisite for the correctness of TSK-024 and TSK-025.

## Consequences

### Positive
- The `style.Overflow` enum becomes meaningful in paint — closes a longstanding correctness gap.
- Reuses already-implemented machinery (`Surface.Clip`, `clippedSurface`, `Rect.Intersect`); no new abstractions.
- Composable: nested overflow boxes work for free via intersection.
- Independent of scrolling — can ship before ADR-012 and benefits other widgets immediately (e.g., a fixed-size card with overflowing children).

### Negative / Trade-offs
- One small allocation per overflow box per paint recursion (the `clippedSurface` wrapper). Negligible in practice; can be pooled later if benchmarks demand.
- Authors who relied on "overflow: clip / hidden are silently visible" will see their layouts change. This is a correctness fix; no existing in-tree test depends on the old behavior.
