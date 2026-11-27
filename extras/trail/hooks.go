package trail

import (
	"strings"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/geom"
)

// AppHandle provides controls to interact with the running trail instance.
type AppHandle interface {
	// Exit terminates the application and records an optional exit error.
	Exit(err error)

	// InsertAbove permanently flushes a rendered node into the terminal scrollback
	// above the active inline frame.
	InsertAbove(node kitex.Node)

	// Println permanently writes a text line into the scrollback above the frame.
	Println(text string)

	// Clear clears the terminal and scrollback and repaints the mounted app.
	Clear()

	// Engine returns the underlying Kite Engine.
	Engine() *engine.Engine
}

// appContext is the kitex context key used to provide the AppHandle to child components.
var appContext = kitex.CreateContext[AppHandle](nil)

// UseApp returns the AppHandle for the current trail application.
func UseApp() AppHandle {
	return kitex.UseContext(appContext)
}

// InputOptions configures keyboard input handling for UseInput.
type InputOptions struct {
	IsActive bool
}

// UseInput registers a keyboard handler for raw or printable key presses.
// handler is called with the printable character (or empty string for control/navigation keys)
// and the underlying event.KeyEvent.
// By default, input handling is active unless explicitly disabled via InputOptions{IsActive: false}.
func UseInput(handler func(input string, k event.KeyEvent), opts ...InputOptions) {
	isActive := true
	if len(opts) > 0 {
		isActive = opts[0].IsActive
	}

	handlerRef := kitex.UseRef(handler)
	handlerRef.Current = handler

	kitex.UseKeyboard(func(ke event.KeyEvent) {
		if !isActive || handlerRef.Current == nil {
			return
		}
		input := ke.Text
		if input == "" && ke.Code >= 32 && ke.Code != 127 {
			input = string(ke.Code)
		}
		handlerRef.Current(input, ke)
	}, isActive)
}

// FocusOptions configures focus management for a focusable element.
type FocusOptions struct {
	AutoFocus bool
	IsActive  bool
}

// UseFocus binds a focus listener to a DOM element ref and reports whether it is focused.
// If AutoFocus is enabled (and active), it automatically focuses the element on mount.
func UseFocus(ref kitex.Ref[dom.Element], opts ...FocusOptions) bool {
	focused := kitex.UseFocus(ref)

	if len(opts) > 0 && opts[0].AutoFocus {
		kitex.UseEffect(func() {
			if ref.Current != nil {
				ref.Current.Focus()
			}
		})
	}

	if len(opts) > 0 && !opts[0].IsActive && !opts[0].AutoFocus {
		return false
	}

	return focused
}

// FocusManager provides programmatic focus navigation controls for the active Document.
type FocusManager struct {
	getDoc func() dom.Document
}

// FocusNext moves focus to the next focusable element in the document.
func (fm FocusManager) FocusNext() bool {
	if doc := fm.getDoc(); doc != nil {
		return doc.NextFocus()
	}
	return false
}

// FocusPrevious moves focus to the previous focusable element in the document.
func (fm FocusManager) FocusPrevious() bool {
	if doc := fm.getDoc(); doc != nil {
		return doc.PreviousFocus()
	}
	return false
}

// Focus moves focus to the element with the specified ID.
func (fm FocusManager) Focus(id string) bool {
	if doc := fm.getDoc(); doc != nil {
		if el := doc.GetElementByID(id); el != nil {
			el.Focus()
			return true
		}
	}
	return false
}

// Blur clears the current document focus.
func (fm FocusManager) Blur() {
	if doc := fm.getDoc(); doc != nil {
		if cur := doc.CurrentFocus(); cur != nil {
			cur.Blur()
		} else {
			doc.Focus(nil)
		}
	}
}

// UseFocusManager returns a FocusManager to control document focus programmatically.
func UseFocusManager() FocusManager {
	getDoc := kitex.UseDocument()
	return FocusManager{getDoc: getDoc}
}

// StdoutHandle provides access to inline stdout routing and terminal sizing.
type StdoutHandle struct {
	app    AppHandle
	getDoc func() dom.Document
}

// Write writes text permanently into the terminal scrollback above the inline frame.
func (h StdoutHandle) Write(s string) {
	if h.app != nil {
		h.app.Println(strings.TrimRight(s, "\r\n"))
	}
}

// Size returns the current terminal viewport size.
func (h StdoutHandle) Size() geom.Size {
	if h.getDoc != nil {
		if doc := h.getDoc(); doc != nil && doc.DefaultView() != nil {
			return doc.DefaultView().ViewportSize()
		}
	}
	return geom.Size{}
}

// UseStdout returns a StdoutHandle providing ink-compatible stdout write and sizing controls.
func UseStdout() StdoutHandle {
	app := UseApp()
	getDoc := kitex.UseDocument()
	return StdoutHandle{
		app:    app,
		getDoc: getDoc,
	}
}

// StderrHandle provides access to inline stderr routing above the inline frame.
type StderrHandle struct {
	app AppHandle
}

// Write writes text permanently into the terminal scrollback above the inline frame.
func (h StderrHandle) Write(s string) {
	if h.app != nil {
		h.app.Println(strings.TrimRight(s, "\r\n"))
	}
}

// UseStderr returns a StderrHandle providing ink-compatible stderr write controls.
func UseStderr() StderrHandle {
	app := UseApp()
	return StderrHandle{app: app}
}
