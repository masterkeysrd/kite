# TSK-038: Implement Checkbox and Radio Components

## Description
Create `element.Checkbox`, `element.RadioGroup`, and `element.Radio` components leveraging the UA Shadow Subtree for visual glyphs.

## Requirements

### 1. Checkbox (`element/checkbox.go`)
- Create `CheckboxElement`. It maintains a `Checked bool` state.
- **UA Shadow Subtree:** On creation, build a shadow `Box` containing a `Text` node.
- Provide props to customize the strings (Default: `[ ]` and `[X]`).
- On `Click` or `Space` key, toggle the state, update the hidden Text node, and fire `event.EventChange`.

### 2. RadioGroup & Radio (`element/radio.go`)
- **`RadioGroupElement`:**
  - Maintains `Value string`.
  - Exposes an `OnChange` listener.
- **`RadioElement`:**
  - Maintains a constant `Value string`.
  - Uses the UA Shadow Subtree to render `( )` or `(•)`.
  - On `Click` or `Space`, it does **not** toggle freely. It checks if its parent is a `RadioGroup`. If so, it calls an internal method on the parent (e.g. `parent.notifySelected(radio.Value)`).
- **Coordination:**
  - The `RadioGroup` updates its internal value and iterates over its `Children()`.
  - For each child that is a `RadioElement`, it forces them to re-evaluate their visual state based on the group's new active value.
  - Fires `event.EventChange` on the Group.

## Tests
- Mount a Checkbox, trigger a click, verify the state toggles and the shadow text updates.
- Mount a RadioGroup with 3 Radios. Click the second one. Verify the group value updates, the second radio shows `(•)`, and the others show `( )`.