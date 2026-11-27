package form

import (
	"errors"
	"strings"
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

	onSubmit := func(d TestData) error {
		onSubmitCalled = true
		submittedValues = d
		return nil
	}

	myComp := kitex.FC("TestComp", func(_ kitex.NoProps) kitex.Node {
		formAPI = Use(Options[TestData]{
			InitialValues: TestData{Name: "Default", Age: 18},
			Validate: func(d TestData) map[string]string {
				errs := make(map[string]string)
				if len(d.Name) < 3 {
					errs["name"] = "Too short"
				}
				return errs
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
	formAPI.HandleSubmit(onSubmit)(map[string]any{
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
	formAPI.HandleSubmit(onSubmit)(map[string]any{
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

	myComp := kitex.FC("TestComp", func(_ kitex.NoProps) kitex.Node {
		formAPI = Use(Options[TestData]{})
		return kitex.Box(kitex.BoxProps{})
	})

	node := myComp()
	_ = node.Instantiate(doc)

	onSubmit := func(d TestData) error {
		return errors.New("submission failed")
	}
	formAPI.HandleSubmit(onSubmit)(map[string]any{"name": "test"})

	state := formAPI.State()
	if state.Errors["root"] != "submission failed" {
		t.Errorf("expected root error, got %v", state.Errors["root"])
	}
	if state.IsValid {
		t.Errorf("expected IsValid false after submit error")
	}
}

func TestDecodeFormData_TagsAndConversions(t *testing.T) {
	type payload struct {
		FormTagged string `form:"user_handle,omitempty" json:"ignored_json"`
		JSONTagged string `json:"display_name,omitempty"`
		Ignored    string `form:"-"`
		JSONSkip   string `json:"-"`
		Untagged   string
		CaseFold   string
		Flag       bool    `form:"flag"`
		Count      int     `form:"count"`
		SmallInt   int8    `form:"small_int"`
		Unsigned   uint64  `form:"unsigned"`
		Score      float32 `form:"score"`
		Precise    float64 `form:"precise"`
		unexported string
	}

	var out payload
	err := decodeFormData(map[string]any{
		"user_handle":  "alice",
		"display_name": "Alice",
		"Ignored":      "should-not-bind",
		"JSONSkip":     "should-not-bind",
		"Untagged":     "exact",
		"casefold":     "folded",
		"flag":         "true",
		"count":        "-42",
		"small_int":    "127",
		"unsigned":     "9001",
		"score":        "3.5",
		"precise":      "99.25",
		"unexported":   "secret",
	}, &out)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if out.FormTagged != "alice" {
		t.Errorf("FormTagged = %q, want 'alice'", out.FormTagged)
	}
	if out.JSONTagged != "Alice" {
		t.Errorf("JSONTagged = %q, want 'Alice'", out.JSONTagged)
	}
	if out.Ignored != "" || out.JSONSkip != "" {
		t.Errorf("expected '-' tagged fields to be skipped, got Ignored=%q JSONSkip=%q", out.Ignored, out.JSONSkip)
	}
	if out.Untagged != "exact" || out.CaseFold != "folded" {
		t.Errorf("untagged/casefold mismatch: Untagged=%q CaseFold=%q", out.Untagged, out.CaseFold)
	}
	if !out.Flag || out.Count != -42 || out.SmallInt != 127 || out.Unsigned != 9001 {
		t.Errorf("primitive conversion mismatch: %+v", out)
	}
	if out.Score != 3.5 || out.Precise != 99.25 {
		t.Errorf("float conversion mismatch: Score=%v Precise=%v", out.Score, out.Precise)
	}
	if out.unexported != "" {
		t.Errorf("expected unexported field to remain empty, got %q", out.unexported)
	}
}

func TestDecodeFormData_DirectAssignableValues(t *testing.T) {
	type direct struct {
		Count   int     `form:"count"`
		Enabled bool    `form:"enabled"`
		Tags    []int   `form:"tags"`
		Ratio   float64 `form:"ratio"`
	}

	var out direct
	err := decodeFormData(map[string]any{
		"count":   7,
		"enabled": true,
		"tags":    []int{1, 2, 3},
		"ratio":   42,
	}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Count != 7 || !out.Enabled || len(out.Tags) != 3 || out.Ratio != 42 {
		t.Errorf("unexpected decoded values: %+v", out)
	}
}

func TestDecodeFormData_ErrorsAndHandleSubmit(t *testing.T) {
	type nums struct {
		Age      int     `form:"age"`
		Unsigned uint    `form:"unsigned"`
		Active   bool    `form:"active"`
		Score    float64 `form:"score"`
	}

	cases := []struct {
		name  string
		input map[string]any
		want  string
	}{
		{"invalid int", map[string]any{"age": "abc"}, "age"},
		{"invalid uint", map[string]any{"unsigned": "-5"}, "unsigned"},
		{"invalid bool", map[string]any{"active": "maybe"}, "active"},
		{"invalid float", map[string]any{"score": "nan-ish"}, "score"},
		{"incompatible type", map[string]any{"active": 123}, "active"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out nums
			err := decodeFormData(tc.input, &out)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error to mention %q, got %q", tc.want, err.Error())
			}
		})
	}

	// Verify HandleSubmit surfaces conversion error as root error without calling onSubmit.
	doc := dom.NewDocument()
	var formAPI API[nums]
	called := false
	comp := kitex.FC("Comp", func(_ kitex.NoProps) kitex.Node {
		formAPI = Use(Options[nums]{})
		return kitex.Box(kitex.BoxProps{})
	})
	comp().Instantiate(doc)

	formAPI.HandleSubmit(func(n nums) error {
		called = true
		return nil
	})(map[string]any{"age": "not-a-number"})

	if called {
		t.Fatal("expected onSubmit not to be called on conversion error")
	}
	state := formAPI.State()
	if state.IsValid || !strings.Contains(state.Errors["root"], "age") {
		t.Errorf("expected root error mentioning 'age', got state=%+v", state)
	}
}
