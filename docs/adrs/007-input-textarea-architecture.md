# ADR-007 - Input and TextArea Architecture

## Status
**Deprecated.** Superseded by **ADR-009 (UA Shadow Subtree for Replaced Elements)** and **ADR-010 (Intrinsic Style Layer)**.

The "Replaced Element via Direct Casting" pattern described below does not scale to compound widgets (checkbox, radio, select, slider) and leads to duplicated styling guards and bespoke layout algorithms for every new form control. Replaced elements now compose their visuals as a closed UA Shadow Subtree (ADR-009) and enforce UA-mandated styles through the Intrinsic Style Layer (ADR-010). `render.CustomObjectProvider` (TSK-016) remains available for elements whose visuals genuinely cannot be expressed as a subtree.

## Context
Kite requires interactive text input widgets (`input` and `textarea`). While browsers implement these as replaced elements using complex Shadow DOM structures, introducing Shadow DOM into Kite would add unnecessary complexity. However, we must still respect the separation of concerns: logical DOM nodes manage state and events, while render objects handle physical layout and geometry. 

## Decision
We will implement `input` and `textarea` as **Replaced Elements**, mimicking the existing pattern used by `render.Text`.

1. **Custom Render Object Hook:**
   - Add a `CustomRenderObjectProvider` interface in the `render` package to allow logical nodes to instantiate specialized render objects during the engine's Sync phase.

2. **Replaced Render Objects via Direct Casting:**
   - Introduce `render.Input` and `render.TextArea`.
   - These objects embed `render.Box` and act as atomic blocks to the standard layout engine.
   - During `Layout()` and `Paint()`, they cast their `LogicalNode()` back to `*element.Input` or `*element.TextArea` to directly read the raw string, cursor index, and scroll offsets.
   - The render objects perform their own `text.Shape()` calls, layout math, and fragment generation. This avoids complex synchronization interfaces and keeps the render objects stateless.

3. **Cursor Abstraction:**
   - Create a new `cursor` package (`cursor.State`, `cursor.Shape`) to prevent cyclic dependencies between the engine and DOM.
   - The engine will query the *Render Object* of the currently focused node (via a `cursor.Provider` interface) to obtain the absolute screen coordinates for the hardware cursor.

4. **Logical Text Controller:**
   - To safely navigate strings via keyboard events (handling grapheme clusters, words, and boundaries), we will build a 1D `text.Buffer` utility.
   - 2D geometric navigation (e.g., `Up`/`Down` in wrapped textareas) will be facilitated by the render object querying its layout fragments and returning the target index to the logical DOM.

## Consequences
- **Positive:** Pragmatic implementation of replaced elements without Shadow DOM overhead.
- **Positive:** Matches the existing `render.Text` pattern, ensuring consistency.
- **Positive:** Strict separation is maintained; logical DOM knows nothing of terminal cell widths or screen coordinates.
- **Negative:** `render.Input` is tightly coupled to `element.Input`, but this is acceptable and matches standard browser implementations for replaced media/form controls.
