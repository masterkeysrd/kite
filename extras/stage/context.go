package stage

import (
	"sync"
	"time"
)

// ControlType represents the type of interactive knob/input control.
type ControlType string

const (
	ControlTypeText   ControlType = "text"
	ControlTypeBool   ControlType = "bool"
	ControlTypeSelect ControlType = "select"
	ControlTypeInt    ControlType = "int"
)

// Control contains the metadata for a single scene knob.
type Control struct {
	Name    string      `json:"name"`
	Type    ControlType `json:"type"`
	Default any         `json:"default"`
	Options []string    `json:"options,omitempty"` // populated only for ControlTypeSelect
}

// ActionLog holds a single logged event from a running scene.
type ActionLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// Context is passed to each Scene's Render function to register/query knobs and log actions.
type Context struct {
	mu           sync.RWMutex
	controls     map[string]Control
	values       map[string]any
	setVal       func(name string, val any)
	globals      map[string]Control
	globalValues map[string]any
	logs         []ActionLog
	onLogAdded   func()
}

// NewContext creates a new Context for scene execution.
func NewContext(values map[string]any, setVal func(name string, val any), onLogAdded func()) *Context {
	return &Context{
		controls:   make(map[string]Control),
		values:     values,
		setVal:     setVal,
		onLogAdded: onLogAdded,
	}
}

// Text registers/returns a text input knob.
func (c *Context) Text(name string, defaultValue string) string {
	c.mu.Lock()
	if _, exists := c.controls[name]; !exists {
		c.controls[name] = Control{
			Name:    name,
			Type:    ControlTypeText,
			Default: defaultValue,
		}
	}
	c.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[name]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultValue
}

// Bool registers/returns a boolean checkbox knob.
func (c *Context) Bool(name string, defaultValue bool) bool {
	c.mu.Lock()
	if _, exists := c.controls[name]; !exists {
		c.controls[name] = Control{
			Name:    name,
			Type:    ControlTypeBool,
			Default: defaultValue,
		}
	}
	c.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[name]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// Select registers/returns a dropdown-like select input knob.
func (c *Context) Select(name string, options []string, defaultValue string) string {
	c.mu.Lock()
	if _, exists := c.controls[name]; !exists {
		c.controls[name] = Control{
			Name:    name,
			Type:    ControlTypeSelect,
			Default: defaultValue,
			Options: options,
		}
	}
	c.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[name]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultValue
}

// Int registers/returns an integer spinner knob.
func (c *Context) Int(name string, defaultValue int) int {
	c.mu.Lock()
	if _, exists := c.controls[name]; !exists {
		c.controls[name] = Control{
			Name:    name,
			Type:    ControlTypeInt,
			Default: defaultValue,
		}
	}
	c.mu.Unlock()

	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.values[name]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// Log adds an action log entry.
func (c *Context) Log(msg string) {
	c.mu.Lock()
	c.logs = append(c.logs, ActionLog{
		Timestamp: time.Now().Format("15:04:05"),
		Message:   msg,
	})
	c.mu.Unlock()
	if c.onLogAdded != nil {
		c.onLogAdded()
	}
}

// Logs returns a safe copy of all action logs.
func (c *Context) Logs() []ActionLog {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]ActionLog, len(c.logs))
	copy(res, c.logs)
	return res
}

// ClearLogs clears all logged actions.
func (c *Context) ClearLogs() {
	c.mu.Lock()
	c.logs = nil
	c.mu.Unlock()
	if c.onLogAdded != nil {
		c.onLogAdded()
	}
}

// Controls returns all currently registered controls.
func (c *Context) Controls() []Control {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]Control, 0, len(c.controls))
	for _, ctrl := range c.controls {
		res = append(res, ctrl)
	}
	return res
}

// WithGlobals assigns global controls and their current values to the Context.
func (c *Context) WithGlobals(globals map[string]Control, globalValues map[string]any) *Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.globals = globals
	c.globalValues = globalValues
	return c
}

// Global returns the current value of a global control by name.
func (c *Context) Global(name string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.globalValues[name]; ok {
		return val
	}
	if ctrl, ok := c.globals[name]; ok {
		return ctrl.Default
	}
	return nil
}

// GlobalString returns the value of a global control as a string.
func (c *Context) GlobalString(name string, defaultValue string) string {
	val := c.Global(name)
	if s, ok := val.(string); ok {
		return s
	}
	return defaultValue
}

// GlobalBool returns the value of a global control as a boolean.
func (c *Context) GlobalBool(name string, defaultValue bool) bool {
	val := c.Global(name)
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultValue
}

// GlobalInt returns the value of a global control as an integer.
func (c *Context) GlobalInt(name string, defaultValue int) int {
	val := c.Global(name)
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return defaultValue
}
