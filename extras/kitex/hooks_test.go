package kitex

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
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
	realNodes := node.Instantiate(doc)

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
	node2.Update(realNodes, node)

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
	node3.Update(realNodes, node2)

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
	realNodes := node1.Instantiate(doc)

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
	node2.Update(realNodes, node1)

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
	realNodes := node.Instantiate(doc)

	dispatch("inc")

	// Simulate re-render
	node2 := myComp(struct{}{})
	node2.Update(realNodes, node)

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
	realNodes := node.Instantiate(doc)

	dispatch("inc")
	dispatch("inc")
	dispatch("dec")
	dispatch("inc")

	// Simulate re-render
	node2 := myComp(struct{}{})
	node2.Update(realNodes, node)

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
	realNodes := node1.Instantiate(doc)

	node2 := myComp(struct{}{})
	node2.Update(realNodes, node1)

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
	realNodes := node1.Instantiate(doc)

	node2 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: true, x: 1})
	node2.Update(realNodes, node1)

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
	realNodes := node1.Instantiate(doc)

	node2 := myComp(struct {
		useSecond bool
		x         int
	}{useSecond: true, x: 2})
	node2.Update(realNodes, node1)

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
	realNodes := node1.Instantiate(doc)

	node2 := myComp(struct{ useSecond bool }{useSecond: true})
	node2.Update(realNodes, node1)

	if len(callbacks) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(callbacks))
	}

	ptr1 := reflect.ValueOf(callbacks[0]).Pointer()
	ptr2 := reflect.ValueOf(callbacks[1]).Pointer()
	if ptr1 == ptr2 {
		t.Errorf("expected different callback references with nil deps every render, got same pointer")
	}
}

