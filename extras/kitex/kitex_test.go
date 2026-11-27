package kitex

import (
	"image/color"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

func instOne(n Node, doc dom.Document) dom.Node {
	reals := n.Instantiate(doc)
	if len(reals) == 0 {
		return nil
	}
	return reals[0]
}

func upd(n Node, el dom.Node, old Node) {
	n.Update([]dom.Node{el}, old)
}

func TestFactoryFunctions(t *testing.T) {
	// Test Text node factory
	textNode := Text("hello")
	if textNode.TagName() != "#text" {
		t.Errorf("expected tag name #text, got %s", textNode.TagName())
	}
	if textNode.Props().(string) != "hello" {
		t.Errorf("expected content hello, got %v", textNode.Props())
	}
	if len(textNode.Children()) != 0 {
		t.Errorf("expected 0 children for text node, got %d", len(textNode.Children()))
	}

	// Test Element node factory (Button)
	btnProps := ButtonProps{
		ID:       "btn1",
		Class:    "primary",
		Disabled: true,
	}
	childText := Text("click me")
	btnNode := Button(btnProps, childText)

	if btnNode.TagName() != "button" {
		t.Errorf("expected tag name button, got %s", btnNode.TagName())
	}
	if btnNode.Props().(ButtonProps).ID != "btn1" {
		t.Errorf("expected ID btn1, got %s", btnNode.Props().(ButtonProps).ID)
	}
	if len(btnNode.Children()) != 1 {
		t.Errorf("expected 1 child for button, got %d", len(btnNode.Children()))
	}
	if btnNode.Children()[0] != childText {
		t.Errorf("unexpected child node in button")
	}

	// Test Box node factory
	boxProps := BoxProps{
		ID: "box1",
	}
	boxNode := Box(boxProps)
	if boxNode.TagName() != "box" {
		t.Errorf("expected tag name box, got %s", boxNode.TagName())
	}

	// Test list node factories
	ulNode := UL(ULProps{ID: "list1"}, LI(LIProps{}, Text("item")))
	if ulNode.TagName() != "ul" {
		t.Errorf("expected tag name ul, got %s", ulNode.TagName())
	}
	if ulNode.Props().(ULProps).ID != "list1" {
		t.Errorf("expected ID list1, got %s", ulNode.Props().(ULProps).ID)
	}
	if len(ulNode.Children()) != 1 {
		t.Errorf("expected 1 child for ul, got %d", len(ulNode.Children()))
	}
	if ulNode.Children()[0].TagName() != "li" {
		t.Errorf("expected child tag li, got %s", ulNode.Children()[0].TagName())
	}
}

func TestInstantiateAndUpdate(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("Button instantiation and update", func(t *testing.T) {
		btnProps := ButtonProps{
			ID:       "btn1",
			Class:    "btn-class",
			Style:    style.S().Bold(true),
			Disabled: true,
			Active:   true,
		}
		btnNode := Button(btnProps)
		realNode := instOne(btnNode, doc)

		btnEl, ok := realNode.(*element.ButtonElement)
		if !ok {
			t.Fatalf("expected real node to be *element.ButtonElement, got %T", realNode)
		}

		if btnEl.ID() != "btn1" {
			t.Errorf("expected ID btn1, got %s", btnEl.ID())
		}
		if btnEl.Class() != "btn-class" {
			t.Errorf("expected class btn-class, got %s", btnEl.Class())
		}
		if !btnEl.IsDisabled() {
			t.Errorf("expected button to be disabled")
		}
		if !btnEl.RawStyle().BoldOpt().UnwrapOr(false) {
			t.Errorf("expected bold style to be true")
		}

		// Perform update
		newBtnProps := ButtonProps{
			ID:       "btn2",
			Class:    "btn-class-new",
			Style:    style.S().Bold(false),
			Disabled: false,
			Active:   false,
		}
		newBtnNode := Button(newBtnProps)
		upd(newBtnNode, realNode, btnNode)

		if btnEl.ID() != "btn2" {
			t.Errorf("expected updated ID btn2, got %s", btnEl.ID())
		}
		if btnEl.Class() != "btn-class-new" {
			t.Errorf("expected updated class btn-class-new, got %s", btnEl.Class())
		}
		if btnEl.IsDisabled() {
			t.Errorf("expected button to be enabled after update")
		}
		if btnEl.RawStyle().BoldOpt().UnwrapOr(true) {
			t.Errorf("expected bold style to be false after update")
		}
	})

	t.Run("Checkbox instantiation and update", func(t *testing.T) {
		cbProps := CheckboxProps{
			ID:             "cb1",
			Checked:        false,
			UncheckedGlyph: "[off]",
			CheckedGlyph:   "[on]",
		}
		cbNode := Checkbox(cbProps)
		realNode := instOne(cbNode, doc)

		cbEl, ok := realNode.(*element.CheckboxElement)
		if !ok {
			t.Fatalf("expected *element.CheckboxElement, got %T", realNode)
		}

		if cbEl.Checked() {
			t.Errorf("expected checkbox to be unchecked")
		}

		// Update checked state and glyphs
		newCbProps := CheckboxProps{
			ID:             "cb1",
			Checked:        true,
			UncheckedGlyph: "[off]",
			CheckedGlyph:   "[on]",
		}
		newCbNode := Checkbox(newCbProps)
		upd(newCbNode, realNode, cbNode)

		if !cbEl.Checked() {
			t.Errorf("expected checkbox to be checked after update")
		}
	})

	t.Run("List instantiation and update", func(t *testing.T) {
		ulProps := ULProps{
			ID:    "list1",
			Class: "items",
			Style: style.S().ListStyleType(style.ListStyleSquare),
		}
		ulNode := UL(ulProps, LI(LIProps{}, Text("one")))
		realNode := instOne(ulNode, doc)

		ulEl, ok := realNode.(*element.UnorderedListElement)
		if !ok {
			t.Fatalf("expected *element.UnorderedListElement, got %T", realNode)
		}

		if ulEl.ID() != "list1" {
			t.Errorf("expected ID list1, got %s", ulEl.ID())
		}
		if ulEl.Class() != "items" {
			t.Errorf("expected class items, got %s", ulEl.Class())
		}
		if ulEl.RawStyle().ListStyleTypeOpt().Value() != style.ListStyleSquare {
			t.Errorf("expected list style square, got %v", ulEl.RawStyle().ListStyleTypeOpt().Value())
		}

		newULProps := ULProps{
			ID:    "list2",
			Class: "items-updated",
			Style: style.S().ListStyleType(style.ListStyleCircle),
		}
		newULNode := UL(newULProps, LI(LIProps{}, Text("one")))
		upd(newULNode, realNode, ulNode)

		if ulEl.ID() != "list2" {
			t.Errorf("expected updated ID list2, got %s", ulEl.ID())
		}
		if ulEl.Class() != "items-updated" {
			t.Errorf("expected updated class items-updated, got %s", ulEl.Class())
		}
		if ulEl.RawStyle().ListStyleTypeOpt().Value() != style.ListStyleCircle {
			t.Errorf("expected updated list style circle, got %v", ulEl.RawStyle().ListStyleTypeOpt().Value())
		}
	})

	t.Run("Input and TextArea instantiation and update", func(t *testing.T) {
		inpProps := InputProps{
			ID:    "inp1",
			Value: "initial input",
		}
		inpNode := Input(inpProps)
		realInp := instOne(inpNode, doc).(*element.InputElement)

		if realInp.Value() != "initial input" {
			t.Errorf("expected value 'initial input', got %s", realInp.Value())
		}

		newInpProps := InputProps{
			ID:    "inp1",
			Value: "updated input",
		}
		newInpNode := Input(newInpProps)
		upd(newInpNode, realInp, inpNode)

		if realInp.Value() != "updated input" {
			t.Errorf("expected updated value 'updated input', got %s", realInp.Value())
		}

		// TextArea
		txaProps := TextAreaProps{
			ID:    "txa1",
			Value: "initial text",
		}
		txaNode := TextArea(txaProps)
		realTxa := instOne(txaNode, doc).(*element.TextAreaElement)

		if realTxa.Value() != "initial text" {
			t.Errorf("expected value 'initial text', got %s", realTxa.Value())
		}

		newTxaProps := TextAreaProps{
			ID:    "txa1",
			Value: "updated text",
		}
		newTxaNode := TextArea(newTxaProps)
		upd(newTxaNode, realTxa, txaNode)

		if realTxa.Value() != "updated text" {
			t.Errorf("expected updated value 'updated text', got %s", realTxa.Value())
		}
	})

	t.Run("Radio and RadioGroup instantiation and update", func(t *testing.T) {
		rProps := RadioProps{
			ID:    "r1",
			Value: "val1",
		}
		rNode := Radio(rProps)
		realRadio := instOne(rNode, doc).(*element.RadioElement)

		if realRadio.Value() != "val1" {
			t.Errorf("expected value val1, got %s", realRadio.Value())
		}

		// Update value via reflection
		newRProps := RadioProps{
			ID:    "r1",
			Value: "val2",
		}
		newRNode := Radio(newRProps)
		upd(newRNode, realRadio, rNode)

		if realRadio.Value() != "val2" {
			t.Errorf("expected updated value val2, got %s", realRadio.Value())
		}
	})

	t.Run("Select and Option instantiation and update", func(t *testing.T) {
		optProps := OptionProps{
			ID:    "opt1",
			Text:  "Option 1",
			Value: "val1",
		}
		optNode := Option(optProps)
		realOpt := instOne(optNode, doc).(*element.OptionElement)

		// Reflection updates text/value
		newOptProps := OptionProps{
			ID:    "opt1",
			Text:  "Option 1 New",
			Value: "val1_new",
		}
		newOptNode := Option(newOptProps)
		upd(newOptNode, realOpt, optNode)

		// Check unexported fields using reflection in test
		optVal := reflect.ValueOf(realOpt).Elem()
		if optVal.FieldByName("text").String() != "Option 1 New" {
			t.Errorf("expected text option field to update")
		}
		if optVal.FieldByName("value").String() != "val1_new" {
			t.Errorf("expected value option field to update")
		}

		// SelectElement
		selProps := SelectProps{
			ID:    "sel1",
			Value: "val1",
		}
		selNode := Select(selProps)
		realSel := instOne(selNode, doc).(*element.SelectElement)

		if realSel.Value() != "val1" {
			t.Errorf("expected select value val1, got %s", realSel.Value())
		}

		// Update select value
		newSelProps := SelectProps{
			ID:    "sel1",
			Value: "val2",
		}
		newSelNode := Select(newSelProps)
		upd(newSelNode, realSel, selNode)

		if realSel.Value() != "val2" {
			t.Errorf("expected updated select value val2, got %s", realSel.Value())
		}
	})

	t.Run("Select VDOM children option synchronization", func(t *testing.T) {
		container := instOne(Div(BoxProps{}), doc).(dom.Element)

		selNode := Select(SelectProps{
			Name:  "role",
			Value: "admin",
		},
			Option(OptionProps{Text: "Administrator", Value: "admin"}),
			Option(OptionProps{Text: "User", Value: "user"}),
		)

		Render(selNode, container)

		realSel := container.FirstChild().(*element.SelectElement)

		uaRoot := dom.UARoot(realSel).(dom.Element)
		triggerBtn := uaRoot.FirstChild().(*element.ButtonElement)

		btnText := triggerBtn.TextContent()
		if btnText != "Administrator▼" {
			t.Errorf("expected button text to be 'Administrator▼', got %q (this means select options were not synchronized)", btnText)
		}
	})

	t.Run("Table, TD, TR, THead, TBody, TFoot instantiation and update", func(t *testing.T) {
		tdProps := TDProps{
			ID:      "td1",
			ColSpan: 2,
			RowSpan: 3,
		}
		tdNode := TD(tdProps)
		realTd := instOne(tdNode, doc).(*element.TableCellElement)

		if realTd.ColSpan() != 2 {
			t.Errorf("expected colSpan 2, got %d", realTd.ColSpan())
		}
		if realTd.RowSpan() != 3 {
			t.Errorf("expected rowSpan 3, got %d", realTd.RowSpan())
		}

		newTdProps := TDProps{
			ID:      "td1",
			ColSpan: 1,
			RowSpan: 1,
		}
		newTdNode := TD(newTdProps)
		upd(newTdNode, realTd, tdNode)

		if realTd.ColSpan() != 1 {
			t.Errorf("expected updated colSpan 1, got %d", realTd.ColSpan())
		}
		if realTd.RowSpan() != 1 {
			t.Errorf("expected updated rowSpan 1, got %d", realTd.RowSpan())
		}
	})

	t.Run("Overlay and Dialog instantiation and update", func(t *testing.T) {
		overlayProps := OverlayProps{
			ID:        "over1",
			ZIndex:    10,
			Placement: geom.PlacementTop,
			Flip:      true,
		}
		overlayNode := Overlay(overlayProps, nil)
		realOverlay := instOne(overlayNode, doc).(*element.OverlayElement)

		if realOverlay.Placement() != geom.PlacementTop {
			t.Errorf("expected overlay placement top")
		}

		newOverlayProps := OverlayProps{
			ID:        "over1",
			ZIndex:    20,
			Placement: geom.PlacementBottom,
			Flip:      false,
		}
		newOverlayNode := Overlay(newOverlayProps, nil)
		upd(newOverlayNode, realOverlay, overlayNode)

		if realOverlay.Placement() != geom.PlacementBottom {
			t.Errorf("expected updated overlay placement bottom")
		}

		// Dialog
		dialogProps := DialogProps{
			ID:     "dial1",
			ZIndex: 50,
		}
		dialogNode := Dialog(dialogProps, nil)
		realDialog := instOne(dialogNode, doc).(*element.DialogElement)

		newDialogProps := DialogProps{
			ID:     "dial1",
			ZIndex: 100,
		}
		newDialogNode := Dialog(newDialogProps, nil)
		upd(newDialogNode, realDialog, dialogNode)

		dialVal := reflect.ValueOf(realDialog).Elem()
		if dialVal.FieldByName("zIndex").Int() != 100 {
			t.Errorf("expected dialog zIndex field to be updated to 100")
		}
	})
}

func TestEventListenersUpdate(t *testing.T) {
	doc := dom.NewDocument()

	var clickCount1, clickCount2 int
	fn1 := func(e *event.MouseEvent) { clickCount1++ }
	fn2 := func(e *event.MouseEvent) { clickCount2++ }

	btnNode1 := Button(ButtonProps{
		OnClick: fn1,
	})
	realBtn := instOne(btnNode1, doc).(*element.ButtonElement)

	// Trigger click on realBtn
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if clickCount1 != 1 {
		t.Errorf("expected clickCount1 to be 1, got %d", clickCount1)
	}

	// Update to fn2
	btnNode2 := Button(ButtonProps{
		OnClick: fn2,
	})
	upd(btnNode2, realBtn, btnNode1)

	// Trigger click on realBtn again
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if clickCount1 != 1 {
		t.Errorf("expected clickCount1 to stay 1, got %d", clickCount1)
	}
	if clickCount2 != 1 {
		t.Errorf("expected clickCount2 to be 1, got %d", clickCount2)
	}

	// Update to nil (remove listener)
	btnNode3 := Button(ButtonProps{
		OnClick: nil,
	})
	upd(btnNode3, realBtn, btnNode2)

	// Trigger click on realBtn again
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if clickCount2 != 1 {
		t.Errorf("expected clickCount2 to stay 1, got %d", clickCount2)
	}

	// Clear all subscriptions
	ClearAllSubscriptions(realBtn)
}

func TestEventListenersClosureUpdate(t *testing.T) {
	doc := dom.NewDocument()

	var calledWithVal int
	makeBtn := func(val int) Node {
		return Button(ButtonProps{
			OnClick: func(e *event.MouseEvent) {
				calledWithVal = val
			},
		})
	}

	btnNode1 := makeBtn(10)
	realBtn := instOne(btnNode1, doc).(*element.ButtonElement)

	// Trigger click on realBtn
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 10 {
		t.Errorf("expected calledWithVal to be 10, got %d", calledWithVal)
	}

	// Update to a new button rendering with a new closure capturing 20
	btnNode2 := makeBtn(20)
	upd(btnNode2, realBtn, btnNode1)

	// Trigger click on realBtn again
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 20 {
		t.Errorf("expected calledWithVal to be updated to 20, got %d (closure was not updated)", calledWithVal)
	}

	// Clear all subscriptions
	ClearAllSubscriptions(realBtn)
}

func TestElementRefWiring(t *testing.T) {
	doc := dom.NewDocument()

	// 1. Test Ref with Box (uses BoxProps = ElementProps)
	boxRef := CreateRef[dom.Element]()
	boxNode := Box(BoxProps{
		Ref: boxRef,
		ID:  "my-box",
	})
	realBox := instOne(boxNode, doc)
	if boxRef.Current == nil {
		t.Fatalf("expected boxRef.Current to be populated")
	}
	if boxRef.Current != realBox {
		t.Errorf("expected boxRef.Current to be realBox, got %v", boxRef.Current)
	}

	// 2. Test Ref with Button (uses custom ButtonProps struct)
	btnRef := CreateRef[*element.ButtonElement]()
	btnNode := Button(ButtonProps{
		Ref: btnRef,
		ID:  "my-btn",
	})
	realBtn := instOne(btnNode, doc)
	if btnRef.Current == nil {
		t.Fatalf("expected btnRef.Current to be populated")
	}
	if btnRef.Current != realBtn {
		t.Errorf("expected btnRef.Current to be realBtn, got %v", btnRef.Current)
	}

	// 3. Test Ref with reconciler Render()
	container := instOne(Div(BoxProps{}), doc).(dom.Element)
	reconcileRef := CreateRef[dom.Element]()

	Render(Box(BoxProps{
		Ref: reconcileRef,
		ID:  "reconciled-box",
	}), container)

	if reconcileRef.Current == nil {
		t.Fatalf("expected reconcileRef.Current to be populated after Render")
	}
	realReconciledBox := container.FirstChild().(dom.Element)
	if reconcileRef.Current != realReconciledBox {
		t.Errorf("expected reconcileRef.Current to match container first child, got %v", reconcileRef.Current)
	}
}

func TestSimpleComponents(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("FC with NoProps", func(t *testing.T) {
		simple := FC("Simple", func(_ NoProps) Node {
			return Box(BoxProps{ID: "simple-box"})
		})
		node := simple()
		realNode := instOne(node, doc).(*element.BoxElement)
		if realNode.ID() != "simple-box" {
			t.Errorf("expected ID simple-box, got %s", realNode.ID())
		}
	})

	t.Run("FCC with NoProps", func(t *testing.T) {
		simpleWithChildren := FCC("SimpleChildren", func(_ NoProps, children ...Node) Node {
			return Box(BoxProps{ID: "container"}, children...)
		})
		child := Text("hello")
		nodeWithChildren := simpleWithChildren(child)
		realContainer := instOne(nodeWithChildren, doc).(*element.BoxElement)
		if realContainer.ID() != "container" {
			t.Errorf("expected ID container, got %s", realContainer.ID())
		}
		if !realContainer.HasChildNodes() {
			t.Errorf("expected children, but found none")
		}
	})
}

func TestBuildDevToolsSnapshot(t *testing.T) {
	oldRoots := activeRoots
	activeRoots = make(map[dom.Element]Node)
	defer func() { activeRoots = oldRoots }()

	EnableDevMode = true
	defer func() { EnableDevMode = false }()

	doc := dom.NewDocument()
	container := instOne(Div(BoxProps{}), doc).(dom.Element)

	type MyProps struct {
		Val string
	}

	MyComp := FC("MyComp", func(props MyProps) Node {
		state, _ := UseState(props.Val)
		return Box(BoxProps{ID: "child-box"}, Text(state()))
	})

	Render(MyComp.Props(MyProps{Val: "test-vdom"}), container)
	defer Render(nil, container)

	snapshot := BuildDevToolsSnapshot(nil)
	roots, ok := snapshot.([]*VDOMSnapshot)
	if !ok {
		t.Fatalf("expected slice of VDOMSnapshot, got %T", snapshot)
	}

	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}

	root := roots[0]
	if root.Name != "MyComp" {
		t.Errorf("expected root name MyComp, got %s", root.Name)
	}

	propsMap, ok := root.Props.(map[string]any)
	if !ok {
		t.Fatalf("expected props to be map[string]any, got %T", root.Props)
	}
	if propsMap["Val"] != "test-vdom" {
		t.Errorf("expected prop Val to be 'test-vdom', got %v", propsMap["Val"])
	}

	if len(root.State) != 1 {
		t.Errorf("expected 1 state hook, got %d", len(root.State))
	} else {
		stateVal := root.State[0]
		if stateVal != "test-vdom" {
			t.Errorf("expected state hook value to be 'test-vdom', got %v", stateVal)
		}
	}

	if root.DeclFile == "" {
		t.Errorf("expected DeclFile to be populated")
	}
	if root.DeclLine == 0 {
		t.Errorf("expected DeclLine to be non-zero")
	}
	if root.InstFile == "" {
		t.Errorf("expected InstFile to be populated")
	}
	if root.InstLine == 0 {
		t.Errorf("expected InstLine to be non-zero")
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Name != "box" {
		t.Errorf("expected child name box, got %s", child.Name)
	}
	if child.DomID != "child-box" {
		t.Errorf("expected child DomID child-box, got %s", child.DomID)
	}
}

func TestComplexityTextNode(t *testing.T) {
	n := &textNode{content: "hello"}
	if got := n.complexity(); got != 1 {
		t.Errorf("textNode.complexity() = %d, want 1", got)
	}
}

// TestComplexityElementNodeLeaf verifies that a leaf element has score 1.
func TestComplexityElementNodeLeaf(t *testing.T) {
	n := Box(BoxProps{})
	ni := n.(nodeInternal)
	if got := ni.complexity(); got != 1 {
		t.Errorf("leaf Box complexity = %d, want 1", got)
	}
}

// TestComplexityElementNodeWithChildren verifies that parent accumulates children scores.
func TestComplexityElementNodeWithChildren(t *testing.T) {
	// Box(1) + Text(1) + Text(1) = 3
	n := Box(BoxProps{}, Text("a"), Text("b"))
	ni := n.(nodeInternal)
	if got := ni.complexity(); got != 3 {
		t.Errorf("Box with 2 texts: complexity = %d, want 3", got)
	}
}

// TestComplexityNilChildrenIgnored verifies nil children contribute 0 to score.
func TestComplexityNilChildrenIgnored(t *testing.T) {
	// Box(1) + nil + Text(1) = 2
	n := Box(BoxProps{}, nil, Text("x"))
	ni := n.(nodeInternal)
	if got := ni.complexity(); got != 2 {
		t.Errorf("Box with nil + text: complexity = %d, want 2", got)
	}
}

// TestComplexityDeepNesting verifies bottom-up accumulation for deeply nested trees.
func TestComplexityDeepNesting(t *testing.T) {
	// Box(1) + Box(1+Text(1)) + Box(1+Text(1)+Text(1)) = 1+2+3 = 6
	inner1 := Box(BoxProps{}, Text("a"))            // score=2
	inner2 := Box(BoxProps{}, Text("b"), Text("c")) // score=3
	outer := Box(BoxProps{}, inner1, inner2)        // score=1+2+3=6
	ni := outer.(nodeInternal)
	if got := ni.complexity(); got != 6 {
		t.Errorf("nested Box complexity = %d, want 6", got)
	}
}

// TestComponentNodeComplexityAfterInstantiate verifies that ComponentNode.complexity()
// reflects the rendered subtree after Instantiate.
func TestComponentNodeComplexityAfterInstantiate(t *testing.T) {
	doc := dom.NewDocument()

	// Render 6 text nodes → complexity = Box(1) + 6*Text(1) = 7
	myComp := FC("richComp", func(props NoProps) Node {
		return Box(BoxProps{},
			Text("1"), Text("2"), Text("3"),
			Text("4"), Text("5"), Text("6"),
		)
	})

	node := myComp()
	_ = node.Instantiate(doc)

	comp := node.(*ComponentNode[struct{}])
	if comp.complexityScore != 7 {
		t.Errorf("complexityScore = %d, want 7", comp.complexityScore)
	}
	if !comp.shouldMemo {
		t.Errorf("shouldMemo should be true when complexityScore=%d > threshold=%d",
			comp.complexityScore, memoComplexityThreshold)
	}
}

// TestComponentNodeComplexityBelowThreshold verifies shouldMemo is false for small trees.
func TestComponentNodeComplexityBelowThreshold(t *testing.T) {
	doc := dom.NewDocument()

	myComp := FC("smallComp", func(props NoProps) Node {
		return Box(BoxProps{}, Text("hi")) // score=2
	})

	node := myComp()
	_ = node.Instantiate(doc)

	comp := node.(*ComponentNode[struct{}])
	if comp.complexityScore != 2 {
		t.Errorf("complexityScore = %d, want 2", comp.complexityScore)
	}
	if comp.shouldMemo {
		t.Errorf("shouldMemo should be false when complexityScore=%d <= threshold=%d",
			comp.complexityScore, memoComplexityThreshold)
	}
}

// --- deepEqualProps tests -----------------------------------------------------

// TestDeepEqualPropsMaxDepth verifies that the function returns false when depth=0.
func TestDeepEqualPropsMaxDepth(t *testing.T) {
	type P struct{ V int }
	p := P{V: 1}
	if deepEqualProps(p, p, 0) {
		t.Errorf("deepEqualProps at maxDepth=0 should return false")
	}
}

// TestDeepEqualPropsMaxDepthNested verifies that deeply nested structs beyond the
// maxDepth limit return false conservatively.
func TestDeepEqualPropsMaxDepthNested(t *testing.T) {
	type Inner struct{ X int }
	type Mid struct{ I Inner }
	type Outer struct{ M Mid }

	a := Outer{M: Mid{I: Inner{X: 42}}}
	b := Outer{M: Mid{I: Inner{X: 42}}}

	// maxDepth=1: descends into Outer but cannot fully compare Inner → false.
	if deepEqualProps(a, b, 1) {
		t.Errorf("deepEqualProps with maxDepth=1 should return false for deeply nested equal structs")
	}
	// maxDepth=3: can fully compare → true.
	if !deepEqualProps(a, b, 3) {
		t.Errorf("deepEqualProps with maxDepth=3 should return true for equal nested structs")
	}
}

// TestDeepEqualPropsSimpleEqual verifies basic scalar equality.
func TestDeepEqualPropsSimpleEqual(t *testing.T) {
	type P struct {
		A string
		B int
		C bool
	}
	p := P{A: "hello", B: 42, C: true}
	if !deepEqualProps(p, p, 3) {
		t.Errorf("identical scalar structs should be equal")
	}
}

// TestDeepEqualPropsSimpleNotEqual verifies basic scalar inequality.
func TestDeepEqualPropsSimpleNotEqual(t *testing.T) {
	type P struct{ V string }
	if deepEqualProps(P{V: "a"}, P{V: "b"}, 3) {
		t.Errorf("different scalar structs should not be equal")
	}
}

// TestDeepEqualPropsNilBothNil verifies nil == nil.
func TestDeepEqualPropsNilBothNil(t *testing.T) {
	if !deepEqualProps(nil, nil, 3) {
		t.Errorf("nil == nil should be true")
	}
}

// TestDeepEqualPropsOneNil verifies nil != non-nil.
func TestDeepEqualPropsOneNil(t *testing.T) {
	if deepEqualProps(nil, struct{}{}, 3) {
		t.Errorf("nil should not equal non-nil")
	}
}

// TestDeepEqualPropsFuncField verifies that func fields are compared by pointer.
func TestDeepEqualPropsFuncField(t *testing.T) {
	fn := func() {}
	type P struct{ Fn func() }
	// Same function pointer → equal.
	if !deepEqualProps(P{Fn: fn}, P{Fn: fn}, 3) {
		t.Errorf("same func pointer should be equal")
	}
	// Different function pointers → not equal.
	fn2 := func() {}
	if deepEqualProps(P{Fn: fn}, P{Fn: fn2}, 3) {
		t.Errorf("different func pointers should not be equal")
	}
	// Nil func == nil func → equal.
	if !deepEqualProps(P{Fn: nil}, P{Fn: nil}, 3) {
		t.Errorf("both nil funcs should be equal")
	}
}

// TestDeepEqualPropsSlice verifies slice comparison.
func TestDeepEqualPropsSlice(t *testing.T) {
	type P struct{ Items []int }
	if !deepEqualProps(P{Items: []int{1, 2, 3}}, P{Items: []int{1, 2, 3}}, 3) {
		t.Errorf("equal int slices should be equal")
	}
	if deepEqualProps(P{Items: []int{1, 2}}, P{Items: []int{1, 3}}, 3) {
		t.Errorf("different int slices should not be equal")
	}
	if deepEqualProps(P{Items: nil}, P{Items: []int{}}, 3) {
		t.Errorf("nil slice != empty slice")
	}
}

// TestDeepEqualPropsStyle verifies that style.Style comparison works correctly.
func TestDeepEqualPropsStyle(t *testing.T) {
	type P struct{ Style style.Style }

	// 1. Empty styles are equal
	if !deepEqualProps(P{Style: style.Style{}}, P{Style: style.Style{}}, 3) {
		t.Errorf("empty styles should be equal")
	}

	// 2. Different foreground colors are not equal
	s1 := style.Style{}.Foreground(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	s2 := style.Style{}.Foreground(color.RGBA{R: 0, G: 255, B: 0, A: 255})
	if deepEqualProps(P{Style: s1}, P{Style: s2}, 3) {
		t.Errorf("styles with different foreground colors should not be equal")
	}

	// 3. Same foreground colors are equal
	s3 := style.Style{}.Foreground(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	if !deepEqualProps(P{Style: s1}, P{Style: s3}, 3) {
		t.Errorf("styles with same foreground colors should be equal")
	}
}

type customEquatableProps struct {
	Value int
	Name  string
}

func (p customEquatableProps) Equal(other any) bool {
	o, ok := other.(customEquatableProps)
	return ok && p.Value == o.Value && p.Name == o.Name
}

func TestDeepEqualPropsEquatable(t *testing.T) {
	p1 := customEquatableProps{Value: 10, Name: "A"}
	p2 := customEquatableProps{Value: 10, Name: "A"}
	p3 := customEquatableProps{Value: 20, Name: "A"}
	p4 := customEquatableProps{Value: 10, Name: "B"}

	if !deepEqualProps(p1, p2, 3) {
		t.Errorf("expected p1 and p2 to be equal")
	}
	if deepEqualProps(p1, p3, 3) {
		t.Errorf("expected p1 and p3 to not be equal")
	}
	if deepEqualProps(p1, p4, 3) {
		t.Errorf("expected p1 and p4 to not be equal")
	}
}

// --- Automatic memoization integration tests ----------------------------------

// TestMemoSkipsRenderOnEqualProps verifies that Update() skips RenderFn when
// shouldMemo is true and props are equal.
func TestMemoSkipsRenderOnEqualProps(t *testing.T) {
	doc := dom.NewDocument()
	renderCount := 0

	type RichProps struct {
		Title string
	}

	myComp := FC("RichComp", func(props RichProps) Node {
		renderCount++
		// Build a subtree with score > 5 so shouldMemo activates.
		return Box(BoxProps{},
			Text("1"), Text("2"), Text("3"),
			Text("4"), Text("5"), Text("6"),
		)
	})

	// Initial render (renderCount = 1)
	node1 := myComp.Props(RichProps{Title: "hello"})
	realNode := instOne(node1, doc)
	if renderCount != 1 {
		t.Fatalf("expected renderCount=1 after Instantiate, got %d", renderCount)
	}

	comp1 := node1.(*ComponentNode[RichProps])
	if !comp1.shouldMemo {
		t.Fatalf("shouldMemo should be true (score=%d)", comp1.complexityScore)
	}

	oldRendered := comp1.rendered

	// Update with identical props → memoization should kick in, no RenderFn call.
	node2 := myComp.Props(RichProps{Title: "hello"})
	upd(node2, realNode, node1)
	if renderCount != 1 {
		t.Errorf("expected renderCount=1 after memo hit, got %d (RenderFn was called unexpectedly)", renderCount)
	}

	comp2 := node2.(*ComponentNode[RichProps])
	if comp2.rendered != oldRendered {
		t.Errorf("memoized component should reuse the old rendered node")
	}

	// Update with changed props → RenderFn must be called.
	node3 := myComp.Props(RichProps{Title: "world"})
	upd(node3, realNode, node2)
	if renderCount != 2 {
		t.Errorf("expected renderCount=2 after prop change, got %d", renderCount)
	}
}

// TestMemoDoesNotActivateForSmallTree verifies that small trees always re-render.
func TestMemoDoesNotActivateForSmallTree(t *testing.T) {
	doc := dom.NewDocument()
	renderCount := 0

	type P struct{ V string }

	myComp := FC("SmallComp", func(props P) Node {
		renderCount++
		return Text(props.V) // score=1, below threshold
	})

	node1 := myComp.Props(P{V: "a"})
	realNode := instOne(node1, doc)

	comp := node1.(*ComponentNode[P])
	if comp.shouldMemo {
		t.Fatalf("shouldMemo should be false for small trees (score=%d)", comp.complexityScore)
	}

	node2 := myComp.Props(P{V: "a"}) // identical props but small tree
	upd(node2, realNode, node1)
	if renderCount != 2 {
		t.Errorf("small tree should always re-render, expected renderCount=2, got %d", renderCount)
	}
}

// --- UseMemo hook tests -------------------------------------------------------

// TestUseMemoCallsFactoryOnFirstRender ensures factory is called on the initial render.
func TestUseMemoCallsFactoryOnFirstRender(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0

	myComp := FC("MemoComp", func(props NoProps) Node {
		val := UseMemo(func() int {
			callCount++
			return 42
		})
		_ = val
		return Box(BoxProps{})
	})

	node := myComp()
	_ = node.Instantiate(doc)

	if callCount != 1 {
		t.Errorf("expected factory to be called once on first render, got %d", callCount)
	}
}

// TestUseMemoReturnsCachedValueWhenDepsUnchanged verifies factory is NOT called again
// when deps are the same between renders.
func TestUseMemoReturnsCachedValueWhenDepsUnchanged(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0
	var lastVal int

	type P struct{ Dep int }

	myComp := FC("MemoComp", func(props P) Node {
		val := UseMemo(func() int {
			callCount++
			return props.Dep * 10
		}, props.Dep)
		lastVal = val
		return Box(BoxProps{})
	})

	// Initial render: dep=5
	node1 := myComp.Props(P{Dep: 5})
	realNode := instOne(node1, doc)
	if callCount != 1 || lastVal != 50 {
		t.Fatalf("initial render: callCount=%d lastVal=%d, want 1 / 50", callCount, lastVal)
	}

	// Re-render with same dep → factory should NOT be called.
	node2 := myComp.Props(P{Dep: 5})
	upd(node2, realNode, node1)
	if callCount != 1 {
		t.Errorf("same deps: factory should not be called again, got callCount=%d", callCount)
	}
	if lastVal != 50 {
		t.Errorf("expected cached value 50, got %d", lastVal)
	}

	// Re-render with changed dep → factory MUST be called.
	node3 := myComp.Props(P{Dep: 7})
	upd(node3, realNode, node2)
	if callCount != 2 {
		t.Errorf("changed deps: factory should be called, got callCount=%d", callCount)
	}
	if lastVal != 70 {
		t.Errorf("expected new value 70, got %d", lastVal)
	}
}

// TestUseMemoMultipleDeps verifies that all deps are checked; changing any one triggers re-eval.
func TestUseMemoMultipleDeps(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0

	type P struct {
		A int
		B string
	}

	myComp := FC("MultiDep", func(props P) Node {
		_ = UseMemo(func() string {
			callCount++
			return "computed"
		}, props.A, props.B)
		return Box(BoxProps{})
	})

	node1 := myComp.Props(P{A: 1, B: "x"})
	realNode := instOne(node1, doc)
	if callCount != 1 {
		t.Fatalf("initial: callCount=%d, want 1", callCount)
	}

	// Same deps → no re-eval.
	node2 := myComp.Props(P{A: 1, B: "x"})
	upd(node2, realNode, node1)
	if callCount != 1 {
		t.Errorf("same deps: callCount=%d, want 1", callCount)
	}

	// Changing B only → re-eval.
	node3 := myComp.Props(P{A: 1, B: "y"})
	upd(node3, realNode, node2)
	if callCount != 2 {
		t.Errorf("changed B: callCount=%d, want 2", callCount)
	}
}

// TestUseMemoEmptyDeps verifies that an empty dep slice never re-triggers the factory
// (equivalent to "run once" semantics).
func TestUseMemoEmptyDeps(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0

	myComp := FC("EmptyDeps", func(props NoProps) Node {
		_ = UseMemo(func() int {
			callCount++
			return 99
		})
		return Box(BoxProps{})
	})

	node1 := myComp()
	realNode := instOne(node1, doc)
	if callCount != 1 {
		t.Fatalf("initial: callCount=%d, want 1", callCount)
	}

	node2 := myComp()
	upd(node2, realNode, node1)
	node3 := myComp()
	upd(node3, realNode, node2)
	if callCount != 1 {
		t.Errorf("empty deps: factory should run only once, got callCount=%d", callCount)
	}
}

// TestUseMemoNilDepsSliceLength verifies nil and empty slices are not equal (length differs).
func TestUseMemoNilVsEmptyDeps(t *testing.T) {
	// nil and []any{} have different lengths (0 == 0), so they are equal by depsEqual.
	if !depsEqual(nil, []any{}) {
		t.Errorf("nil and empty slice should be equal by depsEqual (both length 0)")
	}
}

// TestUseMemoOutsideRenderPanics verifies that UseMemo panics when called outside a render cycle.
func TestUseMemoOutsideRenderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected UseMemo to panic when called outside a component render phase")
		}
	}()
	UseMemo(func() int { return 0 })
}

