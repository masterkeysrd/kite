package trail

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
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
}

// appContext is the kitex context key used to provide the AppHandle to child components.
var appContext = kitex.CreateContext[AppHandle](nil)

// UseApp returns the AppHandle for the current trail application.
func UseApp() AppHandle {
	return kitex.UseContext(appContext)
}

// UseInput registers a keyboard handler for raw or printable key presses.
// handler is called with the printable character (or empty string for control/navigation keys)
// and the underlying event.KeyEvent.
func UseInput(handler func(input string, k event.KeyEvent)) {
	handlerRef := kitex.UseRef(handler)
	handlerRef.Current = handler

	kitex.UseKeyboard(func(ke event.KeyEvent) {
		if handlerRef.Current != nil {
			input := ke.Text
			if input == "" && ke.Code >= 32 && ke.Code != 127 {
				input = string(ke.Code)
			}
			handlerRef.Current(input, ke)
		}
	}, nil)
}
