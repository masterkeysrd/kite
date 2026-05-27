# TSK-069: Implement Low-Level DOM Form Mechanics

## Objective
Implement the foundational DOM elements and interfaces required to support forms. This includes defining a standardized way to extract values from inputs, creating a new `EventSubmit`, and building `element.Form` to implicitly handle "Enter" keystrokes and submit buttons.

## Requirements

### 1. Interfaces & Events
- Modify `dom/interfaces.go`:
  - Add `type FormControl interface { Node; Name() string; Value() any }`
- Modify `event/events.go`:
  - Add `const TypeSubmit EventType = "submit"`
  - Add `type SubmitEvent struct { BaseEvent; FormData map[string]any }`
  - Implement a constructor `NewSubmitEvent`.

### 2. Form Controls
- Update existing form elements (`element.Input`, `element.TextArea`, `element.Checkbox`, `element.Radio`, `element.Select`) to implement `dom.FormControl`. They must return their `Name` property and their current value.
- Update `element.Button`:
  - Add a `.Type(btnType string)` builder method (defaults to `"button"` or `"submit"`).

### 3. `element.Form` Component (`element/form.go`)
- Create `type Form struct { elementBase[Form] }`.
- Default to `style.DisplayBlock`.
- Add `func (f *Form) Submit()`:
  - Traverses the logical DOM subtree (`f.Children()`).
  - For every node that implements `dom.FormControl`, extracts the name and value into a `map[string]any` (skip if name is empty).
  - Dispatches `event.SubmitEvent` with the gathered map.
- **Implicit Submit Behaviors (in `initBase` or similar):**
  - Add a capture-phase listener for `event.TypeKey`. If `key == Enter` AND `event.Target()` is an `element.Input` (single-line), call `f.Submit()` and `StopPropagation()`.
  - Add a capture-phase listener for `event.TypeClick`. If `event.Target()` is an `element.Button` with `Type == "submit"`, call `f.Submit()` and `StopPropagation()`.

### 4. Kitex Wrapper (`extras/kitex/form.go`)
- `type FormProps struct { OnSubmit func(map[string]any); Children Node; ...layout }`
- Create `var Form = FC(...)` that renders `element.NewForm()` and attaches the `OnSubmit` listener to `event.TypeSubmit`.

## Testing Requirements
- Unit tests verifying `element.Form` correctly walks the tree and gathers nested `FormControl` values into the map.
- Unit tests verifying that pressing Enter on an input triggers `Submit()`.
- Unit tests verifying that clicking a `type="submit"` button triggers `Submit()`.