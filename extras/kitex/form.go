package kitex

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// FormProps specifies attributes for Form elements.
type FormProps struct {
	Key      string
	ID       string
	Class    string
	Style    style.Style
	Hidden   bool
	OnSubmit func(map[string]any)
	Ref      refSetter
	Children []Node
}

func (p FormProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Ref: p.Ref,
	}
}

// Form creates a VDOM representation of a FormElement.
var Form = FCC("Form", func(props FormProps) Node {
	score, hasP, hasDirectP := buildElementInfo(props.Children)
	return trackSource(&elementNode[FormProps]{
		tagName:  "form",
		props:    props,
		children: props.Children,
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
})
