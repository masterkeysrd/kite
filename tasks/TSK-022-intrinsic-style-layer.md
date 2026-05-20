# TSK-022: Intrinsic Style Layer

## 1. Objective
Add a new top-precedence cascade layer to the styling engine — `IntrinsicStyle()` — so that replaced and compound elements can declaratively force UA-mandated properties (e.g., `display: inline-block`, `overflow: clip`, `white-space: pre-wrap`) that authors cannot override.

See ADR-010.

## 2. Design & Requirements

### 2.1 Feature Design
- Add a third style accessor to `dom.Element` alongside `RawStyle()` and `DefaultStyle()`:
  - `IntrinsicStyle() style.Style` — sparse `style.Style` using `Optional[T]` fields; returns the empty style by default.
- Extend `style.Resolver` to apply layers in this order from weakest to strongest:
  1. Inherited values from parent's `Computed`.
  2. `DefaultStyle()`.
  3. `RawStyle()`.
  4. **`IntrinsicStyle()`** (new top layer).
- Internally tag the new layer with a `style.CascadeOrigin` enum value `OriginUserAgent`. The other origins (`OriginInherited`, `OriginUADefault`, `OriginAuthor`) are introduced for symmetry but not exposed to author code today.

### 2.2 Rules
- The `IntrinsicStyle` layer is **sparse** — element implementations only set the properties they want to force. Unset properties cascade normally.
- The layer participates in the existing `Optional[T]` merge logic; no new merge algorithm is needed.
- Inheritable properties set via `IntrinsicStyle` inherit to descendants normally (this is required by the UA subtree model: a `<textarea>`'s `white-space: pre-wrap` must cascade into its UA-internal text node).
- The cascade origin enum is internal to the `style` package; it must not leak into `dom`, `render`, `layout`, or `paint`.
- No render-side hard-coded `SetComputedStyle` calls for UA-forced properties — those guards now live on the element via `IntrinsicStyle()` and disappear from render code.

### 2.3 Out of Scope
- Exposing cascade origins to author tooling / devtools.
- Author-defined `!important` or other cascade markers.
- Per-property origin queries on `style.Computed`.

## 3. Implementation Steps
1. Add `IntrinsicStyle() style.Style` to the `dom.Element` interface. Provide a default implementation on the unexported base element that returns the zero `style.Style{}`.
2. Define `style.CascadeOrigin` enum with values `OriginInherited`, `OriginUADefault`, `OriginAuthor`, `OriginUserAgent`. Keep the type unexported-from-package or exported but documented as internal.
3. Modify `style.Resolver.Resolve(el, parent)`:
   - Build the layered merge in the order specified above.
   - After merging, the resolver continues to emit a `style.Computed` exactly as today; no downstream consumer needs to change.
4. Audit existing UA-mandated properties currently set on `DefaultStyle()` for replaced elements (input, textarea) and **move** any that must not be author-overridable to `IntrinsicStyle()`. (The actual moves for input/textarea happen in TSK-024/TSK-025.)
5. Add a section to `style/doc.go` describing the four-layer cascade.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] An element with `IntrinsicStyle().Display = DisplayInlineBlock` and `RawStyle().Display = DisplayBlock` resolves to `Computed.Display == DisplayInlineBlock` (intrinsic wins).
- [ ] An element with only `RawStyle().Color = red` and no intrinsic color resolves to `red` (intrinsic empty does not nullify author).
- [ ] An inherited property set via `IntrinsicStyle` on the parent (e.g., `WhiteSpace`) cascades to a child that has no explicit value for it.
- [ ] `DefaultStyle` is overridden by `RawStyle`, which is in turn overridden by `IntrinsicStyle`, verified per property.
- [ ] Elements that do not implement `IntrinsicStyle` see no behavioral change (regression guard).

### 4.2 Integration Tests
- [ ] An input-like test element forces `Display: inline-block` and `OverflowX: clip` via `IntrinsicStyle`. After setting `RawStyle().Display(Block)`, the resolved `Computed.Display` remains `inline-block` and the test verifies the resulting layout is inline.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `intrinsic_style_test.go`: a test element with intrinsic `OverflowX: clip` resists an author attempt to set `OverflowX: visible`. Render the element and confirm content outside the box is clipped in the output framebuffer.

### 4.4 Benchmarks
- [ ] `BenchmarkResolver_NoIntrinsic`: elements that return empty `IntrinsicStyle()` show < 3 % overhead vs the prior resolver.
- [ ] `BenchmarkResolver_WithIntrinsic`: elements with a populated `IntrinsicStyle()` (3–5 forced properties) show < 10 % overhead.

### 4.5 Documentation
- [ ] Update `AGENT.md`: add a rule under Styling Paradigm: "UA-mandated styles must live on the element via `IntrinsicStyle()`. They must not be hard-coded in render objects."
- [ ] Update `style/doc.go` describing the four-layer cascade with origin names.
- [ ] If `README.md` documents the styling model, mirror the addition there.
