package stage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/geom"
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
	defer kitex.Render(nil, container)

	// Helper to find the input element in the tree
	var findInput func(dom.Node) *element.InputElement
	var inputCount int
	findInput = func(n dom.Node) *element.InputElement {
		if el, ok := n.(*element.InputElement); ok {
			inputCount++
			if inputCount == 2 {
				return el
			}
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
	defer kitex.Render(nil, root)
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

func TestStageApp_NestedSections(t *testing.T) {
	stg := New()
	// Flat component (1 level) -> Default/Default/FlatComp
	stg.Register("FlatComp", []Scene{{Name: "Scene1", Render: func(c *Context) kitex.Node { return nil }}})
	// 2-level component -> Default/Suite/Button
	stg.Register("Suite/Button", []Scene{{Name: "Scene3", Render: func(c *Context) kitex.Node { return nil }}})
	// Nested 3 levels -> Category/Subcat/Comp
	stg.Register("Category/Subcat/Comp", []Scene{{Name: "Scene2", Render: func(c *Context) kitex.Node { return nil }}})

	doc := dom.NewDocument()
	container := element.NewBox(doc)

	// Render the stage UI
	kitex.Render(renderUI(stg), container)
	defer kitex.Render(nil, container)

	// Helper to find all text elements
	var collectTexts func(dom.Node, *[]string)
	collectTexts = func(n dom.Node, res *[]string) {
		if txt, ok := n.(*element.TextElement); ok {
			*res = append(*res, txt.TextContent())
		}
		for child := range n.ChildNodes() {
			collectTexts(child, res)
		}
	}

	var textContents []string
	collectTexts(container, &textContents)

	// Helper to check if any string in slice contains substring
	hasSubstring := func(slice []string, substring string) bool {
		for _, s := range slice {
			if strings.Contains(s, substring) {
				return true
			}
		}
		return false
	}

	// Verify Category and Subcat are rendered as folder nodes
	if !hasSubstring(textContents, "Category") {
		t.Errorf("expected Category folder header in sidebar, got texts: %v", textContents)
	}
	if !hasSubstring(textContents, "Subcat") {
		t.Errorf("expected Subcat folder header in sidebar, got texts: %v", textContents)
	}
	if !hasSubstring(textContents, "Comp") {
		t.Errorf("expected Comp component header in sidebar, got texts: %v", textContents)
	}

	// Verify "Default" is rendered as a closed top-level folder
	if !hasSubstring(textContents, "◆ Default") {
		t.Errorf("expected closed '◆ Default' folder in sidebar, got texts: %v", textContents)
	}

	// FlatComp and Suite/Button should NOT be visible yet since Default is collapsed
	if hasSubstring(textContents, "FlatComp") {
		t.Errorf("FlatComp should not be visible when Default folder is collapsed, got texts: %v", textContents)
	}
	if hasSubstring(textContents, "Suite") {
		t.Errorf("Suite folder should not be visible when Default folder is collapsed, got texts: %v", textContents)
	}

	// Find the Box/Element containing "◆ Default"
	var defaultBox dom.Node
	var findDefaultBox func(dom.Node)
	findDefaultBox = func(n dom.Node) {
		if txt, ok := n.(*element.TextElement); ok && txt.TextContent() == "◆ Default" {
			defaultBox = n.Parent()
			return
		}
		for child := range n.ChildNodes() {
			findDefaultBox(child)
			if defaultBox != nil {
				return
			}
		}
	}
	findDefaultBox(container)
	if defaultBox == nil {
		t.Fatal("expected to find '◆ Default' in sidebar")
	}

	// Click to expand Default (level 1)
	el, ok := defaultBox.EventTarget().(element.Element)
	if !ok {
		t.Fatal("expected defaultBox to be an element.Element")
	}
	el.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))

	// Re-collect texts to find "▶ Suite" (level 2) and "⬢ FlatComp" (level 2)
	textContents = nil
	collectTexts(container, &textContents)

	if !hasSubstring(textContents, "▶ Suite") {
		t.Errorf("expected '▶ Suite' folder in sidebar under Default category, got texts: %v", textContents)
	}

	// FlatComp should now be visible directly under Default category!
	if !hasSubstring(textContents, "FlatComp") {
		t.Errorf("expected FlatComp header in sidebar directly under 'Default' category, got texts: %v", textContents)
	}
}

