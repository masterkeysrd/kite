package kitex

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// FormProps specifies attributes for Form elements.
type FormProps struct {
	Attributes      map[string]string
	Key             string
	ID              string
	Class           string
	Style           style.Style
	Hidden          bool
	Disabled        bool
	CursorNavigable bool
	OnKeyDown       func(*event.KeyEvent)
	OnKeyUp         func(*event.KeyEvent)
	OnKeyPress      func(*event.KeyEvent)
	OnMouseDown     func(*event.MouseEvent)
	OnMouseUp       func(*event.MouseEvent)
	OnMouseMove     func(*event.MouseEvent)
	OnClick         func(*event.MouseEvent)
	OnDrag          func(*event.MouseEvent)
	OnWheel         func(*event.WheelEvent)
	OnFocus         func(*event.FocusEvent)
	OnBlur          func(*event.FocusEvent)
	OnChange        func(*event.ChangeEvent)
	OnInput         func(*event.InputEvent)
	OnScroll        func(*event.ScrollEvent)
	OnMouseEnter    func(*event.MouseEvent)
	OnMouseLeave    func(*event.MouseEvent)
	OnMouseOver     func(*event.MouseEvent)
	OnMouseOut      func(*event.MouseEvent)
	OnSubmit        func(map[string]any)
	Ref             refSetter
	Children        []Node
}

func (p FormProps) elementProps() ElementProps {
	return ElementProps{
		Attributes: p.Attributes,
		Key:        p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:        p.Disabled,
		CursorNavigable: p.CursorNavigable,
		OnKeyDown:       p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnInput: p.OnInput, OnScroll: p.OnScroll,
		OnMouseEnter: p.OnMouseEnter, OnMouseLeave: p.OnMouseLeave, OnMouseOver: p.OnMouseOver, OnMouseOut: p.OnMouseOut,
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
