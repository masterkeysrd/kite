package trail

import (
	"strings"
	"time"

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

	// Clear clears the current dynamic inline frame output.
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
	}, []any{isActive})
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
		}, nil)
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

// Common kitex hooks re-exported for convenience in trail components.

// UseState declares a reactive component state variable with getter and setter.
func UseState[T any](initial T) (func() T, func(T)) {
	return kitex.UseState(initial)
}

// UseReducer declares a state machine managed by a reducer function.
func UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A)) {
	return kitex.UseReducer(reducer, initial)
}

// UseRef persists a mutable reference value across component renders.
func UseRef[T any](initial T) kitex.Ref[T] {
	return kitex.UseRef(initial)
}

// CreateRef creates a new standalone Ref.
func CreateRef[T any]() kitex.Ref[T] {
	return kitex.CreateRef[T]()
}

// UseMemo memoizes an expensive calculation factory across renders.
func UseMemo[T any](factory func() T, deps []any) T {
	return kitex.UseMemo(factory, deps)
}

// UseCallback memoizes a callback function identity across renders.
func UseCallback[T any](callback T, deps []any) T {
	return kitex.UseCallback(callback, deps)
}

// UseEffect runs an effect callback after DOM changes are committed.
var UseEffect = kitex.UseEffect

// UseEffectCleanup runs an effect callback with cleanup after DOM commit.
var UseEffectCleanup = kitex.UseEffectCleanup

// UseLayoutEffect runs an effect callback synchronously after component reconciliation.
var UseLayoutEffect = kitex.UseLayoutEffect

// UseLayoutEffectCleanup runs an effect with cleanup synchronously after component reconciliation.
var UseLayoutEffectCleanup = kitex.UseLayoutEffectCleanup

// UseInterval schedules a periodic callback on the engine task scheduler.
func UseInterval(handler func(), interval time.Duration, deps []any) {
	kitex.UseInterval(handler, interval, deps)
}

// UseTimeout schedules a delayed callback on the engine task scheduler.
func UseTimeout(handler func(), delay time.Duration, deps []any) {
	kitex.UseTimeout(handler, delay, deps)
}

// UseWindowSize returns the current viewport size in terminal cells.
func UseWindowSize() geom.Size {
	return kitex.UseViewportSize()
}

// CreateContext creates a new kitex context.
func CreateContext[T comparable](defaultValue T) *kitex.Context[T] {
	return kitex.CreateContext(defaultValue)
}

// UseContext reads a value from a kitex context provider.
func UseContext[T comparable](ctx *kitex.Context[T]) T {
	return kitex.UseContext(ctx)
}