type MyTestProps struct {
	Label    string `stage:"label:Custom Label;default:Hello"`
	Disabled bool   `stage:"default:true"`
	Padding  int    `stage:"default:3;min:0;max:10"`
	Theme    string `stage:"select:light,dark;default:light"`
}

func TestStageApp_ReflectionAndRegistry(t *testing.T) {
	reg := NewRegistry()

	Register(reg, ComponentConfig[MyTestProps]{
		Name: "ReflectedComponent",
		DefaultProps: MyTestProps{
			Label:    "InitialHello",
			Disabled: true,
			Padding:  5,
			Theme:    "dark",
		},
		Render: func(c *Context, props MyTestProps) kitex.Node {
			println("RENDERING SCENE PROPS:", props.Label, props.Disabled, props.Padding, props.Theme)
			return kitex.Text(fmt.Sprintf("%s-%v-%d-%s", props.Label, props.Disabled, props.Padding, props.Theme))
		},
		Controls: map[string]ControlOverride{
			"Label": {
				Label: "Override Label",
			},
		},
		Scenes: []SceneConfig[MyTestProps]{
			{
				Name: "Variation1",
				Props: MyTestProps{
					Label:    "VarHello",
					Disabled: true,
					Padding:  8,
					Theme:    "light",
				},
			},
		},
	})

	stg := New()
	stg.Merge("NestedFolder", reg)

	doc := dom.NewDocument()
	container := element.NewBox(doc)

	// Render the stage UI
	kitex.Render(renderUI(stg), container)
	defer kitex.Render(nil, container)

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

	// Helper to find all text elements
	var collectTexts func(dom.Node, *[]string)
	collectTexts = func(n dom.Node, res *[]string) {
		if txt, ok := n.(*element.TextElement); ok {
			*res = append(*res, txt.TextContent())
		}
		for child := range n.ChildNodes() {
			collectTexts(child, res)
		}
	}
	var textContents []string
	collectTexts(container, &textContents)
	t.Logf("Rendered text nodes: %v", textContents)

	// Verify the default scene rendered correctly with the default props values
	// Our initial values: "InitialHello", true, 5, "dark"
	// So text should be "InitialHello-true-5-dark"
	if txt := findText(container, "InitialHello-true-5-dark"); txt == nil {
		t.Fatal("expected default scene to render initial props text")
	}

	// Verify the controls labels are rendered correctly (with the override)
	if txt := findText(container, "Override Label"); txt == nil {
		t.Error("expected 'Override Label' control label to be rendered")
	}
	if txt := findText(container, "Disabled"); txt == nil {
		t.Error("expected 'Disabled' control label to be rendered")
	}
	if txt := findText(container, "Padding"); txt == nil {
		t.Error("expected 'Padding' control label to be rendered")
	}
	if txt := findText(container, "Theme"); txt == nil {
		t.Error("expected 'Theme' control label to be rendered")
	}

	// Find the Variation1 scene item in the sidebar and click it
	var variation1Item dom.Node
	var findVariation1 func(dom.Node)
	findVariation1 = func(n dom.Node) {
		if txt, ok := n.(*element.TextElement); ok && txt.TextContent() == "▪ Variation1" {
			variation1Item = n.Parent()
			return
		}
		for child := range n.ChildNodes() {
			findVariation1(child)
			if variation1Item != nil {
				return
			}
		}
	}
	findVariation1(container)
	if variation1Item == nil {
		t.Fatal("expected to find scene item '▪ Variation1' in sidebar")
	}

	// Click the variation scene
	el, ok := variation1Item.EventTarget().(element.Element)
	if !ok {
		t.Fatal("expected variation1Item to be an element.Element")
	}
	t.Logf("variation1Item type: %T, tag: %s, Listeners: %v", el, el.TagName(), el.Listeners())
	el.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))

	// Re-collect texts
	textContents = nil
	collectTexts(container, &textContents)
	t.Logf("Rendered text nodes after click: %v", textContents)

	// Verify that the Canvas text node has updated to the Variation1 props values:
	// "VarHello-true-8-light"
	if txt := findText(container, "VarHello-true-8-light"); txt == nil {
		t.Fatal("expected scene to render Variation1 props text after clicking it")
	}
}
