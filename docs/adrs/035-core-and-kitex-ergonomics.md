# ADR 035: Core & Kitex Ergonomics and Layer Boundaries

## Status
Accepted

## Context
As Kite and `extras/kitex` have matured, real-world application and example code revealed several sources of repetitive boilerplate, weak typing, and architectural friction:

1. **Verbose Style Declarations & Missing Pseudo-States (`style` / `styler`):**
   - Authoring colors requires verbose `color.RGBA{R: ..., G: ..., B: ..., A: 255}` struct literals everywhere because CSS-equivalent `#hex` and `rgb()` constructors are absent.
   - Conditional style merging breaks fluent chains into imperative `if` blocks.
   - Interactive visual states (`:hover`, `:focus`, `:active`, `:disabled`) cannot be expressed declaratively in `style.Style`. In `kitex`, authors must build stateful wrapper components (`UseState(false)` + `OnMouseEnter`/`OnMouseLeave`) just to change hover colors, triggering full VDOM re-renders for purely visual state transitions.
2. **Incomplete `element.Element` Symmetry & Untyped Event Helpers (`element`):**
   - `element.Element` exposes `RawStyle()` and `IsHidden()` getters without corresponding `SetStyle(style.Style)` and `SetHidden(bool)` mutators, forcing `extras/kitex` to maintain two 22-branch concrete type switches (`setStyle`, `setHidden`) that also prevent custom `element.Element` implementations from participating in VDOM reconciliation.
   - `elementBase[Self]` only exposes `OnEvent(event.EventType, func(event.Event))`, forcing manual type assertions (`e.(*event.KeyEvent)`) at every call site.
   - Controlled value callbacks are inconsistent: `SelectElement` and `RadioGroupElement` expose `.OnChange(func(string))`, while `InputElement`, `TextAreaElement`, and `CheckboxElement` lack typed value-change callbacks and `InputElement`/`TextAreaElement` require going through `Value() any` or `TextContent() string`.
3. **Untyped Events, Missing Controlled-Prop Parity, & Copy-Paste Drift in `extras/kitex`:**
   - Every event handler in `ElementProps` is typed as `func(event.Event)` rather than its concrete event type (`*event.KeyEvent`, `*event.MouseEvent`, `*event.WheelEvent`, `*event.FocusEvent`, `*event.ChangeEvent`, `*event.InputEvent`, `*event.ScrollEvent`).
   - `SelectProps` and `RadioGroupProps` expose `OnValueChange func(string)`, while `InputProps` and `TextAreaProps` lack `OnValueChange func(string)` and `CheckboxProps` lacks `OnCheckedChange func(bool)`.
   - Manual duplication of `ElementProps` fields across `*Props` structs caused a silent bug where `Attributes` is dropped in `elementProps()` for `CheckboxProps`, `RadioGroupProps`, `RadioProps`, `SelectProps`, `OptionProps`, `InputProps`, and `TextAreaProps`.
   - Formatted text requires `kitex.Text(fmt.Sprintf(...))`, and `UseState` lacks an atomic functional updater (`func(T) T`) for goroutine-safe read-modify-write transitions.

At the same time, we must enforce **strict architectural boundaries** so opinionated UI components (`VStack`, `HStack`, `Spacer`, `Card`) or Tailwind-like style shortcuts (`Row()`, `Col()`, `.Center()`) never pollute Core or `extras/kitex`.

## Decision

### 1. Strict Four-Layer Boundary Policy
Every API addition is governed by a strict four-layer placement rule:

1. **Layer 1 — Kite Core (`dom`, `style`, `element`, `event`, `engine`):** Models W3C/CSS-equivalent platform primitives and cascade rules; never imports `backend/uv` or `extras/kitex`, and never encodes design-system opinions.
2. **Layer 2 — `extras/kitex`:** Models `react` + `react-dom` (`ADR-024`); maintains **strict 1:1 parity** with `element.*` primitives, typed DOM events, and core reconciler/hook primitives. Synthetic layout components (`VStack`, `HStack`, `Spacer`, `Label`) are explicitly excluded.
3. **Layer 3 — `extras/<pkg>` (`extras/trail`, `extras/flight`, `extras/wind`, etc.):** Opt-in runtime and ecosystem integrations that layer cleanly without coupling lower layers.
4. **Layer 4 — External / Userland Packages:** Opinionated component libraries (`VStack`, `HStack`, `Card`, `Badge`), style utility presets (`Row()`, `Col()`, `.Center()`, color themes), and pure composite hooks (`UseDebounce`, `UseToggle`).

