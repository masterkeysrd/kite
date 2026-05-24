# Task: System Clipboard Integration

## Description
Wire up standard OS clipboard integration (`Copy` and `Paste`) leveraging the existing `event.ClipboardBridge` on the Synthesizer.

## Requirements
- Hook into the `event.TypeCopy` standard event (already fired by the synthesizer when `Ctrl+C` or backend equivalents occur).
- When a `Copy` event is received at the `window`/`document` root:
  - Call `doc.Selection().String()`.
  - If the string is non-empty, inject it into the event's `ClipboardData` and write it to the OS via the synthesizer's backend bridge.
- (Optional but recommended) Hook `Paste` events to insert text at the caret if an input is focused, or just ensure the `event.ClipboardData` is populated correctly for user-land handlers to consume.

## Tests
- In `events_test.go`, dispatch a synthesized `Copy` event and verify the mock `ClipboardBridge` receives the exact string from the current `dom.Selection`.