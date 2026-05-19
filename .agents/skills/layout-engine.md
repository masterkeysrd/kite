---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: layout-engine
  description: Instructions and architectural patterns for operating on the Kite layout engine (LayoutNG-inspired immutable fragment tree).
  displayName: Layout Engine
---

# Layout Engine Skill

The Kite `layout` package uses a modern, high-performance architecture inspired by Chromium's LayoutNG. It is designed around the concept of immutable layout fragments and constraint spaces to optimize rendering efficiency and minimize re-layouts.

## 1. Immutable Fragment Tree
The output of the layout process is an **Immutable Fragment Tree**. 
* **Fragments are immutable** after creation. Do not modify an existing fragment's dimensions or positions after layout.
* Fragments do not contain their positioning information relative to their parent directly inside the fragment body. Positioning is handled by links or wrappers (like `NGLink` in LayoutNG) to allow caching and reusing identical fragments across different layout constraints.
* Do not introduce "up" references (pointers from child to parent) inside fragments. A child only reads information from its constraints, not from inspecting its parent.

## 2. Layout Constraints & Inputs
Every layout operation receives input constraints (similar to `NGConstraintSpace` in LayoutNG).
* The inputs to lay out a node are the **Layout Node** (which holds computed styles) and the **Constraint Space** (available width, height, and rules like BFC boundaries).
* A render object caching this information can determine if it needs to re-measure. If the layout node's computed styles haven't changed and the input constraint space is identical, the engine should immediately return the cached immutable fragment.

## 3. Two-Pass Layouts and Performance
* Advanced layouts like Flexbox often require **two-pass layouts** (a measure pass to determine intrinsic child sizes, followed by a layout pass to stretch/align them).
* To prevent exponential layout times ($O(2^n)$), the engine **must** cache the results of both the measure and layout passes independently.
* Rely on the difference between `old_constraints` and `new_constraints` to decide if invalidation is necessary. For example, a fixed-width child does not need re-layout if only the parent's available width expands, whereas a percentage-width child does.

## 4. Block Fragmentation & Breaking
* Layouts must support the concept of breaking (e.g., across columns or pages). 
* Use break tokens (`BreakToken`) to pause and resume layout of a node across fragmentainers. 
* Break decisions should keep track of "early breaks" to find optimal unforced breakpoints when rules like `break-before: avoid` or `orphans` are encountered.

## 5. Inline Text Layout
Text layout is executed via a specialized inline pipeline to maintain high performance.
* **Pre-layout:** Collects text, collapses whitespace, and determines BiDi runs.
* **Line Breaking:** Measures text items and breaks them into logical lines based on the constraint space.
* **Line Box Construction:** Builds physical text fragments and orders them visually (handling Bidirectional text via UAX#9). Uses a flat list structure rather than deep nested boxes where possible for memory locality.
* Text shapes should be heavily cached (word-level or paragraph-level) to avoid hitting the shaper repeatedly for static text.

## 6. RenderObject Unified Caching Bridge
The `render.Object` sits between the logical DOM and the layout engine.
* It is responsible for storing dirty flags.
* It caches the previous `ConstraintSpace` and the resulting `ImmutableFragment`.
* **Unified Box:** There are only two render types: `render.Box` and `render.Text`. The engine does not swap objects if `Display` changes. The `layout` engine dynamic selects `BlockAlgorithm` or `FlexAlgorithm` based on the Box's `ComputedStyle.Display`.
* **Rule**: When `layout.Compute()` is called, if the dirty flag is false and the incoming constraints match the cached constraints, return the cached fragment immediately.

## 7. Separation of Layout and Paint
* Layout calculates the physical sizes and bounds (creating the Immutable Fragment Tree). 
* The fragment tree does not contain drawing logic. Drawing is deferred to the `/paint` layer, which walks the finalized fragment tree.
* Layout must execute extremely fast (aiming for a 60FPS cycle) by maximizing caching and avoiding deep layout invalidation whenever possible.