### 2. Layer 1: Style Value Constructors, Conditional Merging, & Pseudo-State Cascade
- **Color Constructors (`style`):** Add `style.RGB(r, g, b uint8) color.Color`, `style.RGBA(r, g, b, a uint8) color.Color`, `style.Hex(hex string) color.Color` (supporting `#RGB`, `#RRGGBB`, `#RRGGBBAA`, with optional leading `#`), and `style.HexInt(v uint32) color.Color` (`0xRRGGBB`).
- **Conditional Composition (`style`):** Add `(s Style) When(cond bool, other Style) Style` returning `s.Merge(other)` when `cond` is true and `s` unchanged otherwise.
- **Pseudo-State Cascade (`style`, `internal/dom`, `internal/styler`, `engine`):**
  - Introduce `style.PseudoState` bitmask (`PseudoNone`, `PseudoHover`, `PseudoFocus`, `PseudoActive`, `PseudoDisabled`) and `PseudoRule{State PseudoState, Style Style}` stored on `style.Style` symmetrically to `MediaRule` (`Hover(Style)`, `Focus(Style)`, `Active(Style)`, `Disabled(Style)`, `HasPseudoRules() bool`, `EvaluatePseudo(PseudoState) Style`).
  - Track transient interaction flags (`Hovered`, `Focused`, `Active`) on the logical DOM element (`internal/dom`) and mark style dirty (`MarkStyleDirty`) in `engine` when hover (`EventMouseEnter`/`EventMouseLeave`), focus (`focus.Manager` transitions), active (`EventMouseDown`/`EventMouseUp`), or disabled (`SetDisabled`) state changes.
  - `styler.Resolver.Resolve` evaluates `authorStyle.EvaluateMedia(viewportSize).EvaluatePseudo(pseudoState)` at Layer 4 (`OriginAuthor`), enabling zero-VDOM-rerender interactive styling across both Core and Kitex.

