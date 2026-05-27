package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// FormElement implements the form element.
type FormElement struct {
	elementBase[FormElement]
}

var _ Element = (*FormElement)(nil)

// NewForm creates a new FormElement owned by doc.
func NewForm(doc dom.Document, children ...any) *FormElement {
	f := &FormElement{}
	el := doc.CreateElement("form", f)
	f.initBase(el, f, style.Style{
		Display: style.Some(style.DisplayBlock),
	})
	processChildren(f, children)
	f.wireEvents()
	return f
}

// Form creates a new FormElement using the orphan document.
func Form(children ...any) *FormElement {
	return NewForm(orphanDocument, children...)
}

// Submit collects all FormControl values in the form's subtree
// and dispatches a SubmitEvent.
func (f *FormElement) Submit() {
	formData := make(map[string]any)
	var walk func(n dom.Node)
	walk = func(n dom.Node) {
		target := n.EventTarget()
		if fc, ok := target.(dom.FormControl); ok {
			name := fc.Name()
			if name != "" {
				// Radio elements only submit their value if they are checked.
				if radio, ok := fc.(*RadioElement); ok {
					if radio.Checked() {
						formData[name] = fc.Value()
					}
				} else {
					formData[name] = fc.Value()
				}
			}
		}
		for child := range n.ChildNodes() {
			walk(child)
		}
	}
	walk(f)

	f.DispatchEvent(event.NewSubmitEvent(formData))
}

func (f *FormElement) wireEvents() {
	// Capture phase listeners to handle Enter in inputs and Click on submit buttons.
	f.AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke, ok := e.(*event.KeyEvent)
		if !ok {
			return
		}
		if ke.MatchString("enter") {
			if _, ok := e.Target().(*InputElement); ok {
				f.Submit()
				e.StopPropagation()
			}
		}
	}, event.Capture())

	f.AddEventListener(event.EventClick, func(e event.Event) {
		if btn, ok := e.Target().(*ButtonElement); ok {
			if btn.ButtonType() == "submit" {
				f.Submit()
				e.StopPropagation()
			}
		}
	}, event.Capture())
}
