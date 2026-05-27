package form

import (
	"errors"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type TestData struct {
	Name     string `json:"name"`
	Age      int    `json:"age"`
	IsActive bool   `json:"is_active"`
}

func TestFormHook(t *testing.T) {
	doc := dom.NewDocument()

	var formAPI API[TestData]
	var onSubmitCalled bool
	var submittedValues TestData

	myComp := kitex.SimpleFC("TestComp", func() kitex.Node {
		formAPI = Use(Options[TestData]{
			InitialValues: TestData{Name: "Default", Age: 18},
			Validate: func(d TestData) map[string]string {
				errs := make(map[string]string)
				if len(d.Name) < 3 {
					errs["name"] = "Too short"
				}
				return errs
			},
			OnSubmit: func(d TestData) error {
				onSubmitCalled = true
				submittedValues = d
				return nil
			},
		})
		return kitex.Box(kitex.BoxProps{})
	})

	// Initial render
	node := myComp()
	realNode := node.Instantiate(doc)

	// Verify initial state
	state := formAPI.State()
	if state.Values.Name != "Default" || state.Values.Age != 18 {
		t.Errorf("expected initial values, got %+v", state.Values)
	}
	if !state.IsValid {
		t.Errorf("expected initial state to be valid")
	}

	// 1. Test Validation Failure
	formAPI.HandleSubmit(map[string]any{
		"name":      "Jo",
		"age":       25,
		"is_active": true,
	})
	// Trigger update to see changes
	node.Update(realNode, node)

	state = formAPI.State()
	if state.IsValid {
		t.Errorf("expected state to be invalid due to short name")
	}
	if state.Errors["name"] != "Too short" {
		t.Errorf("expected 'Too short' error, got %s", state.Errors["name"])
	}
	if onSubmitCalled {
		t.Errorf("OnSubmit should not be called when validation fails")
	}

	// 2. Test Successful Submission (Sync)
	onSubmitCalled = false
	formAPI.HandleSubmit(map[string]any{
		"name":      "John",
		"age":       30,
		"is_active": true,
	})
	node.Update(realNode, node)

	state = formAPI.State()
	if state.IsSubmitting {
		t.Errorf("expected IsSubmitting to be false after sync completion")
	}
	if !onSubmitCalled {
		t.Errorf("expected OnSubmit to be called")
	}
	if submittedValues.Name != "John" || submittedValues.Age != 30 || !submittedValues.IsActive {
		t.Errorf("submitted values do not match: %+v", submittedValues)
	}

	// 3. Test SetError
	formAPI.SetError("manual", "Something went wrong")
	node.Update(realNode, node)
	state = formAPI.State()
	if state.IsValid {
		t.Errorf("expected IsValid false after SetError")
	}
	if state.Errors["manual"] != "Something went wrong" {
		t.Errorf("expected manual error")
	}

	formAPI.SetError("manual", "")
	node.Update(realNode, node)
	state = formAPI.State()
	if !state.IsValid {
		t.Errorf("expected IsValid true after clearing manual error")
	}
}

func TestFormSubmitError(t *testing.T) {
	doc := dom.NewDocument()
	var formAPI API[TestData]

	myComp := kitex.SimpleFC("TestComp", func() kitex.Node {
		formAPI = Use(Options[TestData]{
			OnSubmit: func(d TestData) error {
				return errors.New("submission failed")
			},
		})
		return kitex.Box(kitex.BoxProps{})
	})

	node := myComp()
	_ = node.Instantiate(doc)

	formAPI.HandleSubmit(map[string]any{"name": "test"})

	state := formAPI.State()
	if state.Errors["root"] != "submission failed" {
		t.Errorf("expected root error, got %v", state.Errors["root"])
	}
	if state.IsValid {
		t.Errorf("expected IsValid false after submit error")
	}
}
