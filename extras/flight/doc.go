// Package flight implements a stack-based navigation system for Kitex applications.
//
// By favoring a type-safe stack navigator (push/pop) over URL-based path routing,
// flight provides a paradigm that matches TUI (Terminal User Interface) design
// patterns and enforces strict keyboard focus containment to the active screen.
//
// # Type-Safe Routes
// Routes are represented as empty interfaces (flight.Route). Concrete routes
// are defined using user-defined structs which hold type-safe parameters:
//
//	type HomeRoute struct{}
//	type DetailsRoute struct {
//	    ItemID string
//	}
//
// # Rendering and Type Switches
// The Stack component renders the active route by dispatching it to a developer-provided
// RenderRoute function, which uses a Go type switch to instantiate the corresponding component:
//
//	flight.Stack(flight.StackProps{
//	    InitialRoute: HomeRoute{},
//	    RenderRoute: func(r flight.Route) kitex.Node {
//	        switch route := r.(type) {
//	        case HomeRoute:
//	            return HomeView()
//	        case DetailsRoute:
//	            return DetailsView(route.ItemID)
//	        default:
//	            panic("unknown route")
//	        }
//	    },
//	})
package flight
