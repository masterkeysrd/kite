package stage

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

func TestStage_Register(t *testing.T) {
	stg := New()
	stg.Register("Button", []Scene{
		{
			Name: "Default",
			Render: func(c *Context) kitex.Node {
				return nil
			},
		},
	})

	if len(stg.components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(stg.components))
	}
	scenes, ok := stg.components["Button"]
	if !ok {
		t.Fatal("expected component 'Button' to exist")
	}
	if len(scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(scenes))
	}
	if scenes[0].Name != "Default" {
		t.Errorf("expected scene name 'Default', got %q", scenes[0].Name)
	}
}

func TestContext_ControlsRegistration(t *testing.T) {
	values := make(map[string]any)
	setVal := func(name string, val any) {
		values[name] = val
	}

	c := NewContext(values, setVal, nil)

	// First query registers controls and sets defaults
	txtVal := c.Text("TextControl", "default-text")
	boolVal := c.Bool("BoolControl", true)
	selectVal := c.Select("SelectControl", []string{"a", "b"}, "a")

	if txtVal != "default-text" {
		t.Errorf("expected text default-text, got %q", txtVal)
	}
	if boolVal != true {
		t.Errorf("expected bool true, got %v", boolVal)
	}
	if selectVal != "a" {
		t.Errorf("expected select a, got %q", selectVal)
	}

	// Verify they are registered in metadata
	ctrls := c.Controls()
	if len(ctrls) != 3 {
		t.Fatalf("expected 3 controls, got %d", len(ctrls))
	}

	types := make(map[string]ControlType)
	for _, ctrl := range ctrls {
		types[ctrl.Name] = ctrl.Type
	}

	if types["TextControl"] != ControlTypeText {
		t.Errorf("expected TextControl type text, got %q", types["TextControl"])
	}
	if types["BoolControl"] != ControlTypeBool {
		t.Errorf("expected BoolControl type bool, got %q", types["BoolControl"])
	}
	if types["SelectControl"] != ControlTypeSelect {
		t.Errorf("expected SelectControl type select, got %q", types["SelectControl"])
	}

	// Subsequent query retrieves updated value from map
	values["TextControl"] = "new-text"
	newTxtVal := c.Text("TextControl", "default-text")
	if newTxtVal != "new-text" {
		t.Errorf("expected retrieved value new-text, got %q", newTxtVal)
	}
}

func TestContext_ActionLogs(t *testing.T) {
	logAddedCount := 0
	c := NewContext(nil, nil, func() {
		logAddedCount++
	})

	c.Log("action 1")
	c.Log("action 2")

	logs := c.Logs()
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Message != "action 1" {
		t.Errorf("expected first log message 'action 1', got %q", logs[0].Message)
	}
	if logs[1].Message != "action 2" {
		t.Errorf("expected second log message 'action 2', got %q", logs[1].Message)
	}
	if logAddedCount != 2 {
		t.Errorf("expected callback called 2 times, got %d", logAddedCount)
	}

	c.ClearLogs()
	if len(c.Logs()) != 0 {
		t.Errorf("expected logs cleared, got %d items", len(c.Logs()))
	}
	if logAddedCount != 3 { // clear triggers callback too
		t.Errorf("expected callback called 3 times after clear, got %d", logAddedCount)
	}
}