// BenchmarkVDOMConstruction measures the performance of constructing a standard VDOM tree.
func BenchmarkVDOMConstruction(b *testing.B) {
	EnableDevMode = false

	for b.Loop() {
		_ = Box(BoxProps{
			ID:    "container",
			Class: "wrapper",
		},
			Button(ButtonProps{
				ID:       "btn-ok",
				Disabled: true,
			}, Text("OK")),
			Button(ButtonProps{
				ID: "btn-cancel",
			}, Text("Cancel")),
		)
	}
}

// BenchmarkComponentInstantiation measures functional component rendering and instantiation overhead.
func BenchmarkComponentInstantiation(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()

	type MyButtonProps struct {
		Label string
	}

	myButton := FC("MyButton", func(props MyButtonProps) Node {
		return Button(ButtonProps{
			Class: "fancy-btn",
		}, Text(props.Label))
	})

	for b.Loop() {
		compNode := myButton.Props(MyButtonProps{Label: "click me"})
		_ = compNode.Instantiate(doc)
	}
}

// BenchmarkComponentHooks measures the performance of rendering a component that uses UseState hooks.
func BenchmarkComponentHooks(b *testing.B) {
	doc := dom.NewDocument()

	myCounter := FC("Counter", func(props NoProps) Node {
		getVal1, setVal1 := UseState(0)
		getVal2, setVal2 := UseState("hello")

		if getVal1() == 0 {
			setVal1(1)
		}
		if getVal2() == "hello" {
			setVal2("world")
		}

		return Box(BoxProps{})
	})

	for b.Loop() {
		compNode := myCounter()
		_ = compNode.Instantiate(doc)
	}
}

