# ADR-010 - Intrinsic Style Layer

## Status
Accepted.

## Context
Replaced and compound elements need certain styles to be guaranteed regardless of what the author specifies. An `<input>` must be `display: inline-block` so it behaves as an atomic block within text flow; a `<textarea>` must use `white-space: pre-wrap` so newlines and wrap behavior match user expectations; both must `overflow: clip` so they act as a layout boundary and never paint outside their content box.

Today these guarantees are stated twice — once on the DOM element's `DefaultStyle()` and once when the bespoke render object initializes its `ComputedStyle`. Neither location actually prevents author overrides: `RawStyle()` cascades over `DefaultStyle()`, so an author who writes `input.Style().Display(Block)` silently breaks the element. The render-side reassignment is dead defense.

We need a cascade layer with strictly higher precedence than author styles for these UA-mandated properties, modeled cleanly after the CSS cascade origins (where the user-agent origin is one of the resolved layers).

We considered and rejected:
- **A post-resolution mutation hook on the element** (`OnStyleResolved`). Imperative, easy to forget, hard to test, hides the cascade.
- **Hard-coding forced properties inside the resolver per element type.** Type-switching in the resolver violates package isolation (style cannot depend on element types).
- **Doing nothing and relying on convention.** Has not held — bugs already exist.

## Decision
We introduce a new style layer that participates in the cascade with the strongest precedence and is contributed by the element itself.

### 1. Element Interface
The `dom.Element` interface gains a third style accessor alongside the existing two:

| Method | Cascade role | Author override? |
|---|---|---|
| `DefaultStyle() style.Style` | UA-default origin, weakest of the element-contributed layers | **Yes** — author wins. |
| `RawStyle() style.Style` | Author origin | n/a (this *is* the author input) |
| `IntrinsicStyle() style.Style` | **NEW** — UA-intrinsic origin, strongest | **No** — author cannot override. |

Like `DefaultStyle` and `RawStyle`, `IntrinsicStyle` returns a sparse `style.Style` with `Optional[T]` fields. Only the properties the element wishes to force are set; everything else cascades normally.

### 2. Resolver Cascade Order
The `style.Resolver` is extended to apply layers in this order (lowest → highest precedence):
1. Inherited values from the parent's `Computed` (existing behavior).
2. `DefaultStyle()` of the element.
3. `RawStyle()` of the element (author).
4. **`IntrinsicStyle()` of the element (UA-forced).** New top layer.

Internally the resolver tags the new layer with the origin `OriginUserAgent` for spec alignment, in case we later expose cascade origins to author tooling.

### 3. Inheritance Rules
- `IntrinsicStyle` participates in the same property-by-property `Optional[T]` merge as the other layers.
- It does **not** introduce new inheritable properties; if an intrinsic value happens to be on an inherited property (e.g., `WhiteSpace`), it inherits to descendants normally — which is desirable for `<textarea>` since the inner UA text node should inherit `white-space: pre-wrap`.

### 4. Scope
- This decision adds a *layer*. It does not change which properties exist, which properties inherit, or how `Computed` is consumed downstream.
- It does not introduce per-cell or per-fragment style storage; the layer is collapsed into the final `style.Computed` exactly as before.

### 5. Relationship to UA Shadow Subtree (ADR-009)
- ADR-009 establishes *what* a replaced element is structurally. ADR-010 establishes *how* its styling guarantees are enforced.
- Together they replace ADR-007: forced styling no longer lives in two places.

## Consequences
### Positive
- UA-mandated properties are enforced declaratively in a single, discoverable location per element type.
- Author overrides for non-forced properties continue to work exactly as before.
- The cascade gains a spec-aligned UA origin layer that future tooling (devtools, style debugging) can surface.
- Eliminates the silent-disagreement bug between DOM defaults and render-object hard-coded styles.

### Negative / Trade-offs
- One new method on every `dom.Element` implementation. The default implementation returns an empty `style.Style`, so most elements pay zero cost.
- The cascade gains one more merge pass per resolution. The pass is sparse (most elements return empty), so the cost is dominated by elements that actually use it.
