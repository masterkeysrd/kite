package flight

import (
	"github.com/masterkeysrd/kite/extras/kitex"
)

// navigatorContext is the kitex context used to provide Navigator access to descendants.
var navigatorContext = kitex.CreateContext[Navigator](nil)

// UseNavigation retrieves the current Navigator instance from the kitex context.
// It panics if called outside of a flight.Stack component.
func UseNavigation() Navigator {
	nav := kitex.UseContext(navigatorContext)
	if nav == nil {
		panic("UseNavigation must be called inside a functional component that is a descendant of a flight.Stack")
	}
	return nav
}
