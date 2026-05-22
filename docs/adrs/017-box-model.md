# ADR 017: Strict Border-Box and Non-Collapsing Margins

## Status
Accepted

## Context
Standard CSS allows developers to toggle between `content-box` and `border-box` via the `box-sizing` property. It also implements "Margin Collapsing" in Block Formatting Contexts, where adjacent vertical margins combine into the largest single margin, and child margins can "leak" outside of parents lacking borders or padding.

In a Terminal UI (TUI), coordinates are discrete character cells. Predictable, rigid grid math is paramount. Web UI developers overwhelmingly override CSS defaults to use `box-sizing: border-box` to make component sizing predictable, yet they still battle margin collapsing logic. 

As Kite aims to be a modern UI framework, we must decide whether to mimic web defaults (for familiarity) or enforce strict, simplified rules optimized for TUI layouts.

## Decision

We will depart from CSS defaults to enforce a simplified, strict Box Model globally across the framework:

### 1. Strict `border-box` Sizing
Kite will not implement a `box-sizing` property. Every layout algorithm (`BlockAlgorithm`, `FlexAlgorithm`, `TableAlgorithm`) will unconditionally evaluate author-provided `Width` and `Height` styles as the **`border-box`**.
- If a developer sets `Width: 10` and `Border: Single`, the framework reserves exactly 10 cells of screen real estate. The internal content area will be exactly 8 cells wide (10 - left border - right border).
- Padding further reduces the available content area.

### 2. No Margin Collapsing
Kite will **never** collapse margins.
- If Block A has `MarginBottom: 1` and adjacent sibling Block B has `MarginTop: 1`, there will be exactly 2 empty cells between their borders.
- Margins will never "leak" outside of a parent. A child with `MarginTop: 1` will always push itself down 1 cell relative to the parent's inner content edge.

## Consequences

### Positive
- **Predictability:** Layout math becomes incredibly simple and completely additive: `OuterSpace = Margin + Border + Padding + Content`.
- **Developer Experience:** Eradicates the notorious layout bugs caused by margin collapsing. Developers get exactly the amount of spacing they ask for.
- **Performance:** Layout algorithms do not need complex lookahead or sibling-state tracking to calculate margin unions.

### Negative
- **Spec Divergence:** Authors attempting to port raw CSS directly into Kite may find vertical spacing slightly larger than expected if they relied heavily on margin collapsing.
