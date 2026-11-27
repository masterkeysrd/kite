package form

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"

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
		maps.Copy(newErrors, s.Errors)
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

			// 1. Map raw map data into struct T via direct reflection decoding
			if err := decodeFormData(rawData, &values); err != nil {
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
				maps.Copy(newErrors, s.Errors)
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

func decodeFormData(rawData map[string]any, dst any) error {
	if len(rawData) == 0 {
		return nil
	}

	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer, got %T", dst)
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("destination must point to a struct, got %s", elem.Kind())
	}

	return decodeStruct(rawData, elem)
}

func decodeStruct(rawData map[string]any, target reflect.Value) error {
	t := target.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			if _, hasForm := sf.Tag.Lookup("form"); !hasForm {
				if _, hasJSON := sf.Tag.Lookup("json"); !hasJSON {
					if err := decodeStruct(rawData, target.Field(i)); err != nil {
						return err
					}
					continue
				}
			}
		}

		key, skip := resolveFieldKey(sf)
		if skip {
			continue
		}

		raw, found := lookupFieldValue(rawData, key, sf.Name)
		if !found {
			continue
		}

		displayKey := key
		if displayKey == "" {
			displayKey = sf.Name
		}

		if err := assignFieldValue(target.Field(i), displayKey, raw); err != nil {
			return err
		}
	}
	return nil
}

func resolveFieldKey(sf reflect.StructField) (key string, skip bool) {
	if tag, ok := sf.Tag.Lookup("form"); ok {
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			return "", true
		}
		if name != "" {
			return name, false
		}
	}
	if tag, ok := sf.Tag.Lookup("json"); ok {
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			return "", true
		}
		if name != "" {
			return name, false
		}
	}
	return "", false
}

func lookupFieldValue(rawData map[string]any, tagKey, fieldName string) (any, bool) {
	if tagKey != "" {
		v, ok := rawData[tagKey]
		return v, ok
	}
	if v, ok := rawData[fieldName]; ok {
		return v, true
	}
	for k, v := range rawData {
		if strings.EqualFold(k, fieldName) {
			return v, true
		}
	}
	return nil, false
}

func assignFieldValue(fieldVal reflect.Value, fieldName string, raw any) error {
	if raw == nil {
		return nil
	}

	if fieldVal.Kind() == reflect.Pointer {
		ptr := reflect.New(fieldVal.Type().Elem())
		if err := assignFieldValue(ptr.Elem(), fieldName, raw); err != nil {
			return err
		}
		fieldVal.Set(ptr)
		return nil
	}

	rv := reflect.ValueOf(raw)
	if rv.Type().AssignableTo(fieldVal.Type()) {
		fieldVal.Set(rv)
		return nil
	}

	if s, ok := raw.(string); ok {
		return assignFromString(fieldVal, fieldName, s)
	}

	if isNumericKind(rv.Kind()) && isNumericKind(fieldVal.Kind()) {
		return assignNumeric(fieldVal, fieldName, rv)
	}

	return fmt.Errorf("field %s: cannot assign %T to %s", fieldName, raw, fieldVal.Type())
}

func assignFromString(fieldVal reflect.Value, fieldName, s string) error {
	trimmed := strings.TrimSpace(s)
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(s)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return fmt.Errorf("field %s: invalid bool %q: %w", fieldName, s, err)
		}
		fieldVal.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(trimmed, 10, fieldVal.Type().Bits())
		if err != nil {
			return fmt.Errorf("field %s: invalid integer %q: %w", fieldName, s, err)
		}
		fieldVal.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(trimmed, 10, fieldVal.Type().Bits())
		if err != nil {
			return fmt.Errorf("field %s: invalid unsigned integer %q: %w", fieldName, s, err)
		}
		fieldVal.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(trimmed, fieldVal.Type().Bits())
		if err != nil {
			return fmt.Errorf("field %s: invalid float %q: %w", fieldName, s, err)
		}
		fieldVal.SetFloat(f)
		return nil
	default:
		return fmt.Errorf("field %s: cannot convert string %q to %s", fieldName, s, fieldVal.Type())
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func assignNumeric(fieldVal reflect.Value, fieldName string, rv reflect.Value) error {
	switch fieldVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var v int64
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v = rv.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			u := rv.Uint()
			v = int64(u)
		case reflect.Float32, reflect.Float64:
			v = int64(rv.Float())
		}
		if fieldVal.OverflowInt(v) {
			return fmt.Errorf("field %s: value %v overflows %s", fieldName, rv.Interface(), fieldVal.Type())
		}
		fieldVal.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var u uint64
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iv := rv.Int()
			if iv < 0 {
				return fmt.Errorf("field %s: negative value %d cannot convert to %s", fieldName, iv, fieldVal.Type())
			}
			u = uint64(iv)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			u = rv.Uint()
		case reflect.Float32, reflect.Float64:
			fv := rv.Float()
			if fv < 0 {
				return fmt.Errorf("field %s: negative value %v cannot convert to %s", fieldName, fv, fieldVal.Type())
			}
			u = uint64(fv)
		}
		if fieldVal.OverflowUint(u) {
			return fmt.Errorf("field %s: value %v overflows %s", fieldName, rv.Interface(), fieldVal.Type())
		}
		fieldVal.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		var f float64
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f = float64(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			f = float64(rv.Uint())
		case reflect.Float32, reflect.Float64:
			f = rv.Float()
		}
		if fieldVal.OverflowFloat(f) {
			return fmt.Errorf("field %s: value %v overflows %s", fieldName, rv.Interface(), fieldVal.Type())
		}
		fieldVal.SetFloat(f)
		return nil
	}
	return fmt.Errorf("field %s: cannot convert %T to %s", fieldName, rv.Interface(), fieldVal.Type())
}