// BenchmarkComponentUpdate measures the performance of calling Update on a component with hooks.
func BenchmarkComponentUpdate(b *testing.B) {
	doc := dom.NewDocument()

	myCounter := FC("Counter", func(props NoProps) Node {
		getVal, setVal := UseState(0)
		if getVal() < 100 {
			setVal(getVal() + 1)
		}
		return Box(BoxProps{})
	})

	// Initial render
	node1 := myCounter()
	realNode := instOne(node1, doc)

	for b.Loop() {
		node2 := myCounter()
		upd(node2, realNode, node1)
		node1 = node2
	}
}

// --- Memoization benchmarks ---------------------------------------------------

// buildRichTree creates a component that renders a deeply nested tree with score > 5.
// This simulates a realistic "complex" component that would benefit from memoization.
func buildRichTree(label string) Node {
	return Box(BoxProps{},
		Span(SpanProps{}, Text(label)),
		Box(BoxProps{},
			Text("row1-col1"), Text("row1-col2"),
		),
		Box(BoxProps{},
			Text("row2-col1"), Text("row2-col2"),
		),
	)
}

// BenchmarkMemoizedUpdateIdenticalProps measures the cost of Update() when
// memoization fires — i.e., shouldMemo=true and props are deeply equal.
// In this path the RenderFn is skipped entirely; only a deepEqualProps reflection
// walk is performed.
//
// Performance delta vs BenchmarkMemoizedUpdateChangedProps (below) documents the
// savings that automatic memoization delivers on complex subtrees.
func BenchmarkMemoizedUpdateIdenticalProps(b *testing.B) {
	doc := dom.NewDocument()

	type RichProps struct{ Label string }

	myComp := FC("RichComp", func(props RichProps) Node {
		return buildRichTree(props.Label)
	})

	node1 := myComp.Props(RichProps{Label: "hello"})
	realNode := instOne(node1, doc)

	// Confirm memoization is active.
	comp := node1.(*ComponentNode[RichProps])
	if !comp.shouldMemo {
		b.Fatalf("shouldMemo must be true for this benchmark (score=%d)", comp.complexityScore)
	}

	for b.Loop() {
		node2 := myComp.Props(RichProps{Label: "hello"}) // identical props
		upd(node2, realNode, node1)
		node1 = node2
	}
}

