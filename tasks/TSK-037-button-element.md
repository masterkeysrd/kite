# TSK-037: Implement Button Element

## Description
Create a dedicated `element.Button` component. It should act as a standard layout container while managing interaction states (Focus, Active) and translating keyboard inputs into standard `EventClick` events.

## Requirements
1. **Creation:** Add `element/button.go`. Define `type ButtonElement struct`.
2. **Layout:** It should act as a generic container. Do NOT use a UA Shadow Subtree for this; children appended to the button should be normal, public DOM children.
3. **Focus:** Must return `true` for `IsFocusable()`.
4. **Interactivity:**
   - On `event.EventMouseDown`, if the target is within the button, it should visually reflect an "active/pressed" state (via pseudo-classes or custom style updates if applicable) and request focus.
   - On `event.EventMouseUp`, fire an `EventClick`.
   - On `event.EventKey` for `Enter` or `Space` while focused, fire an `EventClick`.
5. **Styling:** Provide a sensible `IntrinsicStyle` (e.g., `Display: Flex`, `AlignItems: Center`, `JustifyContent: Center`).

## Tests
- Mount a Button. Send a spacebar KeyEvent. Assert that the Click listener triggers.
- Send a MouseDown then MouseUp. Assert that the Click listener triggers.