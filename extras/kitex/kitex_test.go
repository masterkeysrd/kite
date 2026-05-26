package kitex

import (
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

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
}

func TestInstantiateAndUpdate(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("Button instantiation and update", func(t *testing.T) {
		btnProps := ButtonProps{
			ID:       "btn1",
			Class:    "btn-class",
			Style:    style.Style{Bold: style.Some(true)},
			Disabled: true,
			Active:   true,
		}
		btnNode := Button(btnProps)
		realNode := btnNode.Instantiate(doc)

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
		if !btnEl.RawStyle().Bold.UnwrapOr(false) {
			t.Errorf("expected bold style to be true")
		}

		// Perform update
		newBtnProps := ButtonProps{
			ID:       "btn2",
			Class:    "btn-class-new",
			Style:    style.Style{Bold: style.Some(false)},
			Disabled: false,
			Active:   false,
		}
		newBtnNode := Button(newBtnProps)
		newBtnNode.Update(realNode, btnNode)

		if btnEl.ID() != "btn2" {
			t.Errorf("expected updated ID btn2, got %s", btnEl.ID())
		}
		if btnEl.Class() != "btn-class-new" {
			t.Errorf("expected updated class btn-class-new, got %s", btnEl.Class())
		}
		if btnEl.IsDisabled() {
			t.Errorf("expected button to be enabled after update")
		}
		if btnEl.RawStyle().Bold.UnwrapOr(true) {
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
		realNode := cbNode.Instantiate(doc)

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
		newCbNode.Update(realNode, cbNode)

		if !cbEl.Checked() {
			t.Errorf("expected checkbox to be checked after update")
		}
	})

	t.Run("Input and TextArea instantiation and update", func(t *testing.T) {
		inpProps := InputProps{
			ID:    "inp1",
			Value: "initial input",
		}
		inpNode := Input(inpProps)
		realInp := inpNode.Instantiate(doc).(*element.InputElement)

		if realInp.Value() != "initial input" {
			t.Errorf("expected value 'initial input', got %s", realInp.Value())
		}

		newInpProps := InputProps{
			ID:    "inp1",
			Value: "updated input",
		}
		newInpNode := Input(newInpProps)
		newInpNode.Update(realInp, inpNode)

		if realInp.Value() != "updated input" {
			t.Errorf("expected updated value 'updated input', got %s", realInp.Value())
		}

		// TextArea
		txaProps := TextAreaProps{
			ID:    "txa1",
			Value: "initial text",
		}
		txaNode := TextArea(txaProps)
		realTxa := txaNode.Instantiate(doc).(*element.TextAreaElement)

		if realTxa.Value() != "initial text" {
			t.Errorf("expected value 'initial text', got %s", realTxa.Value())
		}

		newTxaProps := TextAreaProps{
			ID:    "txa1",
			Value: "updated text",
		}
		newTxaNode := TextArea(newTxaProps)
		newTxaNode.Update(realTxa, txaNode)

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
		realRadio := rNode.Instantiate(doc).(*element.RadioElement)

		if realRadio.Value() != "val1" {
			t.Errorf("expected value val1, got %s", realRadio.Value())
		}

		// Update value via reflection
		newRProps := RadioProps{
			ID:    "r1",
			Value: "val2",
		}
		newRNode := Radio(newRProps)
		newRNode.Update(realRadio, rNode)

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
		realOpt := optNode.Instantiate(doc).(*element.OptionElement)

		// Reflection updates text/value
		newOptProps := OptionProps{
			ID:    "opt1",
			Text:  "Option 1 New",
			Value: "val1_new",
		}
		newOptNode := Option(newOptProps)
		newOptNode.Update(realOpt, optNode)

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
		realSel := selNode.Instantiate(doc).(*element.SelectElement)

		if realSel.Value() != "val1" {
			t.Errorf("expected select value val1, got %s", realSel.Value())
		}

		// Update select value
		newSelProps := SelectProps{
			ID:    "sel1",
			Value: "val2",
		}
		newSelNode := Select(newSelProps)
		newSelNode.Update(realSel, selNode)

		if realSel.Value() != "val2" {
			t.Errorf("expected updated select value val2, got %s", realSel.Value())
		}
	})

	t.Run("Table, TD, TR, THead, TBody, TFoot instantiation and update", func(t *testing.T) {
		tdProps := TDProps{
			ID:      "td1",
			ColSpan: 2,
			RowSpan: 3,
		}
		tdNode := TD(tdProps)
		realTd := tdNode.Instantiate(doc).(*element.TableCellElement)

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
		newTdNode.Update(realTd, tdNode)

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
			Placement: layout.PlacementTop,
			Flip:      true,
		}
		overlayNode := Overlay(overlayProps, nil)
		realOverlay := overlayNode.Instantiate(doc).(*element.OverlayElement)

		if realOverlay.Placement() != layout.PlacementTop {
			t.Errorf("expected overlay placement top")
		}

		newOverlayProps := OverlayProps{
			ID:        "over1",
			ZIndex:    20,
			Placement: layout.PlacementBottom,
			Flip:      false,
		}
		newOverlayNode := Overlay(newOverlayProps, nil)
		newOverlayNode.Update(realOverlay, overlayNode)

		if realOverlay.Placement() != layout.PlacementBottom {
			t.Errorf("expected updated overlay placement bottom")
		}

		// Dialog
		dialogProps := DialogProps{
			ID:     "dial1",
			ZIndex: 50,
		}
		dialogNode := Dialog(dialogProps, nil)
		realDialog := dialogNode.Instantiate(doc).(*element.DialogElement)

		newDialogProps := DialogProps{
			ID:     "dial1",
			ZIndex: 100,
		}
		newDialogNode := Dialog(newDialogProps, nil)
		newDialogNode.Update(realDialog, dialogNode)

		dialVal := reflect.ValueOf(realDialog).Elem()
		if dialVal.FieldByName("zIndex").Int() != 100 {
			t.Errorf("expected dialog zIndex field to be updated to 100")
		}
	})
}

