# Task: TSK-079 — Core `element` Interface Symmetry, Typed Events & Control Parity

**ADR:** 035-core-and-kitex-ergonomics
**Depends on:** TSK-078

## 1. Objective
Complete symmetry on `element.Element` (`SetStyle`, `SetHidden`), ensure `SetDisabled` marks style dirty for `:disabled` pseudo-rules, add typed fluent event listeners on `elementBase[Self]`, and standardize `OnChange` / `Text()` on `InputElement`, `TextAreaElement`, and `CheckboxElement`.

## 2. Design & Requirements
- **`element.Element` Symmetry (`element/element.go`):**
  - Add `SetStyle(s style.Style)` and `SetHidden(v bool)` to `element.Element` and implement them on `elementBase[Self]` (delegating to the same dirty-marking logic as `.Style(s)` and `.Hidden(v)`).
- **Disabled Dirty-Marking:**
  - Ensure `SetDisabled(bool)` and `Disabled(bool)` on `ButtonElement`, `CheckboxElement`, `RadioElement`, `RadioGroupElement`, `SelectElement`, `OptionElement`, `InputElement`, and `TextAreaElement` call `MarkStyleDirty()` (and `MarkNeedsSync()` where needed) when the disabled state actually changes.
- **Typed Fluent Event Listeners (`element/element.go`):**
  - Add fluent methods on `elementBase[Self]` returning `*Self`:
    - `OnClick(fn func(*event.MouseEvent)) *Self`
    - `OnPress(fn func()) *Self` (fires on `event.EventClick`; works for both mouse click and `Enter`/`Space` activation on buttons)
    - `OnMouseDown(fn func(*event.MouseEvent)) *Self`, `OnMouseUp(fn func(*event.MouseEvent)) *Self`, `OnMouseMove(fn func(*event.MouseEvent)) *Self`, `OnMouseEnter(fn func(*event.MouseEvent)) *Self`, `OnMouseLeave(fn func(*event.MouseEvent)) *Self`
    - `OnKeyDown(fn func(*event.KeyEvent)) *Self`, `OnKeyUp(fn func(*event.KeyEvent)) *Self`, `OnKeyPress(fn func(*event.KeyEvent)) *Self`
    - `OnKey(combo string, fn func(*event.KeyEvent)) *Self` (registers `EventKeyDown` listener that calls `fn(ke)` when `ke.MatchString(combo)` is true)
    - `OnFocus(fn func(*event.FocusEvent)) *Self`, `OnBlur(fn func(*event.FocusEvent)) *Self`
    - `OnWheelEvent(fn func(*event.WheelEvent)) *Self` (named `OnWheelEvent` to avoid colliding with `event.Scrollable`'s `OnWheel(*event.WheelEvent)` method on `InputElement`, `TextAreaElement`, and `DialogElement`).
  - Nil handlers must be safely ignored without panicking.
- **Form Control Parity (`element/input.go`, `element/textarea.go`, `element/text_control.go`, `element/checkbox.go`):**
  - Add `Text() string` to `InputElement` and `TextAreaElement` returning `buf.Value()`.
  - Add `OnChange(fn func(string)) *InputElement` to `InputElement` and `OnChange(fn func(string)) *TextAreaElement` to `TextAreaElement`, invoked on user-initiated buffer edits (keystroke insert/delete, paste, cut — alongside `event.NewInput`), **never** on programmatic `SetValue`.
  - Add `OnChange(fn func(bool)) *CheckboxElement` to `CheckboxElement`, invoked on user toggle (click / Space), **never** on programmatic `SetChecked`.

## 3. Implementation Steps
1. Extend `element.Element` and `elementBase[Self]` in `element/element.go` with `SetStyle`, `SetHidden`, and typed fluent event helpers.
2. Audit all concrete elements' `SetDisabled` / `Disabled` methods to mark style dirty when changed, and ensure compile-time `var _ Element = (*XxxElement)(nil)` assertions remain satisfied.
3. Add `Text()` and `OnChange(func(string))` to `InputElement` and `TextAreaElement` via `TextControlBase`.
4. Add `OnChange(func(bool))` to `CheckboxElement`.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test `SetStyle`, `SetHidden`, `OnClick`, `OnPress`, `OnKeyDown`, `OnKey`, `OnFocus`, `OnBlur`, `OnWheelEvent`, and nil-handler safety in `element/element_test.go`.
- [ ] Test `InputElement.Text()`, `InputElement.OnChange`, `TextAreaElement.Text()`, `TextAreaElement.OnChange`, and `CheckboxElement.OnChange` (verifying user edits fire `OnChange` while programmatic `SetValue`/`SetChecked` do not).
- [ ] Test `:disabled` style invalidation when toggling `SetDisabled(true)` / `SetDisabled(false)`.
