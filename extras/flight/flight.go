package flight

// Route represents a target screen or view to navigate to.
// Concrete route configurations are user-defined structs.
type Route interface{}

// Navigator defines the operations to manipulate the navigation history stack.
type Navigator interface {
	// Push pushes a new route onto the stack.
	Push(r Route)
	// Pop pops the current route from the stack, returning to the previous one.
	Pop()
	// Replace replaces the current top route with a new one.
	Replace(r Route)
	// Reset clears the stack and sets the given route as the only entry.
	Reset(r Route)
}
