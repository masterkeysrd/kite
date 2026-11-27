# Task: TSK-080 — Kitex Typed Events, Controlled-Prop Parity, `Textf` & State Updaters

**ADR:** 035-core-and-kitex-ergonomics
**Depends on:** TSK-079

## 1. Objective
Upgrade `extras/kitex` to strongly typed event callbacks, add 1:1 controlled-prop parity (`OnValueChange`, `OnCheckedChange`, `OnPress`), fix dropped `Attributes` bugs, replace concrete element type switches with `element.Element` interface calls, add `kitex.Textf` and atomic functional state updaters, and split the monolithic `extras/kitex/kitex.go` into focused files.

## 2. Design & Requirements
- **Typed Intrinsic Event Callbacks (Breaking Change):**
  - Update `ElementProps` and all element `*Props` structs (`ButtonProps`, `CheckboxProps`, `RadioGroupProps`, `RadioProps`, `SelectProps`, `OptionProps`, `InputProps`, `TextAreaProps`, `TDProps`, `OverlayProps`, `DialogProps`, etc.) to concrete event pointer types:
    - `OnKeyDown`, `OnKeyUp`, `OnKeyPress`: `func(*event.KeyEvent)`
    - `OnMouseDown`, `OnMouseUp`, `OnMouseMove`, `OnClick`, `OnDrag`, `OnMouseEnter`, `OnMouseLeave`, `OnMouseOver`, `OnMouseOut`: `func(*event.MouseEvent)`
    - `OnWheel`: `func(*event.WheelEvent)`
    - `OnFocus`, `OnBlur`: `func(*event.FocusEvent)`
    - `OnChange`: `func(*event.ChangeEvent)`
    - `OnInput`: `func(*event.InputEvent)` (newly exposed on `ElementProps` and forwarded in all `elementProps()`)
    - `OnScroll`: `func(*event.ScrollEvent)`
  - Implement generic `updateTypedListener[E event.Event](el element.Element, typ event.EventType, oldFn, newFn func(E))` that checks `funcEquals(oldFn, newFn)` before re-subscribing so stable closures never churn subscriptions.
- **Controlled-Prop Parity & Bug Fixes:**
  - Add `OnValueChange func(string)` to `InputProps` and `TextAreaProps` (wired via `inp.OnChange(new.OnValueChange)` / `txa.OnChange(new.OnValueChange)`).
  - Add `OnCheckedChange func(bool)` to `CheckboxProps` (wired via `cb.OnChange(new.OnCheckedChange)`).
  - Add `OnPress func()` to `ButtonProps` (fires on button click/activation without colliding with `OnClick` if both are provided).
  - Fix every `*Props.elementProps()` method so `Attributes: p.Attributes` and `OnInput: p.OnInput` are never dropped.
  - Replace the 22-branch `setStyle` and `setHidden` switches with `el.SetStyle(new.Style)` and `el.SetHidden(new.Hidden)` (and `setDisabled` with `dom.Disableable` / `SetDisabled(bool)` interface check).
- **`kitex.Textf` & Atomic State Updaters:**
  - Add `func Textf(format string, args ...any) Node` returning `Text(fmt.Sprintf(format, args...))` (adjusting `trackSource` skip depth so DevTools source tracking points to the caller of `Textf`).
  - In `extras/kitex/hooks.go`:
    - Define `type StateSetter[T any] func(T)` with `func (s StateSetter[T]) Update(fn func(T) T)` that looks up the hook state via a zero-allocation closure-pointer registry (`setterRegistry`) and executes `hs.value = fn(hs.value)` atomically under `hs.mu` before calling `activeNode.MarkDirty()`, with deterministic cleanup on `ComponentNode.Destroy()`.
    - Update `UseState[T any](initial T) (func() T, StateSetter[T])` to return `StateSetter[T]`, which remains directly assignable to `func(T)` fields with stable `funcEquals` identity.
- **Modularize `extras/kitex` & Migrate Dependent Packages:**
  - Split `extras/kitex/kitex.go` (2,831 lines) into cohesive files within package `kitex`: `node.go`, `memo.go`, `listeners.go`, `elements.go`, and `component.go`.
  - Update all affected callers in `extras/` (`extras/stage`, `extras/trail`, `extras/flight`, `extras/form`, `extras/wind`, `extras/kites`) and tests to match the new typed event signatures.
  - Do **not** add synthetic layout components (`VStack`, `HStack`, `Spacer`, `Label`) to `extras/kitex`.

## 3. Implementation Steps
1. Split `extras/kitex/kitex.go` into `node.go`, `memo.go`, `listeners.go`, `elements.go`, and `component.go`.
2. Implement `updateTypedListener`, update `ElementProps` and all `*Props` structs, fix `Attributes`/`OnInput` forwarding, and simplify `setStyle`/`setHidden`/`setDisabled`.
3. Wire `InputProps.OnValueChange`, `TextAreaProps.OnValueChange`, `CheckboxProps.OnCheckedChange`, `ButtonProps.OnPress`, and `kitex.Textf`.
4. Implement `StateSetter[T]` and update `UseState` in `extras/kitex/hooks.go` (plus cleanup in `extras/kitex/component.go`).
5. Update all `extras/*` packages and unit tests to compile and pass with typed event signatures.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Update and expand `extras/kitex/kitex_test.go`, `reconciler_test.go`, and `hooks_test.go` to cover typed event callbacks, `Attributes` forwarding on all form controls, `OnValueChange` (`Input`, `TextArea`), `OnCheckedChange` (`Checkbox`), `OnPress` (`Button`), `Textf`, and concurrent goroutine `Update(func(T) T)` safety on `StateSetter[T]` (`UseState`).