### 3. Layer 1: `element.Element` Symmetry, Typed Event Helpers, & Control Parity
- **Interface Symmetry:** Add `SetStyle(style.Style)` and `SetHidden(bool)` to `element.Element` (implemented on `elementBase[Self]`), allowing generic callers to update style and visibility without concrete type switches. Ensure `SetDisabled(bool)` on disableable elements marks style dirty so `:disabled` pseudo-rules re-evaluate.
- **Typed Fluent Event Helpers on `elementBase[Self]`:**
  - `OnClick(func(*event.MouseEvent)) *Self`
  - `OnPress(func()) *Self` (fires on primary click; since `ButtonElement` synthesizes `EventClick` on `Enter`/`Space`, `OnPress` handles both mouse and keyboard activation on buttons)
  - `OnKeyDown(func(*event.KeyEvent)) *Self`, `OnKeyUp(func(*event.KeyEvent)) *Self`, `OnKeyPress(func(*event.KeyEvent)) *Self`
  - `OnKey(combo string, fn func(*event.KeyEvent)) *Self` (matches `ke.MatchString(combo)` on `keydown`)
  - `OnMouseDown(func(*event.MouseEvent)) *Self`, `OnMouseUp(func(*event.MouseEvent)) *Self`, `OnMouseEnter(func(*event.MouseEvent)) *Self`, `OnMouseLeave(func(*event.MouseEvent)) *Self`
  - `OnWheel(func(*event.WheelEvent)) *Self` (named `OnWheelEvent` if concrete types already implement `event.Scrollable`'s `OnWheel(*event.WheelEvent)`)
  - `OnFocus(func(*event.FocusEvent)) *Self`, `OnBlur(func(*event.FocusEvent)) *Self`
- **Form Control Parity:**
  - Add `Text() string` and `OnChange(fn func(string)) *InputElement` to `InputElement` and `TextAreaElement` (invoked on user-initiated buffer edits alongside `event.NewInput`, never on programmatic `SetValue`).
  - Add `OnChange(fn func(bool)) *CheckboxElement` to `CheckboxElement` (invoked on user toggle, never on programmatic `SetChecked`).

### 4. Layer 2: `extras/kitex` Typed Events, Controlled Props, `Textf`, & State Updaters
- **Typed Intrinsic Event Props (Breaking Change):** Update `ElementProps` and all 1:1 element `*Props` structs to use concrete event pointers:
  - `OnKeyDown`, `OnKeyUp`, `OnKeyPress`: `func(*event.KeyEvent)`
  - `OnMouseDown`, `OnMouseUp`, `OnMouseMove`, `OnClick`, `OnDrag`, `OnMouseEnter`, `OnMouseLeave`, `OnMouseOver`, `OnMouseOut`: `func(*event.MouseEvent)`
  - `OnWheel`: `func(*event.WheelEvent)`
  - `OnFocus`, `OnBlur`: `func(*event.FocusEvent)`
  - `OnChange`: `func(*event.ChangeEvent)`
  - `OnInput`: `func(*event.InputEvent)`
  - `OnScroll`: `func(*event.ScrollEvent)`
  - Wire listeners via a generic `updateTypedListener[E event.Event]` helper that preserves `funcEquals` pointer comparison without reflection-based `isNilFunc`.
- **1:1 Controlled-Prop Parity & Bug Fixes:**
  - Add `OnValueChange func(string)` to `InputProps` and `TextAreaProps` (delegating to `InputElement.OnChange` / `TextAreaElement.OnChange`).
  - Add `OnCheckedChange func(bool)` to `CheckboxProps` (delegating to `CheckboxElement.OnChange`).
  - Add `OnPress func()` to `ButtonProps` for zero-arg button actions.
  - Fix all `*Props.elementProps()` converters so `Attributes: p.Attributes` and `OnInput: p.OnInput` are forwarded consistently, and replace `setStyle`/`setHidden` 22-case switches with `el.SetStyle(new.Style)` / `el.SetHidden(new.Hidden)`.
- **Text & Hook Primitives:**
  - Add `kitex.Textf(format string, args ...any) Node` backed by `textNodePool`.
  - Update `kitex.UseState[T any](initial T) (func() T, StateSetter[T])` where `type StateSetter[T any] func(T)` supports both direct replacement calls (`set(val)`) and atomic functional updates via `set.Update(func(T) T)` executing the transformation atomically under `hookState[T].mu` before calling `MarkDirty()`, while remaining directly assignable to `func(T)` callbacks.
- **Internal File Modularization:** Split monolithic `extras/kitex/kitex.go` into focused files (`node.go`, `memo.go`, `listeners.go`, `elements.go`, `component.go`) without changing package boundaries.

## Consequences

### Positive
- **Dramatically Cleaner Call Sites:** Eliminates `color.RGBA` verbosity, `fmt.Sprintf` text wrappers, manual `*event.KeyEvent` / `*event.InputEvent` type assertions, and stateful `HoverButton` VDOM wrappers.
- **Zero-Allocation / Zero-VDOM-Rerender Interactive States:** `:hover`, `:focus`, `:active`, and `:disabled` resolve directly in the C-speed style cascade without waking up the VDOM reconciler.
- **Strict Layer Purity:** `engine` and `extras/kitex` remain completely decoupled from `backend/uv`, and neither Core nor `extras/kitex` accumulates opinionated design-system components (`VStack`, `HStack`, `Card`).
- **Extensibility & Bug Elimination:** Custom `element.Element` types automatically work with `extras/kitex` via `SetStyle`/`SetHidden`, and dropped `Attributes` bugs in `*Props` are eliminated.

### Negative / Trade-offs
- **Breaking Event Signatures in `extras/kitex`:** Existing `ElementProps` listeners typed as `func(event.Event)` must update to concrete event pointers (`func(*event.MouseEvent)`, `func(*event.KeyEvent)`, etc.), or use `OnPress: func()` on buttons. All internal packages (`extras/*`), tests, and `examples/*` are migrated as part of this rollout.