// BenchmarkMemoizedUpdateChangedProps measures the cost of Update() when props
// differ and the RenderFn must execute. This is the baseline (no memo saving).
func BenchmarkMemoizedUpdateChangedProps(b *testing.B) {
	doc := dom.NewDocument()

	type RichProps struct{ Label string }

	myComp := FC("RichComp", func(props RichProps) Node {
		return buildRichTree(props.Label)
	})

	node1 := myComp.Props(RichProps{Label: "hello"})
	realNode := instOne(node1, doc)

	for i := 0; b.Loop(); i++ {
		// Alternate labels so props always differ, defeating the memo.
		label := "hello"
		if i%2 == 0 {
			label = "world"
		}
		node2 := myComp.Props(RichProps{Label: label})
		upd(node2, realNode, node1)
		ReleaseTree(node1)
		node1 = node2
	}
}

// BenchmarkUseMemoHitRate measures how fast UseMemo returns a cached value
// when deps do not change between renders (the common hot path).
func BenchmarkUseMemoHitRate(b *testing.B) {
	doc := dom.NewDocument()

	myComp := FC("UseMemoComp", func(props NoProps) Node {
		_ = UseMemo(func() []int {
			// Simulated "expensive" computation.
			out := make([]int, 100)
			for i := range out {
				out[i] = i * i
			}
			return out
		}, "stable-key")
		return Box(BoxProps{})
	})

	node1 := myComp()
	realNode := instOne(node1, doc)

	for b.Loop() {
		node2 := myComp()
		upd(node2, realNode, node1)
		node1 = node2
	}
}

