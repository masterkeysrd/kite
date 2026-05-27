# TSK-070: Implement `extras/form` High-Level API

## Objective
Implement `extras/form`, a high-level API inspired by React Hook Form. It will act as the state manager and validation engine for Kite applications, mapping raw DOM `SubmitEvent` data into strongly-typed Go structs and managing the `IsSubmitting` async lifecycle.

## Requirements

### 1. Core Types (`extras/form/form.go`)
- `type State[T any] struct { Values T; Errors map[string]string; IsSubmitting bool; IsValid bool }`
- `type Options[T any] struct { InitialValues T; Validate func(T) map[string]string; OnSubmit func(T) error }`
- `type API[T any] struct { State State[T]; HandleSubmit func(map[string]any); SetError func(string, string) }`

### 2. The `Use` Hook
- `func Use[T any](opts Options[T]) API[T]`
  - Use `kitex.UseState` to initialize and track the `State[T]`.
  - **`HandleSubmit` logic:**
    1. Accept the `map[string]any` from `kitex.Form`.
    2. Map the raw map data into the generic struct `T` (using a lightweight reflection helper or JSON unmarshaling trick). If mapping fails, set a root error.
    3. Run `opts.Validate(values)`. If errors exist, update state (`IsValid = false`, set `Errors`) and abort.
    4. Transition state: `IsSubmitting = true`, `IsValid = true`, clear `Errors`.
    5. Execute `opts.OnSubmit(values)` in a goroutine.
    6. When the goroutine finishes, use `kitex.PostMacro` to transition `IsSubmitting = false` back on the main thread. If `OnSubmit` returned an error, place it in `Errors["root"]`.

### 3. Documentation & Examples
- Add a package-level `doc.go`.
- Create an example application in `examples/form_demo/main.go`:
  - Define a complex struct (e.g., User Registration with nested fields if supported, or just multiple types like int/string/bool).
  - Use `form.Use` to define validation rules.
  - Render a `kitex.Form` passing `s.HandleSubmit`.
  - Render error messages below inputs conditionally based on `s.State.Errors`.
  - Simulate a slow network request in `OnSubmit` and disable the submit button using `s.State.IsSubmitting`.

## Testing Requirements
- Unit tests verifying the mapping from `map[string]any` to a struct `T` works for standard types (string, bool).
- Unit tests verifying the state machine: Valid submissions trigger `IsSubmitting = true` then `false`. Invalid submissions populate `Errors` and do not trigger `OnSubmit`.