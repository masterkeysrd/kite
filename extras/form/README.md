# 📝 Form

Form provides a high-level form state and validation engine for Kitex applications. It is inspired by React Hook Form and provides strongly-typed form management, automatic mapping from raw DOM map data to Go structs, and asynchronous submission handling.

## ✨ Features

- 🔒 **Strongly Typed**: Manage form state using strongly-typed Go structs.
- ⚡ **Reactive State**: Integrates seamlessly with Kitex's VDOM using `kitex.UseState`.
- 🛡️ **Validation Engine**: Built-in support for synchronous validation rules.
- 🚦 **Submission State**: Automatically tracks `IsSubmitting` state during async operations.
- 🧩 **Manual Error Handling**: Set or clear arbitrary errors manually with `SetError`.

## 🚀 Getting Started

### 1. Define your form structure

Create a Go struct that represents your form data. You can use JSON tags to map field names from the UI:

```go
type UserForm struct {
	Name     string `json:"name"`
	Age      int    `json:"age"`
	IsActive bool   `json:"is_active"`
}
```

### 2. Initialize the hook

Inside a Kitex functional component, initialize the form using `form.Use`:

```go
package main

import (
	"fmt"

	"github.com/masterkeysrd/kite/extras/form"
	"github.com/masterkeysrd/kite/extras/kitex"
)

var MyForm = kitex.SimpleFC("MyForm", func() kitex.Node {
	formAPI := form.Use(form.Options[UserForm]{
		InitialValues: UserForm{Name: "Default User", Age: 18},
		Validate: func(values UserForm) map[string]string {
			errors := make(map[string]string)
			if len(values.Name) < 3 {
				errors["name"] = "Name must be at least 3 characters"
			}
			return errors
		},
		OnSubmit: func(values UserForm) error {
			// Handle API request or state update here
			fmt.Printf("Submitting: %+v\n", values)
			return nil // returning an error populates the "root" error
		},
	})

	state := formAPI.State()

	// ... Render your UI ...
```

### 3. Connect to the UI

Use `kitex.Form` or manually wire form elements to the `HandleSubmit` function:

```go
	return kitex.Form(kitex.FormProps{
		OnSubmit: formAPI.HandleSubmit,
	},
		kitex.Box(kitex.BoxProps{},
			kitex.Input(kitex.InputProps{
				Name:  "name",
				Value: state.Values.Name,
			}),
			kitex.If(state.Errors["name"] != "", kitex.Text(state.Errors["name"])),
		),
		
		kitex.Box(kitex.BoxProps{},
			kitex.Checkbox(kitex.CheckboxProps{
				Name:    "is_active",
				Checked: state.Values.IsActive,
			}),
		),

		kitex.Button(kitex.ButtonProps{
			Type:     "submit",
			Disabled: state.IsSubmitting || !state.IsValid,
		}, kitex.IfElse(state.IsSubmitting, kitex.Text("Saving..."), kitex.Text("Save"))),

		kitex.If(state.Errors["root"] != "", kitex.Text(state.Errors["root"])),
	)
})
```

## 🛠 API Reference

### `form.Use(Options[T]) API[T]`

Initializes the form hook with configuration options and returns the API interface.

### `Options[T]`

- **`InitialValues T`**: The starting values of the form struct.
- **`Validate func(T) map[string]string`**: A callback executed before submission. Should return a map of field names to error messages. Return an empty map or `nil` if there are no errors.
- **`OnSubmit func(T) error`**: The function invoked when validation passes. It receives the fully parsed struct. Returning an error populates `Errors["root"]`.

### `API[T]`

- **`State() State[T]`**: Returns the current snapshot of the form.
- **`HandleSubmit(map[string]any)`**: An event handler that accepts a map of raw values (typically from `kitex.Form` `OnSubmit`), parses it into struct `T`, validates, and potentially invokes `OnSubmit`.
- **`SetError(key string, message string)`**: Manually set an error for a given field key. Passing an empty message `""` clears the error for that key.

### `State[T]`

- **`Values T`**: The current fully parsed values.
- **`Errors map[string]string`**: Current validation or manual errors.
- **`IsSubmitting bool`**: `true` while the `OnSubmit` function is executing.
- **`IsValid bool`**: `true` if `Errors` map is empty.