func TestInput_ProgrammaticClearFocusReconciliation(t *testing.T) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

	type State struct {
		value string
	}
	state := &State{value: "hello"}

	App := FC("App", func(_ NoProps) Node {
		return Input(InputProps{
			ID:    "inp1",
			Value: state.value,
		})
	})

	container := doc.CreateElement("container", nil)
	doc.AppendChild(container)
	Render(App(), container)
	testScheduler.flushMacrotasks()

	realInp := container.FirstChild().(*element.InputElement)

	// Focus the input
	fm.Focus(realInp)
	if fm.Current() != realInp {
		t.Fatalf("expected input to be focused, got %v", fm.Current())
	}

	// Change state value to empty (clear programmatically) and re-render
	state.value = ""
	Render(App(), container)
	testScheduler.flushMacrotasks()

	// Verify it remains focused
	if fm.Current() != realInp {
		t.Errorf("after programmatic VDOM clear: focused element = %v, want input", fm.Current())
	}
}

func TestVDOMAttributes(t *testing.T) {
	doc := dom.NewDocument()

	// Initial render with attributes
	boxNode := Box(BoxProps{
		Attributes: map[string]string{
			"type": "button",
			"name": "submit-btn",
		},
	})
	realNode := instOne(boxNode, doc)
	if realNode == nil {
		t.Fatal("expected real DOM node")
	}
	el, ok := realNode.(dom.Element)
	if !ok {
		t.Fatal("expected real DOM element")
	}

	val, ok := el.Attribute("type")
	if !ok || val != "button" {
		t.Errorf("expected attribute 'type' to be 'button', got %q, ok=%t", val, ok)
	}

	val, ok = el.Attribute("name")
	if !ok || val != "submit-btn" {
		t.Errorf("expected attribute 'name' to be 'submit-btn', got %q, ok=%t", val, ok)
	}

	// Update attributes (modify one, delete one, add one)
	newBoxNode := Box(BoxProps{
		Attributes: map[string]string{
			"type":  "submit",
			"value": "go",
		},
	})
	upd(newBoxNode, realNode, boxNode)

	// 'type' should be updated to 'submit'
	val, ok = el.Attribute("type")
	if !ok || val != "submit" {
		t.Errorf("expected attribute 'type' to be updated to 'submit', got %q, ok=%t", val, ok)
	}

	// 'name' should be removed
	if el.HasAttribute("name") {
		t.Error("expected attribute 'name' to be removed")
	}

	// 'value' should be added
	val, ok = el.Attribute("value")
	if !ok || val != "go" {
		t.Errorf("expected attribute 'value' to be 'go', got %q, ok=%t", val, ok)
	}
}