func TestEventListenersUpdate(t *testing.T) {
	doc := dom.NewDocument()

	var clickCount1, clickCount2 int
	fn1 := func(e event.Event) { clickCount1++ }
	fn2 := func(e event.Event) { clickCount2++ }

	btnNode1 := Button(ButtonProps{
		OnClick: fn1,
	})
	realBtn := btnNode1.Instantiate(doc).(*element.ButtonElement)

	// Trigger click on realBtn
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))
	if clickCount1 != 1 {
		t.Errorf("expected clickCount1 to be 1, got %d", clickCount1)
	}

	// Update to fn2
	btnNode2 := Button(ButtonProps{
		OnClick: fn2,
	})
	btnNode2.Update(realBtn, btnNode1)

	// Trigger click on realBtn again
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))
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
	btnNode3.Update(realBtn, btnNode2)

	// Trigger click on realBtn again
	realBtn.DispatchEvent(event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0))
	if clickCount2 != 1 {
		t.Errorf("expected clickCount2 to stay 1, got %d", clickCount2)
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
	realBox := boxNode.Instantiate(doc)
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
	realBtn := btnNode.Instantiate(doc)
	if btnRef.Current == nil {
		t.Fatalf("expected btnRef.Current to be populated")
	}
	if btnRef.Current != realBtn {
		t.Errorf("expected btnRef.Current to be realBtn, got %v", btnRef.Current)
	}

	// 3. Test Ref with reconciler Render()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)
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

	t.Run("SimpleFC", func(t *testing.T) {
		simple := SimpleFC("Simple", func() Node {
			return Box(BoxProps{ID: "simple-box"})
		})
		node := simple()
		realNode := node.Instantiate(doc).(*element.BoxElement)
		if realNode.ID() != "simple-box" {
			t.Errorf("expected ID simple-box, got %s", realNode.ID())
		}
	})

	t.Run("SimpleFCC", func(t *testing.T) {
		simpleWithChildren := SimpleFCC("SimpleChildren", func(children []Node) Node {
			return Box(BoxProps{ID: "container"}, children...)
		})
		child := Text("hello")
		nodeWithChildren := simpleWithChildren(child)
		realContainer := nodeWithChildren.Instantiate(doc).(*element.BoxElement)
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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	type MyProps struct {
		Val string
	}

	MyComp := FC("MyComp", func(props MyProps) Node {
		state, _ := UseState(props.Val)
		return Box(BoxProps{ID: "child-box"}, Text(state()))
	})

	Render(MyComp(MyProps{Val: "test-vdom"}), container)
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
