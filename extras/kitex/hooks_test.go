package kitex

import (
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

type TestProps struct {
	Label string
}

type TestChildrenProps struct {
	Title    string
	Children []Node
}

func TestFCAndFCCConstruction(t *testing.T) {
	// Test FC
	myComp := FC("myComponent", func(props TestProps) Node {
		return Box(BoxProps{
			ID: props.Label,
		})
	})

	node := myComp(TestProps{Label: "hello"})
	compNode, ok := node.(*ComponentNode[TestProps])
	if !ok {
		t.Fatalf("expected *ComponentNode[TestProps], got %T", node)
	}

	if compNode.TagName() != "myComponent" {
		t.Errorf("expected tag name myComponent, got %s", compNode.TagName())
	}
	if compNode.PropsVal.Label != "hello" {
		t.Errorf("expected Label hello, got %s", compNode.PropsVal.Label)
	}

	// Test FCC
	myCompWithChildren := FCC("myChildrenComponent", func(props TestChildrenProps) Node {
		return Box(BoxProps{
			ID: props.Title,
		}, props.Children...)
	})

	child1 := Text("child1")
	child2 := Text("child2")
	nodeWithChildren := myCompWithChildren(TestChildrenProps{Title: "layout"}, child1, child2)

	compNodeWithChildren, ok := nodeWithChildren.(*ComponentNode[TestChildrenProps])
	if !ok {
		t.Fatalf("expected *ComponentNode[TestChildrenProps], got %T", nodeWithChildren)
	}

	if compNodeWithChildren.PropsVal.Title != "layout" {
		t.Errorf("expected Title layout, got %s", compNodeWithChildren.PropsVal.Title)
	}
	children := compNodeWithChildren.PropsVal.Children
	if len(children) != 2 {
		t.Errorf("expected 2 injected children, got %d", len(children))
	}
	if children[0] != child1 || children[1] != child2 {
		t.Errorf("injected children do not match original children")
	}
}

func TestUseStatePersistence(t *testing.T) {
	doc := dom.NewDocument()

	var getState1 func() int
	var setState1 func(int)
	var getState2 func(string) string
	var setState2 func(string)

	renderCount := 0

	myComp := FC("testComp", func(props struct{}) Node {
		renderCount++

		// Multiple hooks to test order-based tracking
		get1, set1 := UseState(10)
		getState1 = get1
		setState1 = set1

		get2, set2 := UseState("initial")
		getState2 = func(_ string) string { return get2() }
		setState2 = set2

		return Box(BoxProps{})
	})

	// 1. Initial instantiate
	node := myComp(struct{}{})
	realNode := node.Instantiate(doc)

	if renderCount != 1 {
		t.Fatalf("expected renderCount to be 1, got %d", renderCount)
	}
	if getState1() != 10 {
		t.Errorf("expected initial state to be 10, got %d", getState1())
	}
	if getState2("") != "initial" {
		t.Errorf("expected initial state to be 'initial', got %s", getState2(""))
	}

	// 2. Simulate update/re-render without state changes
	node2 := myComp(struct{}{})
	node2.Update(realNode, node)

	if renderCount != 2 {
		t.Fatalf("expected renderCount to be 2, got %d", renderCount)
	}
	if getState1() != 10 {
		t.Errorf("expected persisted state to be 10, got %d", getState1())
	}

	// 3. Mutate state
	var dirtyNode Node
	oldDirty := OnComponentDirty
	defer func() {
		OnComponentDirty = oldDirty
	}()
	OnComponentDirty = func(n Node) {
		dirtyNode = n
	}

	// Trigger set
	setState1(42)
	setState2("updated")

	if dirtyNode != node2 {
		t.Errorf("expected dirtyNode callback to be triggered with component node")
	}
	compNode := node2.(*ComponentNode[struct{}])
	if !compNode.dirty {
		t.Errorf("expected component to be marked dirty")
	}

	// In a real reconciler, it would re-render. Let's simulate that re-render:
	node3 := myComp(struct{}{})
	node3.Update(realNode, node2)

	if renderCount != 3 {
		t.Fatalf("expected renderCount to be 3, got %d", renderCount)
	}
	if getState1() != 42 {
		t.Errorf("expected updated state to be 42, got %d", getState1())
	}
	if getState2("") != "updated" {
		t.Errorf("expected updated state to be 'updated', got %s", getState2(""))
	}
}

func TestUseStateMarkDirty(t *testing.T) {
	doc := dom.NewDocument()

	var setState func(int)

	myComp := FC("testSync", func(props struct{}) Node {
		_, set := UseState(0)
		setState = set
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
	_ = node.Instantiate(doc)

	compNode, ok := node.(*ComponentNode[struct{}])
	if !ok {
		t.Fatalf("expected *ComponentNode[struct{}], got %T", node)
	}

	if compNode.IsDirty() {
		t.Errorf("expected component node to be clean initially")
	}

	// Trigger state change
	setState(1)

	// Verify that the ComponentNode is now dirty
	if !compNode.IsDirty() {
		t.Errorf("expected component node to be marked dirty after state update")
	}
}

func TestUseStatePanicOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected UseState to panic when called outside a component render cycle")
		}
	}()

	// Call UseState outside of any functional component rendering context
	UseState(100)
}

