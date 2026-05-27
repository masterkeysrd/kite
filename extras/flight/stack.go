package flight

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/internal/focus"
)

// StackProps specifies the configuration properties for the Stack navigation component.
type StackProps struct {
	InitialRoute Route
	RenderRoute  func(Route) kitex.Node
}

type stackNavigator struct {
	getStack func() []Route
	setStack func([]Route)
}

var _ Navigator = (*stackNavigator)(nil)

func (n *stackNavigator) Push(r Route) {
	s := n.getStack()
	next := make([]Route, len(s)+1)
	copy(next, s)
	next[len(s)] = r
	n.setStack(next)
}

func (n *stackNavigator) Pop() {
	s := n.getStack()
	if len(s) <= 1 {
		return
	}
	next := make([]Route, len(s)-1)
	copy(next, s)
	n.setStack(next)
}

func (n *stackNavigator) Replace(r Route) {
	s := n.getStack()
	if len(s) == 0 {
		n.setStack([]Route{r})
		return
	}
	next := make([]Route, len(s))
	copy(next, s)
	next[len(s)-1] = r
	n.setStack(next)
}

func (n *stackNavigator) Reset(r Route) {
	n.setStack([]Route{r})
}

// Stack is a reactive navigation component that maintains a stack of active routes.
// It wraps the active route with NavigatorContext provider and dynamically isolates
// keyboard interaction to the top-most view using the active route's DOM node.
var Stack = kitex.FC("Stack", func(props StackProps) kitex.Node {
	if props.InitialRoute == nil {
		panic("flight.Stack requires an InitialRoute")
	}
	if props.RenderRoute == nil {
		panic("flight.Stack requires a RenderRoute function")
	}

	getStack, setStack := kitex.UseState([]Route{props.InitialRoute})

	nav := kitex.UseMemo(func() Navigator {
		return &stackNavigator{
			getStack: getStack,
			setStack: setStack,
		}
	}, []any{})

	stack := getStack()
	if len(stack) == 0 {
		return kitex.Box(kitex.BoxProps{})
	}

	active := stack[len(stack)-1]
	rendered := props.RenderRoute(active)

	getDoc := kitex.UseDocument()
	getElement := kitex.UseElement()

	kitex.UseLayoutEffectCleanup(func() func() {
		doc := getDoc()
		if doc == nil {
			return nil
		}

		el := getElement()
		if el == nil {
			return nil
		}

		rawEl := el
		for {
			if unwrapped := rawEl.Unwrap(); unwrapped != nil {
				rawEl = unwrapped
			} else {
				break
			}
		}

		domEl := rawEl.(dom.Element)
		scope := &focus.Scope{
			Root:      domEl,
			Autofocus: domEl,
		}
		doc.PushScope(scope)

		return func() {
			doc.PopScope()
		}
	}, []any{active})

	return navigatorContext.Provider(nav, rendered)
})