func TestStageApp_ReconcilesKnobs(t *testing.T) {
	stg := New()
	stg.Register("Button", []Scene{
		{
			Name: "Default",
			Render: func(c *Context) kitex.Node {
				return kitex.Text(c.Text("Label", "DefaultVal"))
			},
		},
	})

	doc := dom.NewDocument()
	container := element.NewBox(doc)

	// Render the stage UI
	kitex.Render(renderUI(stg), container)

	// Helper to find the input element in the tree
	var findInput func(dom.Node) *element.InputElement
	findInput = func(n dom.Node) *element.InputElement {
		if el, ok := n.(*element.InputElement); ok {
			return el
		}
		for child := range n.ChildNodes() {
			if found := findInput(child); found != nil {
				return found
			}
		}
		return nil
	}

	// Helper to find a text node with specific content
	var findText func(dom.Node, string) dom.Node
	findText = func(n dom.Node, content string) dom.Node {
		if txt, ok := n.(*element.TextElement); ok && txt.TextContent() == content {
			return txt
		}
		for child := range n.ChildNodes() {
			if found := findText(child, content); found != nil {
				return found
			}
		}
		return nil
	}

	var printTree func(dom.Node, int)
	printTree = func(n dom.Node, depth int) {
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "  "
		}
		if el, ok := n.(dom.Element); ok {
			var idStr string
			if el.ID() != "" {
				idStr = fmt.Sprintf(" ID=%q", el.ID())
			}
			println(indent + "<" + el.TagName() + idStr + ">")
		} else if txt, ok := n.(*element.TextElement); ok {
			println(indent + "#text " + fmt.Sprintf("%q", txt.TextContent()))
		} else {
			println(indent + fmt.Sprintf("%T", n))
		}
		for child := range n.ChildNodes() {
			printTree(child, depth+1)
		}
	}

	println("--- TREE BEFORE ---")
	printTree(container, 0)

	// Verify default value is rendered in the Canvas
	defaultText := findText(container, "DefaultVal")
	if defaultText == nil {
		t.Fatal("expected default value 'DefaultVal' to be rendered in Canvas")
	}

	// Find the knob input widget in the Controls panel
	inp := findInput(container)
	if inp == nil {
		t.Fatal("expected to find InputElement in Controls panel")
	}

	// Simulate user typing a new value into the input field
	inp.SetValue("NewVal")
	inp.DispatchEvent(event.NewInput("NewVal"))

	println("--- TREE AFTER ---")
	printTree(container, 0)

	// Verify that the Canvas text node has updated to the new value
	newText := findText(container, "NewVal")
	if newText == nil {
		t.Fatal("expected Canvas text to update to 'NewVal' after input event")
	}
}

func TestStageApp_LayoutWidth(t *testing.T) {
	stg := New()
	stg.Register("Button", []Scene{
		{
			Name: "Default",
			Render: func(c *Context) kitex.Node {
				return kitex.Text(c.Text("Label", "DefaultVal"))
			},
		},
	})

	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	root := element.NewBox(eng.Document())
	root.Style(style.S().Width(style.Percent(100)).Height(style.Percent(100)))
	eng.Mount(root)

	kitex.Render(renderUI(stg), root)
	eng.Frame()

	var findInputROs func(ro render.Object) []render.Object
	findInputROs = func(ro render.Object) []render.Object {
		if ro == nil {
			return nil
		}
		var res []render.Object
		if el, ok := ro.EventTarget().(dom.Element); ok && el.TagName() == "input" {
			res = append(res, ro)
		}
		for child := range ro.Children() {
			res = append(res, findInputROs(child)...)
		}
		return res
	}

	rootRO := eng.RenderObject(root)
	inputROs := findInputROs(rootRO)
	if len(inputROs) == 0 {
		t.Fatal("expected to find at least one input element render object")
	}

	for _, ro := range inputROs {
		frag := ro.Fragment()
		var width int
		if frag != nil {
			width = frag.Size.Width
		}
		el := ro.EventTarget().(dom.Element)

		path := el.TagName()
		for p := el.Parent(); p != nil; p = p.Parent() {
			if pel, ok := p.(dom.Element); ok {
				path = pel.TagName() + " > " + path
			}
		}

		t.Logf("Input element DOM path: %s, Width=%d, FragNil=%v", path, width, frag == nil)
		if frag != nil && frag.Size.Width <= 3 {
			t.Errorf("Input element width is too narrow: %d", frag.Size.Width)
		}
	}
}