func TestInputAndTextAreaCursorPreservation(t *testing.T) {
	doc := dom.NewDocument()

	// 1. Input Cursor Preservation Test
	inpProps := InputProps{
		ID:    "inp1",
		Value: "hello",
	}
	inpNode := Input(inpProps)
	realInp := instOne(inpNode, doc).(*element.InputElement)

	// Set selection/cursor position in the middle
	realInp.SetSelectionRange(2, 2)
	start, end := realInp.SelectionRange()
	if start != 2 || end != 2 {
		t.Fatalf("expected selection range (2, 2), got (%d, %d)", start, end)
	}

	// Update with the same value
	newInpProps := InputProps{
		ID:    "inp1",
		Value: "hello",
	}
	newInpNode := Input(newInpProps)
	upd(newInpNode, realInp, inpNode)

	// Cursor position should be preserved (not reset to 5)
	start, end = realInp.SelectionRange()
	if start != 2 || end != 2 {
		t.Errorf("expected input selection range (2, 2) to be preserved, but got (%d, %d)", start, end)
	}

	// 2. TextArea Cursor Preservation Test
	txaProps := TextAreaProps{
		ID:    "txa1",
		Value: "hello",
	}
	txaNode := TextArea(txaProps)
	realTxa := instOne(txaNode, doc).(*element.TextAreaElement)

	// Set selection/cursor position in the middle
	realTxa.SetSelectionRange(2, 2)
	start, end = realTxa.SelectionRange()
	if start != 2 || end != 2 {
		t.Fatalf("expected selection range (2, 2), got (%d, %d)", start, end)
	}

	// Update with the same value
	newTxaProps := TextAreaProps{
		ID:    "txa1",
		Value: "hello",
	}
	newTxaNode := TextArea(newTxaProps)
	upd(newTxaNode, realTxa, txaNode)

	// Cursor position should be preserved (not reset to 5)
	start, end = realTxa.SelectionRange()
	if start != 2 || end != 2 {
		t.Errorf("expected textarea selection range (2, 2) to be preserved, but got (%d, %d)", start, end)
	}
}

func TestAttributesForwardingOnAllProps(t *testing.T) {
	doc := dom.NewDocument()
	attrs := map[string]string{"data-test": "ok", "aria-label": "lbl"}

	nodes := []Node{
		Button(ButtonProps{Attributes: attrs}),
		Checkbox(CheckboxProps{Attributes: attrs}),
		RadioGroup(RadioGroupProps{Attributes: attrs}),
		Radio(RadioProps{Attributes: attrs}),
		Select(SelectProps{Attributes: attrs}),
		Option(OptionProps{Attributes: attrs}),
		Input(InputProps{Attributes: attrs}),
		TextArea(TextAreaProps{Attributes: attrs}),
		TD(TDProps{Attributes: attrs}),
		Overlay(OverlayProps{Attributes: attrs}, nil),
		Dialog(DialogProps{Attributes: attrs}, nil),
		Form(FormProps{Attributes: attrs}),
	}

	for _, n := range nodes {
		el := instOne(n, doc).(dom.Element)
		if got, ok := el.Attribute("data-test"); !ok || got != "ok" {
			t.Errorf("%s dropped data-test attribute: got (%q, %v)", n.TagName(), got, ok)
		}
		if got, ok := el.Attribute("aria-label"); !ok || got != "lbl" {
			t.Errorf("%s dropped aria-label attribute: got (%q, %v)", n.TagName(), got, ok)
		}
	}
}

