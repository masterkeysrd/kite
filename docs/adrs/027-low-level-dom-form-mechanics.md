# ADR 027: Low-Level DOM Form Mechanics

## Status

Accepted

## Context

To support interactive application forms with multiple controls (Inputs, TextAreas, Checkboxes, Radios, Selects), Kite needs a standard way to manage forms, collect control values, and submit the collected data. 

In standard HTML/DOM, forms are native elements (`<form>`) that wrap form controls, handle key/click event interception (e.g. Enter on text inputs or clicking a `type="submit"` button), aggregate form data, and dispatch a submit event. We need a similar architecture in Kite that:
1. Provides a unified `dom.FormControl` interface to generalize data extraction.
2. Implements a dedicated `Form` element in the low-level DOM (`element.FormElement`) that aggregates form controls under its logical subtree.
3. Automatically handles implicit submission rules (Enter keystrokes on single-line inputs, and click events on submit buttons).
4. Normalizes form submission via a semantic `SubmitEvent` carrying the aggregated form data.
5. Integrates with the `kitex` VDOM package using a declarative `kitex.Form` wrapper.

## Decision

We will implement low-level DOM Form Mechanics with the following design:

### 1. Unified Interface and Event Properties
We introduce the `dom.FormControl` interface in the `dom` package:
```go
type FormControl interface {
    Node
    Name() string
    Value() any
}
```
All form control elements (`InputElement`, `TextAreaElement`, `CheckboxElement`, `RadioElement`, and `SelectElement`) are updated to implement `dom.FormControl`. 

A new event type `event.TypeSubmit = "submit"` and a specialized `SubmitEvent` structure are introduced:
```go
type SubmitEvent struct {
    BaseEvent
    FormData map[string]any
}
```

### 2. Low-Level `element.FormElement`
A dedicated `FormElement` is added to the `element` package:
- It defaults to `style.DisplayBlock`.
- It exposes a `Submit()` method that traverses the logical DOM subtree recursively:
  - If a child node implements `dom.FormControl`, its name and value are extracted.
  - Sibling radio buttons are filtered so that only the checked radio button's value is submitted.
  - Values from controls with empty names are ignored.
  - The collected key-value pairs are dispatched via a `SubmitEvent` to listeners on the Form element.

### 3. Implicit Submit Behaviors
The `FormElement` registers capture-phase event listeners to handle implicit submission:
- **Enter Key:** A capture-phase listener for `event.TypeKeyDown` checks if the target is a single-line input field (`element.InputElement`). If so, it calls `f.Submit()` and calls `StopPropagation()` to prevent the Enter key from causing unintended behaviors.
- **Submit Button:** A capture-phase listener for `event.TypeClick` checks if the target is a button with `Type() == "submit"`. If so, it calls `f.Submit()` and calls `StopPropagation()`.

### 4. Kitex VDOM Wrapper (`extras/kitex/form.go`)
A declarative VDOM wrapper `kitex.Form` is implemented using `kitex.FCC`:
- It accepts `FormProps` which includes standard element properties and an `OnSubmit func(map[string]any)` handler.
- It instantiates a real `element.FormElement` and hooks up the `event.EventSubmit` event listener, forwarding the aggregated data payload to `OnSubmit`.

### 5. Dynamic Child Option Synchronization (Select Element)
To ensure option elements inside `SelectElement` are synchronized dynamically (specifically during Virtual DOM reconciliation where properties are updated on the host *before* child nodes are reconciled and appended):
- `SelectElement` overrides `AppendChild(child)` and `RemoveChild(child)` to detect when child `OptionElement` nodes are added to or removed from the DOM tree.
- On child mutation, it automatically updates its internal options list (`s.options`) and triggers the selected value display synchronization (`s.syncValue()`).
- This guarantees correct initial rendering of the selected option name on the select button without requiring a manual re-render pass.

## Consequences

### Positive
- **Web-aligned paradigms:** Developers can build standard forms with standard submission triggers (Enter to submit, submit buttons).
- **Type-safe extraction:** Sibling radio controls and dropdowns are naturally supported, and empty fields do not pollute the form payload.
- **Loose coupling:** Form submission is decoupled from layout and visual rendering; the form only needs logical subtree access.
- **O(1) Option Metadata Caching:** By overriding `AppendChild` and `RemoveChild`, the select trigger button and dropdown list maintain a fast, synchronized cache of option metadata, avoiding the need to traverse the DOM tree on every click/render.

### Negative / Trade-offs
- **Tree Traversal Cost:** The tree-walking implementation runs in $O(N)$ where $N$ is the size of the Form subtree. However, form subtrees in terminals are small enough that this traversal overhead is negligible.
- **Single-line input assumption:** The implicit Enter submit behavior is hardcoded to target `element.InputElement` and explicitly bypasses multiline `element.TextAreaElement`.
