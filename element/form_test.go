package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

func TestForm_Submit_DataGathering(t *testing.T) {
	doc := dom.NewDocument()
	form := element.NewForm(doc)

	username := element.NewInput(doc, "john_doe").WithName("username")
	agree := element.NewCheckbox(doc, true).WithName("agree")

	// Radio group
	rg := element.NewRadioGroup(doc)
	male := element.NewRadio(doc, "male").WithName("gender")
	female := element.NewRadio(doc, "female").WithName("gender")
	rg.AppendChild(male)
	rg.AppendChild(female)
	rg.SetValue("male")

	form.AppendChild(username)
	form.AppendChild(agree)
	form.AppendChild(rg)

	var submittedData map[string]any
	form.OnEvent(event.EventSubmit, func(e event.Event) {
		se, ok := e.(*event.SubmitEvent)
		if !ok {
			t.Error("expected SubmitEvent")
			return
		}
		submittedData = se.FormData
	})

	form.Submit()

	if submittedData == nil {
		t.Fatal("form submit event was not fired or collected no data")
	}

	if submittedData["username"] != "john_doe" {
		t.Errorf("expected username to be 'john_doe', got %v", submittedData["username"])
	}

	if submittedData["agree"] != true {
		t.Errorf("expected agree to be true, got %v", submittedData["agree"])
	}

	if submittedData["gender"] != "male" {
		t.Errorf("expected gender to be 'male', got %v", submittedData["gender"])
	}
}

func TestForm_ImplicitSubmit_EnterOnInput(t *testing.T) {
	doc := dom.NewDocument()
	form := element.NewForm(doc)
	inp := element.NewInput(doc, "test").WithName("q")
	form.AppendChild(inp)

	submitted := false
	form.OnEvent(event.EventSubmit, func(e event.Event) {
		submitted = true
	})

	// Dispatch KeyDown Enter on the input element.
	d := event.NewDispatcher()
	// Capture -> Target -> Bubble sequence
	path := []event.EventTarget{form, inp}

	ke := event.NewKeyEvent(event.EventKeyDown, key.Key{Code: key.KeyEnter})
	d.Dispatch(ke, path)

	if !submitted {
		t.Error("Enter key on InputElement did not trigger form submission")
	}
	if !ke.PropagationStopped() {
		t.Error("expected enter key propagation to be stopped")
	}
}

func TestForm_ImplicitSubmit_ClickOnSubmitButton(t *testing.T) {
	doc := dom.NewDocument()
	form := element.NewForm(doc)
	btn := element.NewButton(doc, "Submit").Type("submit")
	form.AppendChild(btn)

	submitted := false
	form.OnEvent(event.EventSubmit, func(e event.Event) {
		submitted = true
	})

	// Dispatch Click on the button element.
	d := event.NewDispatcher()
	path := []event.EventTarget{form, btn}

	ce := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(ce, path)

	if !submitted {
		t.Error("Click on submit button did not trigger form submission")
	}
	if !ce.PropagationStopped() {
		t.Error("expected click propagation to be stopped")
	}
}

func TestForm_ImplicitSubmit_ClickOnNormalButton_NoSubmit(t *testing.T) {
	doc := dom.NewDocument()
	form := element.NewForm(doc)
	btn := element.NewButton(doc, "Button").Type("button")
	form.AppendChild(btn)

	submitted := false
	form.OnEvent(event.EventSubmit, func(e event.Event) {
		submitted = true
	})

	// Dispatch Click on the button element.
	d := event.NewDispatcher()
	path := []event.EventTarget{form, btn}

	ce := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(ce, path)

	if submitted {
		t.Error("Click on non-submit button triggered form submission")
	}
	if ce.PropagationStopped() {
		t.Error("expected propagation not to be stopped for non-submit button")
	}
}