func TestUseRef(t *testing.T) {
	doc := dom.NewDocument()

	var myRef Ref[int]
	renderCount := 0

	myComp := FC("testRefComp", func(props struct{}) Node {
		renderCount++
		myRef = UseRef(100)
		return Box(BoxProps{})
	})

	// 1. Initial instantiate
	node1 := myComp(struct{}{})
	realNode := node1.Instantiate(doc)

	if renderCount != 1 {
		t.Fatalf("expected renderCount to be 1, got %d", renderCount)
	}
	if myRef == nil {
		t.Fatalf("expected ref to be initialized")
	}
	if myRef.Current != 100 {
		t.Errorf("expected ref current to be 100, got %d", myRef.Current)
	}

	// 2. Mutate ref current value
	var dirtyCalled bool
	oldDirty := OnComponentDirty
	defer func() {
		OnComponentDirty = oldDirty
	}()
	OnComponentDirty = func(n Node) {
		dirtyCalled = true
	}

	myRef.Current = 200

	compNode := node1.(*ComponentNode[struct{}])
	if compNode.IsDirty() {
		t.Errorf("expected modifying ref value not to mark component dirty")
	}
	if dirtyCalled {
		t.Errorf("expected modifying ref value not to trigger OnComponentDirty callback")
	}

	// 3. Simulate update/re-render and verify persistence
	node2 := myComp(struct{}{})
	node2.Update(realNode, node1)

	if renderCount != 2 {
		t.Fatalf("expected renderCount to be 2, got %d", renderCount)
	}
	if myRef.Current != 200 {
		t.Errorf("expected ref current to persist as 200, got %d", myRef.Current)
	}
}

func TestUseRefPanicOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected UseRef to panic when called outside a component render cycle")
		}
	}()
	UseRef(100)
}

func TestCreateRef(t *testing.T) {
	r := CreateRef[string]()
	if r == nil {
		t.Fatalf("expected CreateRef to return non-nil")
	}
	if r.Current != "" {
		t.Errorf("expected initial Current to be empty, got %s", r.Current)
	}
	r.Current = "test"
	if r.Current != "test" {
		t.Errorf("expected Current to be test, got %s", r.Current)
	}
}