func TestControlledPropParity_AndTypedEvents(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("Button OnPress and OnClick coexist", func(t *testing.T) {
		clicks := 0
		presses := 0
		btnNode := Button(ButtonProps{
			OnClick: func(e *event.MouseEvent) { clicks++ },
			OnPress: func() { presses++ },
		})
		btn := instOne(btnNode, doc).(*element.ButtonElement)
		btn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
		btn.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: key.KeyEnter}))
		if clicks != 2 || presses != 2 {
			t.Fatalf("expected clicks=2 presses=2, got clicks=%d presses=%d", clicks, presses)
		}

		// Updating OnClick must not drop OnPress.
		btnNode2 := Button(ButtonProps{
			OnClick: nil,
			OnPress: btnNode.Props().(ButtonProps).OnPress,
		})
		upd(btnNode2, btn, btnNode)
		btn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
		if clicks != 2 || presses != 3 {
			t.Fatalf("expected clicks=2 presses=3 after removing OnClick, got clicks=%d presses=%d", clicks, presses)
		}
	})

	t.Run("Input OnValueChange and OnInput", func(t *testing.T) {
		var vals []string
		var inputs []string
		inpNode := Input(InputProps{
			Value:         "a",
			OnValueChange: func(v string) { vals = append(vals, v) },
			OnInput:       func(e *event.InputEvent) { inputs = append(inputs, e.Value) },
		})
		inp := instOne(inpNode, doc).(*element.InputElement)
		inp.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'b', Text: "b"}))
		if len(vals) != 1 || vals[0] != "ab" {
			t.Fatalf("expected OnValueChange [\"ab\"], got %v", vals)
		}
		if len(inputs) != 1 || inputs[0] != "ab" {
			t.Fatalf("expected OnInput [\"ab\"], got %v", inputs)
		}
	})

	t.Run("TextArea OnValueChange and OnInput", func(t *testing.T) {
		var vals []string
		var inputs []string
		txaNode := TextArea(TextAreaProps{
			Value:         "x",
			OnValueChange: func(v string) { vals = append(vals, v) },
			OnInput:       func(e *event.InputEvent) { inputs = append(inputs, e.Value) },
		})
		txa := instOne(txaNode, doc).(*element.TextAreaElement)
		txa.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: 'y', Text: "y"}))
		if len(vals) != 1 || vals[0] != "xy" {
			t.Fatalf("expected OnValueChange [\"xy\"], got %v", vals)
		}
		if len(inputs) != 1 || inputs[0] != "xy" {
			t.Fatalf("expected OnInput [\"xy\"], got %v", inputs)
		}
	})

	t.Run("Checkbox OnCheckedChange", func(t *testing.T) {
		var states []bool
		cbNode := Checkbox(CheckboxProps{
			Checked:         false,
			OnCheckedChange: func(checked bool) { states = append(states, checked) },
		})
		cb := instOne(cbNode, doc).(*element.CheckboxElement)
		cb.DispatchEvent(event.NewKeyEvent(event.EventKeyDown, key.Key{Code: ' '}))
		if len(states) != 1 || states[0] != true {
			t.Fatalf("expected OnCheckedChange [true], got %v", states)
		}
	})
}

func TestTextf(t *testing.T) {
	EnableDevMode = true
	defer func() { EnableDevMode = false }()

	n := Textf("count=%d name=%s", 42, "kite")
	if n.Props() != "count=42 name=kite" {
		t.Fatalf("Textf Props() = %q, want %q", n.Props(), "count=42 name=kite")
	}
	if st, ok := n.(sourceTracker); ok {
		declFile, declLine, _, _ := st.getSource()
		if declFile == "" || declLine == 0 {
			t.Errorf("expected Textf DevMode source tracking to populate declFile/declLine")
		}
	}
}

func TestComponentAndContainerAPI(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("Component zero-props and explicit props", func(t *testing.T) {
		type LabelProps struct {
			Text string
		}
		Label := FC("Label", func(p LabelProps) Node {
			if p.Text == "" {
				return Text("default")
			}
			return Text(p.Text)
		})

		elZero := instOne(Label(), doc).(*element.TextElement)
		if elZero.Data() != "default" {
			t.Fatalf("Label() = %q, want %q", elZero.Data(), "default")
		}

		elExplicit := instOne(Label.Props(LabelProps{Text: "custom"}), doc).(*element.TextElement)
		if elExplicit.Data() != "custom" {
			t.Fatalf("Label.Props(...) = %q, want %q", elExplicit.Data(), "custom")
		}
	})

	t.Run("Container zero-props and explicit props", func(t *testing.T) {
		type PanelProps struct {
			ID string
		}
		Panel := FCC("Panel", func(p PanelProps, children ...Node) Node {
			id := p.ID
			if id == "" {
				id = "default-panel"
			}
			return Box(BoxProps{ID: id}, children...)
		})

		countChildren := func(el dom.Element) int {
			n := 0
			for range el.ChildNodes() {
				n++
			}
			return n
		}

		elZero := instOne(Panel(Text("a"), Text("b")), doc).(*element.BoxElement)
		if elZero.ID() != "default-panel" || countChildren(elZero) != 2 {
			t.Fatalf("Panel(...) got id=%q children=%d", elZero.ID(), countChildren(elZero))
		}

		elExplicit := instOne(Panel.Props(PanelProps{ID: "custom-panel"}, Text("x")), doc).(*element.BoxElement)
		if elExplicit.ID() != "custom-panel" || countChildren(elExplicit) != 1 {
			t.Fatalf("Panel.Props(...) got id=%q children=%d", elExplicit.ID(), countChildren(elExplicit))
		}
	})

	t.Run("Nil receiver guards", func(t *testing.T) {
		var nilComp Component[int]
		if n := nilComp.Props(42); n == nil || n.TagName() != "#empty" {
			t.Fatalf("nilComp.Props() = %v, want Empty()", n)
		}

		var nilCont Container[int]
		if n := nilCont.Props(42, Text("x")); n == nil || n.TagName() != "#empty" {
			t.Fatalf("nilCont.Props() = %v, want Empty()", n)
		}
	})
}

func TestDevToolsSourceLines_FC_And_FCC(t *testing.T) {
	EnableDevMode = true
	defer func() { EnableDevMode = false }()

	type P struct{ V string }
	_, _, expectedFCDeclLine, _ := runtime.Caller(0)
	Comp := FC("LineComp", func(p P) Node {
		return Text(p.V)
	})
	_, _, expectedFCCDeclLine, _ := runtime.Caller(0)
	Cont := FCC("LineCont", func(p P, children ...Node) Node {
		return Box(BoxProps{ID: p.V}, children...)
	})

	_, _, expectedCompCallLine, _ := runtime.Caller(0)
	n1 := Comp()
	_, _, expectedCompPropsLine, _ := runtime.Caller(0)
	n2 := Comp.Props(P{V: "hi"})
	_, _, expectedContCallLine, _ := runtime.Caller(0)
	n3 := Cont(Text("c1"))
	_, _, expectedContPropsLine, _ := runtime.Caller(0)
	n4 := Cont.Props(P{V: "id"}, Text("c2"))
	_, _, expectedBoxStyleLine, _ := runtime.Caller(0)
	n5 := Box.Style(style.S().Bold(true), Text("styled"))
	_, _, expectedInputStyleLine, _ := runtime.Caller(0)
	n6 := Input.Style(style.S().Padding(0, 1))

	cases := []struct {
		name          string
		node          Node
		checkDeclLine bool
		declLine      int
		instLine      int
	}{
		{"Comp()", n1, true, expectedFCDeclLine + 1, expectedCompCallLine + 1},
		{"Comp.Props()", n2, true, expectedFCDeclLine + 1, expectedCompPropsLine + 1},
		{"Cont()", n3, true, expectedFCCDeclLine + 1, expectedContCallLine + 1},
		{"Cont.Props()", n4, true, expectedFCCDeclLine + 1, expectedContPropsLine + 1},
		{"Box.Style()", n5, false, 0, expectedBoxStyleLine + 1},
		{"Input.Style()", n6, false, 0, expectedInputStyleLine + 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.node.(sourceTracker)
			declFile, declLine, instFile, instLine := st.getSource()
			if declFile == "" || instFile == "" {
				t.Fatalf("expected non-empty files, got decl=%q inst=%q", declFile, instFile)
			}
			if tc.checkDeclLine && declLine != tc.declLine {
				t.Errorf("declLine = %d, want %d", declLine, tc.declLine)
			}
			if instLine != tc.instLine {
				t.Errorf("instLine = %d, want %d", instLine, tc.instLine)
			}
		})
	}
}

