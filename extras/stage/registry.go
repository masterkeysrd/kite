package stage

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/masterkeysrd/kite/extras/kitex"
)

// EmptyProps is a default props type with no fields. Use it when registering
// components that do not have any controls/knobs.
type EmptyProps struct{}

// ControlOverride allows customizing or overriding the auto-generated TUI controls.
type ControlOverride struct {
	Label       string
	Min, Max    int
	Step        int
	Options     []string
	Description string
	Hidden      bool
}

// SceneConfig defines a scene variation with custom override props.
type SceneConfig[P any] struct {
	Name  string
	Props P
}

// ComponentConfig specifies the configuration for a reflection-based component registration.
type ComponentConfig[P any] struct {
	Name         string
	DefaultProps P
	Render       func(c *Context, props P) kitex.Node
	Controls     map[string]ControlOverride
	Scenes       []SceneConfig[P]
	Description  string
}

// Registry stores registered components and their scenes.
type Registry struct {
	components map[string][]Scene
}

// NewRegistry creates a new component registry.
func NewRegistry() *Registry {
	return &Registry{
		components: make(map[string][]Scene),
	}
}

// RegisterRaw registers a list of raw scenes directly.
func (r *Registry) RegisterRaw(component string, scenes []Scene) {
	r.components[component] = append(r.components[component], scenes...)
}

// fieldMetadata holds information about a struct field parsed for control generation.
type fieldMetadata struct {
	FieldName     string
	ControlName   string
	FieldType     reflect.Type
	DefaultString string
	DefaultBool   bool
	DefaultInt    int
	Min, Max, Step int
	Options       []string
	Hidden        bool
}

// Register registers a component in the registry using reflection-based control generation.
func Register[P any](reg *Registry, config ComponentConfig[P]) {
	metadata := buildFieldMetadata(config.DefaultProps, config.Controls)

	// Build the default scene
	scenes := []Scene{
		{
			Name: "Default",
			Render: func(c *Context) kitex.Node {
				props := populateProps(c, config.DefaultProps, metadata)
				return config.Render(c, props)
			},
		},
	}

	// Build other variations
	for _, sc := range config.Scenes {
		capturedProps := sc.Props
		scenes = append(scenes, Scene{
			Name: sc.Name,
			Render: func(c *Context) kitex.Node {
				props := populateProps(c, capturedProps, metadata)
				return config.Render(c, props)
			},
		})
	}

	reg.RegisterRaw(config.Name, scenes)
}

// buildFieldMetadata parses the fields of P using reflection and merges them with overrides.
func buildFieldMetadata(defaultProps any, overrides map[string]ControlOverride) []fieldMetadata {
	t := reflect.TypeOf(defaultProps)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	v := reflect.ValueOf(defaultProps)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	var list []fieldMetadata

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		meta := fieldMetadata{
			FieldName:   field.Name,
			ControlName: field.Name,
			FieldType:   field.Type,
		}

		// 1. Parse struct tags (fallback defaults)
		tag := field.Tag.Get("stage")
		if tag != "" {
			parts := strings.Split(tag, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				kv := strings.SplitN(part, ":", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])

				switch key {
				case "label":
					meta.ControlName = val
				case "default":
					switch field.Type.Kind() {
					case reflect.String:
						meta.DefaultString = val
					case reflect.Bool:
						b, _ := strconv.ParseBool(val)
						meta.DefaultBool = b
					case reflect.Int:
						n, _ := strconv.Atoi(val)
						meta.DefaultInt = n
					}
				case "min":
					n, _ := strconv.Atoi(val)
					meta.Min = n
				case "max":
					n, _ := strconv.Atoi(val)
					meta.Max = n
				case "step":
					n, _ := strconv.Atoi(val)
					meta.Step = n
				case "options", "select":
					var opts []string
					for _, opt := range strings.Split(val, ",") {
						opts = append(opts, strings.TrimSpace(opt))
					}
					meta.Options = opts
				}
			}
		}

		// 2. Get default value from defaultProps instance (overrides tag defaults if non-zero)
		fieldVal := v.Field(i)
		if !fieldVal.IsZero() {
			switch field.Type.Kind() {
			case reflect.String:
				meta.DefaultString = fieldVal.String()
			case reflect.Bool:
				meta.DefaultBool = fieldVal.Bool()
			case reflect.Int:
				meta.DefaultInt = int(fieldVal.Int())
			}
		}
		if field.Type.Kind() == reflect.Int {
			meta.Step = 1 // default step
		}

		// 3. Apply overrides
		if override, ok := overrides[field.Name]; ok {
			if override.Label != "" {
				meta.ControlName = override.Label
			}
			if override.Min != 0 || override.Max != 0 {
				meta.Min = override.Min
				meta.Max = override.Max
			}
			if override.Step != 0 {
				meta.Step = override.Step
			}
			if len(override.Options) > 0 {
				meta.Options = override.Options
			}
			if override.Hidden {
				meta.Hidden = true
			}
		}

		list = append(list, meta)
	}

	return list
}

// populateProps populates a new instance of P from control values in Context c.
func populateProps[P any](c *Context, baseProps P, metadata []fieldMetadata) P {
	// Start with a copy of baseProps (so any non-control fields like callbacks remain intact)
	props := baseProps
	
	v := reflect.ValueOf(&props).Elem()

	for _, meta := range metadata {
		if meta.Hidden {
			continue
		}

		fieldVal := v.FieldByName(meta.FieldName)
		if !fieldVal.CanSet() {
			continue
		}

		switch meta.FieldType.Kind() {
		case reflect.String:
			var val string
			currentDefault := fieldVal.String()
			if len(meta.Options) > 0 {
				val = c.Select(meta.ControlName, meta.Options, currentDefault)
			} else {
				val = c.Text(meta.ControlName, currentDefault)
			}
			fieldVal.SetString(val)
		case reflect.Bool:
			currentDefault := fieldVal.Bool()
			val := c.Bool(meta.ControlName, currentDefault)
			fieldVal.SetBool(val)
		case reflect.Int:
			currentDefault := int(fieldVal.Int())
			val := c.Int(meta.ControlName, currentDefault)
			if meta.Min != 0 || meta.Max != 0 {
				if val < meta.Min {
					val = meta.Min
				}
				if val > meta.Max {
					val = meta.Max
				}
			}
			fieldVal.SetInt(int64(val))
		}
	}

	return props
}
