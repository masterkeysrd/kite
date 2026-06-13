package form

import (
	"encoding/json"

	"github.com/masterkeysrd/kite/extras/kitex"
)

// State stores the form values, errors map, and submission/validity flags.
type State[T any] struct {
	Values       T
	Errors       map[string]string
	IsSubmitting bool
	IsValid      bool
}

// Options configures the form hook.
type Options[T any] struct {
	InitialValues T
	Validate      func(T) map[string]string
}

// API provides methods to interact with the form state.
type API[T any] struct {
	State        func() State[T]
	HandleSubmit func(onSubmit func(T) error) func(map[string]any)
	SetError     func(string, string)
}

// Use initializes and returns a form API for the given type T.
func Use[T any](opts Options[T]) API[T] {
	getState, setState := kitex.UseState(State[T]{
		Values:  opts.InitialValues,
		Errors:  make(map[string]string),
		IsValid: true,
	})

	setError := func(key, message string) {
		s := getState()
		// Clone errors map
		newErrors := make(map[string]string)
		for k, v := range s.Errors {
			newErrors[k] = v
		}
		if message == "" {
			delete(newErrors, key)
		} else {
			newErrors[key] = message
		}
		s.Errors = newErrors
		s.IsValid = len(s.Errors) == 0
		setState(s)
	}

	handleSubmit := func(onSubmit func(T) error) func(map[string]any) {
		return func(rawData map[string]any) {
			var values T

			// 1. Map raw map data into struct T using JSON trick
			data, err := json.Marshal(rawData)
			if err == nil {
				err = json.Unmarshal(data, &values)
			}

			if err != nil {
				setError("root", "Form data mapping failed: "+err.Error())
				return
			}

			// 2. Run validation
			var errors map[string]string
			if opts.Validate != nil {
				errors = opts.Validate(values)
			}

			if len(errors) > 0 {
				s := getState()
				s.Values = values
				s.Errors = errors
				s.IsValid = false
				setState(s)
				return
			}

			// 3. Transition to submitting state
			s := getState()
			s.Values = values
			s.Errors = make(map[string]string)
			s.IsSubmitting = true
			s.IsValid = true
			setState(s)

			// 4. Run OnSubmit callback
			var submitErr error
			if onSubmit != nil {
				submitErr = onSubmit(values)
			}

			// 5. Update state after completion
			s = getState()
			s.IsSubmitting = false
			if submitErr != nil {
				// Clone errors again just in case
				newErrors := make(map[string]string)
				for k, v := range s.Errors {
					newErrors[k] = v
				}
				newErrors["root"] = submitErr.Error()
				s.Errors = newErrors
				s.IsValid = false
			}
			setState(s)
		}
	}

	return API[T]{
		State:        getState,
		HandleSubmit: handleSubmit,
		SetError:     setError,
	}
}
