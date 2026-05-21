package testenv

import (
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

// Environment provides ergonomic tools for testing Kite components headless.
type Environment struct {
	Engine  *engine.Engine
	Backend *mock.Backend
}

// New creates a new test environment wrapping the given engine.
func New(eng *engine.Engine) *Environment {
	return &Environment{
		Engine: eng,
	}
}

// Default creates a new test environment with a mock backend of the given
// dimensions and a default engine.
func Default(width, height int) *Environment {
	b := mock.New(width, height)
	eng := engine.New(b, engine.Options{})
	return &Environment{
		Engine:  eng,
		Backend: b,
	}
}

// Teardown gracefully stops the engine.
func (e *Environment) Teardown() {
	e.Engine.Stop()
}

// Close is an alias for Teardown.
func (e *Environment) Close() {
	e.Teardown()
}

// Document returns the logical document root.
func (e *Environment) Document() dom.Document {
	return e.Engine.Document()
}

// Mount appends n as the body of the document.
func (e *Environment) Mount(n dom.Node) {
	if el, ok := n.(dom.Element); ok {
		e.Engine.Mount(el)
	}
}

// Flush blocks until the engine completes a frame, allowing assertions on the newly painted state.
func (e *Environment) Flush() {
	e.Engine.Frame()
}

// RenderFrame is an alias for Flush.
func (e *Environment) RenderFrame() {
	e.Flush()
}

// GetNodeByID returns the element with the given ID from the logical DOM.
func (e *Environment) GetNodeByID(id string) dom.Element {
	el := e.Engine.Document().GetElementByID(id)
	if el != nil {
		return el
	}
	return e.QuerySelector("#" + id)
}

// QuerySelector returns the first element matching the selector.
// Supports simple tag name ("div"), ID ("#id"), and class (".class") matching.
func (e *Environment) QuerySelector(selector string) dom.Element {
	return e.Engine.Document().QuerySelector(selector)
}

// SendKey simulates a key event.
func (e *Environment) SendKey(k key.Key) {
	e.Engine.ProcessRawEvent(&event.RawKeyEvent{
		Key: k,
	})
}

// Type simulates typing the given text.
func (e *Environment) Type(text string) {
	for _, r := range text {
		e.SendKey(key.Key{
			Code: r,
			Text: string(r),
		})
	}
}

// Click simulates a mouse click at (x, y).
func (e *Environment) Click(x, y int) {
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		Button: event.ButtonLeft,
	})
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		Button: event.ButtonLeft,
		Up:     true,
	})
}

// Wheel simulates a mouse wheel event at (x, y).
func (e *Environment) Wheel(x, y, dx, dy int) {
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		DeltaX: dx,
		DeltaY: dy,
	})
}

// ScrollTo sets the scroll offset of an element.
func (e *Environment) ScrollTo(el dom.Element, x, y int) {
	el.ScrollTo(x, y)
}

// ScrollBy shifts the scroll offset of an element.
func (e *Environment) ScrollBy(el dom.Element, dx, dy int) {
	el.ScrollBy(dx, dy)
}
