# TSK-035: Deferred Scroll & Cursor Rendering for Text Controls

## Description
Remove synchronous layout/scroll evaluation from text control event handlers (e.g. `<textarea>`, `<input>`) to prevent lag during rapid events, relying instead on a deferred lifecycle hook.

## Requirements
1. **Remove Synchronous Math**: In `element/text_control.go`, the `ScrollCursorIntoView()` function currently reads absolute layout boundaries and modifies scroll offsets. Ensure this function is *not* called directly from `OnWheel`, `OnKey`, or pointer events.
2. **Flag Setting**: When events dictate that the cursor moved, set `txa.needsScrollIntoView = true`.
3. **Engine Phase Integration**: 
   - Integrate a deferred hook in the `engine.Frame` pipeline.
   - We already have an "Auto-scroll phase" (Step 5b) in `engine/engine.go:536`.
   - Update `element/text_control.go` so that it safely evaluates its bounds during this phase (via `el.ScrollCursorIntoView()`).
   - Ensure `NeedsScrollIntoView` logic does not trigger `DirtyLayout`, only `DirtyScroll`.
4. **Scroll Integrity**: Validate that regular wheel scrolling does not snap back erratically. (The `TestTextArea_WheelScroll_DoesNotSnapBack` test exists and must remain passing).

## Tests
- Add/update tests in `element/text_control_test.go` to ensure that mutating text or moving the cursor does not recalculate scrolling immediately, but correctly updates `Scroll()` offsets after `engine.Frame()` is invoked.

## Documentation
- Mention in `element/doc.go` that cursor positioning layout math is deferred to the engine frame phase.