package kitex

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
)

// Form creates a VDOM representation of a FormElement.
var Form ContainerTag[FormProps] = func(props FormProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[FormProps]{
		tagName:  "form",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			f := element.NewForm(doc)
			if props.OnSubmit != nil {
				sub := f.AddEventListener(event.EventSubmit, func(e event.Event) {
					if se, ok := e.(*event.SubmitEvent); ok {
						props.OnSubmit(se.FormData)
					}
				})
				setSubscription(f, event.EventSubmit, sub)
			}
			return f
		},
		update: func(el dom.Node, old, new *FormProps) {
			f := el.(*element.FormElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(f, &oldEp, &newEp)

			if old == nil || !funcEquals(old.OnSubmit, new.OnSubmit) {
				clearSubscription(f, event.EventSubmit)
				if new.OnSubmit != nil {
					sub := f.AddEventListener(event.EventSubmit, func(e event.Event) {
						if se, ok := e.(*event.SubmitEvent); ok {
							new.OnSubmit(se.FormData)
						}
					})
					setSubscription(f, event.EventSubmit, sub)
				}
			}
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}
