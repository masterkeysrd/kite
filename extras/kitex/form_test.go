package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
)

func TestForm_VDOM_Instantiation_AndUpdate(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	submitCount1 := 0
	submitCount2 := 0

	props1 := FormProps{
		ID: "form-vdom-test",
		OnSubmit: func(data map[string]any) {
			submitCount1++
		},
	}

	formNode1 := Form(props1)
	Render(formNode1, container)

	realForm := container.FirstChild().(*element.FormElement)

	if realForm.ID() != "form-vdom-test" {
		t.Errorf("expected form ID to be 'form-vdom-test', got %q", realForm.ID())
	}

	// Dispatch submit event and verify OnSubmit callback is fired.
	realForm.DispatchEvent(event.NewSubmitEvent(map[string]any{"username": "test"}))
	if submitCount1 != 1 {
		t.Errorf("expected submitCount1 to be 1, got %d", submitCount1)
	}

	// Update props with new OnSubmit handler and ID.
	props2 := FormProps{
		ID: "form-vdom-test-updated",
		OnSubmit: func(data map[string]any) {
			submitCount2++
		},
	}

	formNode2 := Form(props2)
	Render(formNode2, container)

	if realForm.ID() != "form-vdom-test-updated" {
		t.Errorf("expected form ID to be 'form-vdom-test-updated', got %q", realForm.ID())
	}

	// Dispatch submit event and verify the updated OnSubmit callback is fired.
	realForm.DispatchEvent(event.NewSubmitEvent(map[string]any{"username": "test"}))
	if submitCount1 != 1 {
		t.Errorf("expected submitCount1 to remain 1, got %d", submitCount1)
	}
	if submitCount2 != 1 {
		t.Errorf("expected submitCount2 to be 1, got %d", submitCount2)
	}
}
