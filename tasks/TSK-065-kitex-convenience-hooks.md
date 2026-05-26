# Task: TSK-065 — Kitex Terminal Convenience Hooks

**ADR:** 026-kitex-hooks-expansion
**Depends on:** TSK-062 (UseEffect/UseEffectCleanup)

### Summary
Implement terminal-specific convenience hooks `UseFocus` and `UseKeyboard` in a new `extras/kitex/hooks` sub-package. These are userland hooks built on top of the core `UseEffect`/`UseEffectCleanup` primitives, demonstrating composability.

### Files to Create

#### [NEW] `extras/kitex/hooks/hooks.go`

Package declaration and imports.

#### [NEW] `extras/kitex/hooks/focus.go`

**`UseFocus(ref kitex.Ref[dom.Element]) bool`**

Returns whether the referenced DOM element currently has focus.

Implementation:
1. Uses `kitex.UseState[bool](false)` to track focus state.
2. Uses `kitex.UseEffectCleanup` with `deps = []any{ref.Current}` to:
   - If `ref.Current` is nil, return nil (no cleanup).
   - Add `event.EventFocus` listener on `ref.Current` → sets state to `true`.
   - Add `event.EventBlur` listener on `ref.Current` → sets state to `false`.
   - Return cleanup function that removes both listeners.
3. Returns the current focus state boolean.

The hook re-subscribes when the ref's element changes (captured in deps).

#### [NEW] `extras/kitex/hooks/keyboard.go`

**`UseKeyboard(handler func(event.KeyEvent), deps []any)`**

Registers a scoped keyboard handler that listens for `event.EventKeyPress` on the document.

Implementation:
1. Uses `kitex.UseEffectCleanup` with the user-provided `deps`.
2. Effect function:
   - Adds `event.EventKeyPress` listener to the document (obtained from the component's real DOM node via `ref.Current.OwnerDocument()`).
   - The listener casts the generic `event.Event` to `event.KeyEvent` and calls `handler`.
   - Returns cleanup function that removes the listener.

**Note:** The `handler` itself is NOT included in deps — the caller controls when the handler identity changes via `UseCallback` if needed.

### Required Unit Tests

#### File: `extras/kitex/hooks/focus_test.go`

1. `TestUseFocus_InitiallyFalse` — verify returns false when element is not focused.
2. `TestUseFocus_TrueOnFocus` — verify returns true after focus event fires.
3. `TestUseFocus_FalseOnBlur` — verify returns false after blur event fires.
4. `TestUseFocus_NilRef` — verify gracefully handles nil ref without panic.
5. `TestUseFocus_RefChange` — verify re-subscribes when ref element changes.

#### File: `extras/kitex/hooks/keyboard_test.go`

1. `TestUseKeyboard_HandlerCalled` — verify handler is called on key event.
2. `TestUseKeyboard_Cleanup` — verify handler is removed on unmount.
3. `TestUseKeyboard_DepsChange` — verify handler re-subscribes when deps change.

### Test Cases
- Mount component with `UseFocus(ref)` → returns false. Simulate focus event → returns true. Simulate blur → returns false.
- Mount component with `UseKeyboard(handler, nil)` → simulate key press → handler invoked with correct key event.
- Unmount component → verify event listener is removed from document.

### Acceptance Criteria
- Both hooks are in a separate `extras/kitex/hooks` sub-package.
- Both hooks are built entirely on top of core kitex primitives (`UseState`, `UseEffectCleanup`).
- No modifications to core `kitex` package or `engine` package.
- Hooks handle nil refs and edge cases gracefully.
- All tests pass.

### Documentation Updates
- Update `AGENT.md` to document the convenience hooks sub-package.
- Add example in `examples/` showing `UseFocus` for visual focus indicators and `UseKeyboard` for keyboard shortcuts.