// BenchmarkContextFlatReconcile measures the reconciliation cost of 100 elements
// wrapped inside a Context.Provider (requiring reconciler-level flattening and context push/pop).
func BenchmarkContextFlatReconcile(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	ctx := CreateContext("value")

	listA := make([]Node, 100)
	for i := range 100 {
		listA[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, ctx.Provider("value", listA...))

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		listB[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, ctx.Provider("value", listB...))

	b.ResetTimer()
	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}

var (
	testScheduler = &mockScheduler{}
)

func init() {
	setInternalScheduler(testScheduler)
}

func TestUseEffect_RunsAfterFlush(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	effectCalled := false
	comp := SimpleFC("TestComp", func() Node {
		UseEffect(func() {
			effectCalled = true
		}, []any{})
		return Box(BoxProps{})
	})

	Render(comp(), container)

	if effectCalled {
		t.Error("expected effect to not have run immediately after render")
	}

	testScheduler.flushMacrotasks()

	if !effectCalled {
		t.Error("expected effect to have run after flushing")
	}
}

func TestUseEffect_DepsNil_RunsEveryRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	runCount := 0
	comp := SimpleFC("TestComp", func() Node {
		_, set := UseState(0)
		setTrigger = set
		UseEffect(func() {
			runCount++
		}, nil)
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setTrigger(1) // Trigger re-render
	testScheduler.flushMacrotasks()
	if runCount != 2 {
		t.Errorf("expected 2 runs after first re-render, got %d", runCount)
	}

	setTrigger(2) // Trigger another re-render
	testScheduler.flushMacrotasks()
	if runCount != 3 {
		t.Errorf("expected 3 runs after second re-render, got %d", runCount)
	}
}

func TestUseEffect_DepsEmpty_RunsOnce(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	runCount := 0
	comp := SimpleFC("TestComp", func() Node {
		_, set := UseState(0)
		setTrigger = set
		UseEffect(func() {
			runCount++
		}, []any{})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setTrigger(1) // Trigger re-render
	testScheduler.flushMacrotasks()
	if runCount != 1 {
		t.Errorf("expected still 1 run after re-render, got %d", runCount)
	}
}

func TestUseEffect_DepsChanged_Reruns(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setVal func(int)
	runCount := 0
	comp := SimpleFC("TestComp", func() Node {
		val, set := UseState(0)
		setVal = set
		UseEffect(func() {
			runCount++
		}, []any{val()})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setVal(0) // No change
	testScheduler.flushMacrotasks()
	if runCount != 1 {
		t.Errorf("expected 1 run when deps do not change, got %d", runCount)
	}

	setVal(1) // Change
	testScheduler.flushMacrotasks()
	if runCount != 2 {
		t.Errorf("expected 2 runs after deps change, got %d", runCount)
	}
}

func TestUseEffectCleanup_CleansUpBeforeRerun(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setVal func(int)
	var order []string
	comp := SimpleFC("TestComp", func() Node {
		val, set := UseState(0)
		setVal = set
		valVal := val()
		UseEffectCleanup(func() func() {
			order = append(order, fmt.Sprintf("effect-%d", valVal))
			return func() {
				order = append(order, fmt.Sprintf("cleanup-%d", valVal))
			}
		}, []any{valVal})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	setVal(1) // Change
	testScheduler.flushMacrotasks()

	expected := []string{"effect-0", "cleanup-0", "effect-1"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("unexpected execution order: %v, expected %v", order, expected)
	}
}

func TestUseEffectCleanup_CleansUpOnUnmount(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	cleanupCalled := false
	comp := SimpleFC("TestComp", func() Node {
		UseEffectCleanup(func() func() {
			return func() {
				cleanupCalled = true
			}
		}, []any{})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	if cleanupCalled {
		t.Error("cleanup should not run before unmount")
	}

	Render(nil, container) // Unmount
	if !cleanupCalled {
		t.Error("cleanup should have run on unmount")
	}
}

func TestUseLayoutEffect_RunsSynchronouslyAfterReconcile(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	layoutEffectCalled := false
	comp := SimpleFC("TestComp", func() Node {
		_, set := UseState(0)
		setTrigger = set
		UseLayoutEffect(func() {
			layoutEffectCalled = true
		}, nil)
		return Box(BoxProps{})
	})

	Render(comp(), container)
	if !layoutEffectCalled {
		t.Error("expected layout effect to run synchronously during Render mount")
	}

	layoutEffectCalled = false
	setTrigger(1) // Re-render
	if !layoutEffectCalled {
		t.Error("expected layout effect to run synchronously after re-render")
	}
}

func TestUseLayoutEffect_CanTriggerReRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	renderCount := 0
	comp := SimpleFC("TestComp", func() Node {
		renderCount++
		val, setVal := UseState(0)

		UseLayoutEffect(func() {
			if val() == 0 {
				setVal(1)
			}
		}, []any{val()})

		return Box(BoxProps{})
	})

	Render(comp(), container)
	if renderCount != 2 {
		t.Errorf("expected 2 renders due to state update in layout effect, got %d", renderCount)
	}
}

func TestUseLayoutEffect_ReentrancyCap(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	renderCount := 0
	comp := SimpleFC("TestComp", func() Node {
		renderCount++
		_, setVal := UseState(0)

		UseLayoutEffect(func() {
			setVal(renderCount) // Infinite loop trigger
		}, nil)

		return Box(BoxProps{})
	})

	Render(comp(), container)
	// Unmount to clear global activeRoots state and prevent dirty-component
	// pollution from leaking into subsequent benchmarks.
	defer Render(nil, container)

	// Initial render + 10 re-renders = 11 total renders (cap at 10 iterations)
	if renderCount != 11 {
		t.Errorf("expected renderCount to be capped at 11, got %d", renderCount)
	}
}

func TestDestroy_RunsAllCleanups(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	cleanup1Called := false
	cleanup2Called := false

	comp := SimpleFC("TestComp", func() Node {
		UseEffectCleanup(func() func() {
			return func() { cleanup1Called = true }
		}, []any{})

		UseLayoutEffectCleanup(func() func() {
			return func() { cleanup2Called = true }
		}, []any{})

		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	Render(nil, container) // Destroy/unmount

	if !cleanup1Called {
		t.Error("expected UseEffectCleanup cleanup to run")
	}
	if !cleanup2Called {
		t.Error("expected UseLayoutEffectCleanup cleanup to run")
	}
}

func TestDestroy_RecursiveChildren(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	childCleanupCalled := false
	childComp := SimpleFC("ChildComp", func() Node {
		UseEffectCleanup(func() func() {
			return func() { childCleanupCalled = true }
		}, []any{})
		return Box(BoxProps{})
	})

	parentComp := SimpleFC("ParentComp", func() Node {
		return Box(BoxProps{}, childComp())
	})

	Render(parentComp(), container)
	testScheduler.flushMacrotasks()

	Render(nil, container) // Destroy/unmount

	if !childCleanupCalled {
		t.Error("expected nested child component cleanup to run on unmount")
	}
}

func TestFlushBeforeRender_Guarantee(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	effectRan := false
	renderedVal := -1

	comp := SimpleFC("TestComp", func() Node {
		val, set := UseState(0)
		setTrigger = set

		UseEffect(func() {
			effectRan = true
		}, []any{})

		renderedVal = val()
		if val() > 0 {
			if !effectRan {
				t.Error("expected effect from mount to run before the second render")
			}
		}

		return Box(BoxProps{})
	})

	Render(comp(), container)
	// Do NOT call testScheduler.flushMacrotasks().
	// Trigger state update
	setTrigger(1)

	if renderedVal != 1 {
		t.Errorf("expected to have rendered val 1, got %d", renderedVal)
	}
	if !effectRan {
		t.Error("expected effect to have run")
	}
}

func BenchmarkUseEffect(b *testing.B) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	comp := SimpleFC("BenchComp", func() Node {
		// Queue up 100 effects
		for range 100 {
			UseEffect(func() {}, nil)
		}
		return Box(BoxProps{})
	})

	for b.Loop() {
		Render(comp(), container)
		testScheduler.flushMacrotasks()
	}
}

func BenchmarkDestroy(b *testing.B) {
	doc := dom.NewDocument()

	var buildTree func(depth int) Node
	buildTree = func(depth int) Node {
		if depth <= 0 {
			return Box(BoxProps{})
		}
		c := SimpleFC("Child", func() Node {
			UseEffectCleanup(func() func() {
				return func() {}
			}, []any{})
			return Box(BoxProps{}, buildTree(depth-1))
		})
		return c()
	}

	for b.Loop() {
		container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
		Render(buildTree(5), container)
		testScheduler.flushMacrotasks()
		Render(nil, container) // Destroy
	}
}

func BenchmarkLayoutEffectReRender(b *testing.B) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	comp := SimpleFC("BenchComp", func() Node {
		val, setVal := UseState(0)
		UseLayoutEffect(func() {
			if val() == 0 {
				setVal(1)
			}
		}, []any{val()})
		return Box(BoxProps{})
	})

	for b.Loop() {
		Render(comp(), container)
		Render(nil, container) // cleanup
	}
}

func BenchmarkFlushBeforeRender(b *testing.B) {
	for b.Loop() {
		effectsMutex.Lock()
		pendingEffects = pendingEffects[:0]
		// Mock state to avoid large allocations in loop
		mockState := &effectHookState{
			pending:  true,
			simpleFn: func() {},
		}
		for range 100 {
			pendingEffects = append(pendingEffects, mockState)
		}
		effectsMutex.Unlock()
		testScheduler.flushMacrotasks()
	}
}

func BenchmarkDepsChange(b *testing.B) {
	dep1 := []any{1, "test", true}
	dep2 := []any{2, "test", true}

	for b.Loop() {
		_ = depsEqual(dep1, dep2)
	}
}

func BenchmarkNoDepsChange(b *testing.B) {
	dep1 := []any{1, "test", true}

	for b.Loop() {
		_ = depsEqual(dep1, dep1)
	}
}

func TestUseFocus(t *testing.T) {
	doc := dom.NewDocument()

	type State struct {
		isFocused bool
	}

	state := &State{}

	FocusApp := SimpleFC("FocusApp", func() Node {
		ref := UseRef[dom.Element](nil)

		isFocused := UseFocus(ref)
		state.isFocused = isFocused

		return Box(BoxProps{
			Ref: ref,
		}, Text("Focus me!"))
	})

	container := doc.CreateElement("container", nil)
	Render(FocusApp(), container)
	testScheduler.flushMacrotasks()

	Render(FocusApp(), container)
	testScheduler.flushMacrotasks()

	// TestUseFocus_InitiallyFalse
	if state.isFocused != false {
		t.Errorf("Expected isFocused to be false initially")
	}

	boxEl := container.FirstChild().(dom.Element)

	// TestUseFocus_TrueOnFocus
	focusEv := event.NewFocusEvent(event.EventFocus, nil)
	boxEl.DispatchToTarget(focusEv)

	// Since we are mocking render loop, the state update schedules a re-render.
	// We need to re-render to see the new state.
	Render(FocusApp(), container)

	if state.isFocused != true {
		t.Errorf("Expected isFocused to be true after focus event")
	}

	// TestUseFocus_FalseOnBlur
	blurEv := event.NewFocusEvent(event.EventBlur, nil)
	boxEl.DispatchToTarget(blurEv)

	Render(FocusApp(), container)

	if state.isFocused != false {
		t.Errorf("Expected isFocused to be false after blur event")
	}

	// TestUseFocus_NilRef
	// Let's create another component with a nil ref
	NilRefApp := SimpleFC("NilRefApp", func() Node {
		ref := CreateRef[dom.Element]() // Empty ref
		UseFocus(ref)                   // Should not panic
		return Box(BoxProps{}, Text("Nil ref"))
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UseFocus panicked on nil ref: %v", r)
		}
	}()
	Render(NilRefApp(), container)
}

func TestUseKeyboard(t *testing.T) {
	doc := dom.NewDocument()

	type State struct {
		lastPressed string
	}

	state := &State{}

	KeyboardApp := SimpleFC("KeyboardApp", func() Node {
		UseKeyboard(func(e event.KeyEvent) {
			state.lastPressed = e.Text
		}, nil)

		return Box(BoxProps{}, Text("Keyboard hook app"))
	})

	// TestUseKeyboard_HandlerCalled
	// We need to simulate the macro/micro loop to flush the UseEffect
	container := doc.CreateElement("container", nil)
	Render(KeyboardApp(), container)
	testScheduler.flushMacrotasks()

	// Dispatch key press to document
	keyEv := event.NewKeyEvent(event.EventKeyDown, key.Key{Text: "A", Code: 'A'})
	doc.DispatchToTarget(keyEv)

	if state.lastPressed != "A" {
		t.Errorf("Expected lastPressed to be 'A', got %q", state.lastPressed)
	}

	// TestUseKeyboard_Cleanup
	// Unmount the component to trigger cleanup
	Render(nil, container)
	testScheduler.flushMacrotasks()

	// Dispatch another key press
	keyEv2 := event.NewKeyEvent(event.EventKeyDown, key.Key{Text: "B", Code: 'B'})
	doc.DispatchToTarget(keyEv2)

	// Since the component is unmounted and listener removed, state should not update
	if state.lastPressed == "B" {
		t.Errorf("Expected handler to be removed on unmount, but it was still called")
	}
}

func TestUseState_UnmountedComponentNoop(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setState func(int)
	var getState func() int
	comp := SimpleFC("TestComp", func() Node {
		get, set := UseState(10)
		setState = set
		getState = get
		return Box(BoxProps{})
	})

	Render(comp(), container)

	if getState() != 10 {
		t.Errorf("expected 10, got %d", getState())
	}

	Render(nil, container) // Unmount/destroy component

	// Trigger state update after unmount
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UseState set function panicked on unmounted component: %v", r)
		}
	}()

	setState(42)

	// Getter should return last known value
	if getState() != 10 {
		t.Errorf("expected getter to return last known value 10, got %d", getState())
	}
}