func TestUseReducer_InitialState(t *testing.T) {
	doc := dom.NewDocument()
	var getState func() int

	myComp := FC("testReducerComp", func(props struct{}) Node {
		getState, _ = UseReducer(func(s int, a string) int {
			return s
		}, 42)
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
	_ = node.Instantiate(doc)

	if getState() != 42 {
		t.Errorf("expected initial state 42, got %d", getState())
	}
}

func TestUseReducer_Dispatch(t *testing.T) {
	doc := dom.NewDocument()
	var getState func() int
	var dispatch func(string)

	reducer := func(state int, action string) int {
		switch action {
		case "inc":
			return state + 1
		case "dec":
			return state - 1
		default:
			return state
		}
	}

	myComp := FC("testReducerComp", func(props struct{}) Node {
		getState, dispatch = UseReducer(reducer, 10)
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
	realNode := node.Instantiate(doc)

	dispatch("inc")

	// Simulate re-render
	node2 := myComp(struct{}{})
	node2.Update(realNode, node)

	if getState() != 11 {
		t.Errorf("expected state to be 11 after inc dispatch, got %d", getState())
	}
}

func TestUseReducer_MultipleDispatches(t *testing.T) {
	doc := dom.NewDocument()
	var getState func() int
	var dispatch func(string)

	reducer := func(state int, action string) int {
		switch action {
		case "inc":
			return state + 1
		case "dec":
			return state - 1
		default:
			return state
		}
	}

	myComp := FC("testReducerComp", func(props struct{}) Node {
		getState, dispatch = UseReducer(reducer, 10)
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
	realNode := node.Instantiate(doc)

	dispatch("inc")
	dispatch("inc")
	dispatch("dec")
	dispatch("inc")

	// Simulate re-render
	node2 := myComp(struct{}{})
	node2.Update(realNode, node)

	if getState() != 12 {
		t.Errorf("expected state to be 12 after multiple dispatches, got %d", getState())
	}
}

func TestUseReducer_DispatchTriggersReRender(t *testing.T) {
	doc := dom.NewDocument()
	var dispatch func(string)

	myComp := FC("testReducerComp", func(props struct{}) Node {
		_, dispatch = UseReducer(func(state int, action string) int {
			return state + 1
		}, 0)
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
	_ = node.Instantiate(doc)

	var dirtyNode Node
	oldDirty := OnComponentDirty
	defer func() {
		OnComponentDirty = oldDirty
	}()
	OnComponentDirty = func(n Node) {
		dirtyNode = n
	}

	dispatch("inc")

	if dirtyNode != node {
		t.Errorf("expected OnComponentDirty to be triggered with component node")
	}
	compNode := node.(*ComponentNode[struct{}])
	if !compNode.IsDirty() {
		t.Errorf("expected component node to be marked dirty after dispatch")
	}
}

func TestUseReducer_StableGetterAcrossRenders(t *testing.T) {
	doc := dom.NewDocument()
	var getStates []func() int

	myComp := FC("testReducerComp", func(props struct{}) Node {
		getState, _ := UseReducer(func(state int, action string) int {
			return state
		}, 0)
		getStates = append(getStates, getState)
		return Box(BoxProps{})
	})

	node1 := myComp(struct{}{})
	realNode := node1.Instantiate(doc)

	node2 := myComp(struct{}{})
	node2.Update(realNode, node1)

	if len(getStates) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(getStates))
	}

	ptr1 := reflect.ValueOf(getStates[0]).Pointer()
	ptr2 := reflect.ValueOf(getStates[1]).Pointer()
	if ptr1 != ptr2 {
		t.Errorf("expected stable getState closure across renders, got different pointers")
	}
}

func TestUseCallback_ReturnsSameRef(t *testing.T) {
	doc := dom.NewDocument()
	var callbacks []func()

	func1 := func() {}
	func2 := func() {}

	myComp := FC("testCallbackComp", func(props struct {
		useSecond bool
		x         int
	}) Node {
		var cb func()
		if props.useSecond {
			cb = func2
		} else {
			cb = func1
		}
		cbCached := UseCallback(cb, []any{props.x})
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: false, x: 1})
	realNode := node1.Instantiate(doc)

	node2 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: true, x: 1})
	node2.Update(realNode, node1)

	if len(callbacks) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(callbacks))
	}

	ptr1 := reflect.ValueOf(callbacks[0]).Pointer()
	ptr2 := reflect.ValueOf(callbacks[1]).Pointer()
	if ptr1 != ptr2 {
		t.Errorf("expected same callback reference with same deps, got different pointers")
	}
}

func TestUseCallback_UpdatesOnDepsChange(t *testing.T) {
	doc := dom.NewDocument()
	var callbacks []func()

	func1 := func() {}
	func2 := func() {}

	myComp := FC("testCallbackComp", func(props struct {
		useSecond bool
		x         int
	}) Node {
		var cb func()
		if props.useSecond {
			cb = func2
		} else {
			cb = func1
		}
		cbCached := UseCallback(cb, []any{props.x})
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: false, x: 1})
	realNode := node1.Instantiate(doc)

	node2 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: true, x: 2})
	node2.Update(realNode, node1)

	if len(callbacks) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(callbacks))
	}

	ptr1 := reflect.ValueOf(callbacks[0]).Pointer()
	ptr2 := reflect.ValueOf(callbacks[1]).Pointer()
	if ptr1 == ptr2 {
		t.Errorf("expected different callback references with different deps, got same pointers")
	}
}

func TestUseCallback_NilDeps(t *testing.T) {
	doc := dom.NewDocument()
	var callbacks []func()

	func1 := func() {}
	func2 := func() {}

	myComp := FC("testCallbackComp", func(props struct{ useSecond bool }) Node {
		var cb func()
		if props.useSecond {
			cb = func2
		} else {
			cb = func1
		}
		cbCached := UseCallback(cb, nil)
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp(struct{ useSecond bool }{useSecond: false})
	realNode := node1.Instantiate(doc)

	node2 := myComp(struct{ useSecond bool }{useSecond: true})
	node2.Update(realNode, node1)

	if len(callbacks) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(callbacks))
	}

	ptr1 := reflect.ValueOf(callbacks[0]).Pointer()
	ptr2 := reflect.ValueOf(callbacks[1]).Pointer()
	if ptr1 == ptr2 {
		t.Errorf("expected different callback references with nil deps every render, got same pointer")
	}
}