func TestZeroReflectionKeyExtraction(t *testing.T) {
	t.Run("Direct Key field", func(t *testing.T) {
		type P struct {
			Pad int
			Key string
		}
		comp := FC("DirectKey", func(p P) Node { return Text(p.Key) })
		if got := comp.Props(P{Pad: 1, Key: "k-direct"}).Key(); got != "k-direct" {
			t.Fatalf("Key() = %q, want %q", got, "k-direct")
		}
	})

	t.Run("Embedded ElementProps Key", func(t *testing.T) {
		type P struct {
			Pad int
			ElementProps
		}
		comp := FC("EmbeddedKey", func(p P) Node { return Text(p.Key) })
		if got := comp.Props(P{Pad: 1, ElementProps: ElementProps{Key: "k-embed"}}).Key(); got != "k-embed" {
			t.Fatalf("Key() = %q, want %q", got, "k-embed")
		}
	})

	t.Run("Nested ElementProps field", func(t *testing.T) {
		type P struct {
			Pad          int
			ElementProps ElementProps
		}
		comp := FC("NestedKey", func(p P) Node { return Text(p.ElementProps.Key) })
		if got := comp.Props(P{Pad: 1, ElementProps: ElementProps{Key: "k-nested"}}).Key(); got != "k-nested" {
			t.Fatalf("Key() = %q, want %q", got, "k-nested")
		}
	})

	t.Run("Pointer struct props", func(t *testing.T) {
		type P struct {
			Key string
		}
		comp := FC("PtrKey", func(p *P) Node { return Box(BoxProps{}) })
		if got := comp.Props(nil).Key(); got != "" {
			t.Fatalf("nil ptr Key() = %q, want empty", got)
		}
		if got := comp.Props(&P{Key: "k-ptr"}).Key(); got != "k-ptr" {
			t.Fatalf("ptr Key() = %q, want %q", got, "k-ptr")
		}
	})

	t.Run("No Key field", func(t *testing.T) {
		comp := FC("NoKey", func(_ NoProps) Node { return Box(BoxProps{}) })
		if got := comp().Key(); got != "" {
			t.Fatalf("NoProps Key() = %q, want empty", got)
		}
	})
}

func TestIsSafelyComparable_AndMemoSafety(t *testing.T) {
	type SafeNested struct {
		A int
		B [2]string
	}
	type SafeProps struct {
		N SafeNested
		S string
	}
	type InterfaceFieldProps struct {
		Data any
	}
	type NestedInterfaceProps struct {
		Inner struct {
			Items [1]any
		}
	}
	type StyleFieldProps struct {
		Style style.Style
	}
	type NonComparableProps struct {
		Items []int
	}

	if !isSafelyComparable(reflect.TypeFor[NoProps]()) {
		t.Error("expected NoProps to be safely comparable")
	}
	if !isSafelyComparable(reflect.TypeFor[SafeProps]()) {
		t.Error("expected SafeProps to be safely comparable")
	}
	if isSafelyComparable(reflect.TypeFor[InterfaceFieldProps]()) {
		t.Error("expected struct with interface field NOT to be safely comparable")
	}
	if isSafelyComparable(reflect.TypeFor[NestedInterfaceProps]()) {
		t.Error("expected struct with nested array of interface NOT to be safely comparable")
	}
	if isSafelyComparable(reflect.TypeFor[StyleFieldProps]()) {
		t.Error("expected struct with style.Style field NOT to be safely comparable")
	}
	if isSafelyComparable(reflect.TypeFor[NonComparableProps]()) {
		t.Error("expected struct with slice field NOT to be safely comparable")
	}
	if isSafelyComparable(reflect.TypeFor[customEquatableProps]()) {
		t.Error("expected EquatableProps implementor NOT to bypass Equal()")
	}

	// Verify an interface holding an uncomparable slice never panics during memoized Update.
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	comp := FC("IfaceMemo", func(p InterfaceFieldProps) Node {
		return Box(BoxProps{},
			Text("1"), Text("2"), Text("3"),
			Text("4"), Text("5"), Text("6"),
		)
	})

	Render(comp.Props(InterfaceFieldProps{Data: []int{1, 2}}), container)
	Render(comp.Props(InterfaceFieldProps{Data: []int{1, 2}}), container)
}

func TestFCC_PassedChildrenBypassMemoWhenPropsUnchanged(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	doc.AppendChild(container)
	defer Render(nil, container)

	type CardProps struct {
		Title string
	}

	renderCount := 0
	Card := FCC("Card", func(p CardProps, children ...Node) Node {
		renderCount++
		// Make subtree score > memoComplexityThreshold so shouldMemo activates.
		return Box(BoxProps{ID: "card"},
			Text(p.Title),
			Text("1"), Text("2"), Text("3"), Text("4"), Text("5"),
			Box(BoxProps{ID: "body"}, children...),
		)
	})

	Render(Card.Props(CardProps{Title: "Header"}, Text("first")), container)
	if renderCount != 1 {
		t.Fatalf("expected 1 initial render, got %d", renderCount)
	}
	body := doc.GetElementByID("body")
	if body == nil || body.FirstChild().TextContent() != "first" {
		t.Fatalf("expected initial child text 'first'")
	}

	// Props are identical, but PassedChildren changed -> must NOT skip render!
	Render(Card.Props(CardProps{Title: "Header"}, Text("second")), container)
	if renderCount != 2 {
		t.Fatalf("expected renderCount=2 when PassedChildren exist and props unchanged, got %d", renderCount)
	}
	body = doc.GetElementByID("body")
	if body == nil || body.FirstChild().TextContent() != "second" {
		t.Fatalf("expected updated child text 'second', got %q", body.FirstChild().TextContent())
	}
}

func TestComponentRegistry_GCCleanup(t *testing.T) {
	var regKey uintptr
	func() {
		local := FC("Ephemeral", func(_ NoProps) Node {
			return Text("tmp")
		})
		regKey = *(*uintptr)(unsafe.Pointer(&local))
		if _, ok := componentRegistry.Load(regKey); !ok {
			t.Fatalf("expected componentRegistry to contain key %#x while local is live", regKey)
		}
		runtime.KeepAlive(local)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		if _, ok := componentRegistry.Load(regKey); !ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected componentRegistry entry %#x to be cleaned up after GC", regKey)
}

func BenchmarkFC_NoProps(b *testing.B) {
	EnableDevMode = false
	comp := FC("NoPropsComp", func(_ NoProps) Node {
		return Box(BoxProps{ID: "box"}, Text("ok"))
	})

	b.ReportAllocs()
	for b.Loop() {
		_ = comp()
	}
}

func BenchmarkFC_WithProps(b *testing.B) {
	EnableDevMode = false
	type BenchProps struct {
		Key   string
		Label string
	}
	comp := FC("PropsComp", func(p BenchProps) Node {
		return Box(BoxProps{ID: p.Label}, Text(p.Label))
	})

	p := BenchProps{Key: "k1", Label: "hello"}
	b.ReportAllocs()
	for b.Loop() {
		_ = comp.Props(p)
	}
}

func BenchmarkFCC_WithPropsAndChildren(b *testing.B) {
	EnableDevMode = false
	type BenchProps struct {
		Key   string
		Title string
	}
	comp := FCC("ContainerComp", func(p BenchProps, children ...Node) Node {
		return Box(BoxProps{ID: p.Title}, children...)
	})

	p := BenchProps{Key: "k1", Title: "title"}
	c1 := Text("one")
	c2 := Text("two")
	b.ReportAllocs()
	for b.Loop() {
		_ = comp.Props(p, c1, c2)
	}
}

func BenchmarkBox_Props(b *testing.B) {
	EnableDevMode = false
	s := style.S().Padding(1, 2).Bold(true)
	c1 := Text("one")
	c2 := Text("two")
	b.ReportAllocs()
	for b.Loop() {
		_ = Box(BoxProps{Style: s}, c1, c2)
	}
}

func BenchmarkBox_Style(b *testing.B) {
	EnableDevMode = false
	s := style.S().Padding(1, 2).Bold(true)
	c1 := Text("one")
	c2 := Text("two")
	b.ReportAllocs()
	for b.Loop() {
		_ = Box.Style(s, c1, c2)
	}
}
