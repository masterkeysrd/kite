package kitex

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/terminal"
)

type TestProps struct {
	Label string
}

type TestChildrenProps struct {
	Title string
}

func TestFCAndFCCConstruction(t *testing.T) {
	// Test FC
	myComp := FC("myComponent", func(props TestProps) Node {
		return Box(BoxProps{
			ID: props.Label,
		})
	})

	node := myComp.Props(TestProps{Label: "hello"})
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
	myCompWithChildren := FCC("myChildrenComponent", func(props TestChildrenProps, children ...Node) Node {
		return Box(BoxProps{
			ID: props.Title,
		}, children...)
	})

	child1 := Text("child1")
	child2 := Text("child2")
	nodeWithChildren := myCompWithChildren.Props(TestChildrenProps{Title: "layout"}, child1, child2)

	compNodeWithChildren, ok := nodeWithChildren.(*ComponentNode[TestChildrenProps])
	if !ok {
		t.Fatalf("expected *ComponentNode[TestChildrenProps], got %T", nodeWithChildren)
	}

	if compNodeWithChildren.PropsVal.Title != "layout" {
		t.Errorf("expected Title layout, got %s", compNodeWithChildren.PropsVal.Title)
	}
	children := compNodeWithChildren.PassedChildren
	if len(children) != 2 {
		t.Errorf("expected 2 passed children, got %d", len(children))
	}
	if children[0] != child1 || children[1] != child2 {
		t.Errorf("passed children do not match original children")
	}
}

func TestUseStatePersistence(t *testing.T) {
	doc := dom.NewDocument()

	var getState1 func() int
	var setState1 func(int)
	var getState2 func(string) string
	var setState2 func(string)

	renderCount := 0

	myComp := FC("testComp", func(props NoProps) Node {
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
	node := myComp()
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
	node2 := myComp()
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
	if !compNode.dirty.Load() {
		t.Errorf("expected component to be marked dirty")
	}

	// In a real reconciler, it would re-render. Let's simulate that re-render:
	node3 := myComp()
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

	myComp := FC("testSync", func(props NoProps) Node {
		_, set := UseState(0)
		setState = set
		return Box(BoxProps{})
	})

	node := myComp()
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

	myComp := FC("testRefComp", func(props NoProps) Node {
		renderCount++
		myRef = UseRef(100)
		return Box(BoxProps{})
	})

	// 1. Initial instantiate
	node1 := myComp()
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
	node2 := myComp()
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

	myComp := FC("testReducerComp", func(props NoProps) Node {
		getState, _ = UseReducer(func(s int, a string) int {
			return s
		}, 42)
		return Box(BoxProps{})
	})

	node := myComp()
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

	myComp := FC("testReducerComp", func(props NoProps) Node {
		getState, dispatch = UseReducer(reducer, 10)
		return Box(BoxProps{})
	})

	node := myComp()
	realNodes := node.Instantiate(doc)

	dispatch("inc")

	// Simulate re-render
	node2 := myComp()
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

	myComp := FC("testReducerComp", func(props NoProps) Node {
		getState, dispatch = UseReducer(reducer, 10)
		return Box(BoxProps{})
	})

	node := myComp()
	realNodes := node.Instantiate(doc)

	dispatch("inc")
	dispatch("inc")
	dispatch("dec")
	dispatch("inc")

	// Simulate re-render
	node2 := myComp()
	node2.Update(realNodes, node)

	if getState() != 12 {
		t.Errorf("expected state to be 12 after multiple dispatches, got %d", getState())
	}
}

func TestUseReducer_DispatchTriggersReRender(t *testing.T) {
	doc := dom.NewDocument()
	var dispatch func(string)

	myComp := FC("testReducerComp", func(props NoProps) Node {
		_, dispatch = UseReducer(func(state int, action string) int {
			return state + 1
		}, 0)
		return Box(BoxProps{})
	})

	node := myComp()
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

	myComp := FC("testReducerComp", func(props NoProps) Node {
		getState, _ := UseReducer(func(state int, action string) int {
			return state
		}, 0)
		getStates = append(getStates, getState)
		return Box(BoxProps{})
	})

	node1 := myComp()
	realNodes := node1.Instantiate(doc)

	node2 := myComp()
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
		cbCached := UseCallback(cb, props.x)
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp.Props(struct {
		useSecond bool
		x         int
	}{useSecond: false, x: 1})
	realNodes := node1.Instantiate(doc)

	node2 := myComp.Props(struct {
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
		cbCached := UseCallback(cb, props.x)
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp.Props(struct {
		useSecond bool
		x         int
	}{useSecond: false, x: 1})
	realNodes := node1.Instantiate(doc)

	node2 := myComp.Props(struct {
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

func TestUseCallback_NoDeps_MemoizesAcrossLifetime(t *testing.T) {
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
		cbCached := UseCallback(cb)
		callbacks = append(callbacks, cbCached)
		return Box(BoxProps{})
	})

	node1 := myComp.Props(struct{ useSecond bool }{useSecond: false})
	realNodes := node1.Instantiate(doc)

	node2 := myComp.Props(struct{ useSecond bool }{useSecond: true})
	node2.Update(realNodes, node1)

	if len(callbacks) != 2 {
		t.Fatalf("expected 2 renders, got %d", len(callbacks))
	}

	ptr1 := reflect.ValueOf(callbacks[0]).Pointer()
	ptr2 := reflect.ValueOf(callbacks[1]).Pointer()
	if ptr1 != ptr2 {
		t.Errorf("expected same callback reference with no deps (memoized across lifetime), got different pointers")
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
	comp := FC("TestComp", func(_ NoProps) Node {
		UseEffect(func() {
			effectCalled = true
		})
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

func TestUseEffectEveryRender_RunsEveryRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	runCount := 0
	comp := FC("TestComp", func(_ NoProps) Node {
		_, set := UseState(0)
		setTrigger = set
		UseEffectEveryRender(func() {
			runCount++
		})
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

func TestUseEffectEveryRenderCleanup_RunsAndCleansEveryRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	var events []string
	comp := FC("TestComp", func(_ NoProps) Node {
		val, set := UseState(0)
		setTrigger = set
		cur := val()
		UseEffectEveryRenderCleanup(func() func() {
			events = append(events, fmt.Sprintf("run-%d", cur))
			return func() {
				events = append(events, fmt.Sprintf("clean-%d", cur))
			}
		})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()
	setTrigger(1)
	testScheduler.flushMacrotasks()
	Render(nil, container)

	want := []string{"run-0", "clean-0", "run-1", "clean-1"}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("got %v, want %v", events, want)
	}
}

func TestUseEffect_NoDeps_RunsOnce(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	runCount := 0
	comp := FC("TestComp", func(_ NoProps) Node {
		_, set := UseState(0)
		setTrigger = set
		UseEffect(func() {
			runCount++
		})
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
	comp := FC("TestComp", func(_ NoProps) Node {
		val, set := UseState(0)
		setVal = set
		UseEffect(func() {
			runCount++
		}, val())
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
	comp := FC("TestComp", func(_ NoProps) Node {
		val, set := UseState(0)
		setVal = set
		valVal := val()
		UseEffectCleanup(func() func() {
			order = append(order, fmt.Sprintf("effect-%d", valVal))
			return func() {
				order = append(order, fmt.Sprintf("cleanup-%d", valVal))
			}
		}, valVal)
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
	comp := FC("TestComp", func(_ NoProps) Node {
		UseEffectCleanup(func() func() {
			return func() {
				cleanupCalled = true
			}
		})
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
	comp := FC("TestComp", func(_ NoProps) Node {
		_, set := UseState(0)
		setTrigger = set
		UseLayoutEffectEveryRender(func() {
			layoutEffectCalled = true
		})
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

func TestUseLayoutEffect_NoDeps_RunsOnceAndEveryRenderCleanup(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setTrigger func(int)
	mountRuns := 0
	var everyEvents []string
	comp := FC("TestComp", func(_ NoProps) Node {
		val, set := UseState(0)
		setTrigger = set
		cur := val()
		UseLayoutEffect(func() {
			mountRuns++
		})
		UseLayoutEffectEveryRenderCleanup(func() func() {
			everyEvents = append(everyEvents, fmt.Sprintf("run-%d", cur))
			return func() {
				everyEvents = append(everyEvents, fmt.Sprintf("clean-%d", cur))
			}
		})
		return Box(BoxProps{})
	})

	Render(comp(), container)
	setTrigger(1)
	Render(nil, container)

	if mountRuns != 1 {
		t.Errorf("expected mount-only UseLayoutEffect to run once, got %d", mountRuns)
	}
	want := []string{"run-0", "clean-0", "run-1", "clean-1"}
	if !reflect.DeepEqual(everyEvents, want) {
		t.Errorf("got %v, want %v", everyEvents, want)
	}
}

func TestUseLayoutEffect_CanTriggerReRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	renderCount := 0
	comp := FC("TestComp", func(_ NoProps) Node {
		renderCount++
		val, setVal := UseState(0)

		UseLayoutEffect(func() {
			if val() == 0 {
				setVal(1)
			}
		}, val())

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
	comp := FC("TestComp", func(_ NoProps) Node {
		renderCount++
		_, setVal := UseState(0)

		UseLayoutEffectEveryRender(func() {
			setVal(renderCount) // Infinite loop trigger
		})

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

	comp := FC("TestComp", func(_ NoProps) Node {
		UseEffectCleanup(func() func() {
			return func() { cleanup1Called = true }
		})

		UseLayoutEffectCleanup(func() func() {
			return func() { cleanup2Called = true }
		})

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
	childComp := FC("ChildComp", func(_ NoProps) Node {
		UseEffectCleanup(func() func() {
			return func() { childCleanupCalled = true }
		})
		return Box(BoxProps{})
	})

	parentComp := FC("ParentComp", func(_ NoProps) Node {
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

	comp := FC("TestComp", func(_ NoProps) Node {
		val, set := UseState(0)
		setTrigger = set

		UseEffect(func() {
			effectRan = true
		})

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

	comp := FC("BenchComp", func(_ NoProps) Node {
		// Queue up 100 effects
		for range 100 {
			UseEffectEveryRender(func() {})
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
		c := FC("Child", func(_ NoProps) Node {
			UseEffectCleanup(func() func() {
				return func() {}
			})
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

	comp := FC("BenchComp", func(_ NoProps) Node {
		val, setVal := UseState(0)
		UseLayoutEffect(func() {
			if val() == 0 {
				setVal(1)
			}
		}, val())
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

	FocusApp := FC("FocusApp", func(_ NoProps) Node {
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
	NilRefApp := FC("NilRefApp", func(_ NoProps) Node {
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

	KeyboardApp := FC("KeyboardApp", func(_ NoProps) Node {
		UseKeyboard(func(e event.KeyEvent) {
			state.lastPressed = e.Text
		})

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
	comp := FC("TestComp", func(_ NoProps) Node {
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

func TestUseTimeout(t *testing.T) {
	doc := dom.NewDocument()
	container := doc.CreateElement("container", nil)

	var called atomic.Bool
	TimeoutApp := FC("TimeoutApp", func(_ NoProps) Node {
		UseTimeout(func() {
			called.Store(true)
		}, 10*time.Millisecond)

		return Box(BoxProps{}, Text("Timeout app"))
	})

	Render(TimeoutApp(), container)
	testScheduler.flushMacrotasks()

	// Wait for the timeout to elapse
	time.Sleep(30 * time.Millisecond)
	testScheduler.flushMacrotasks()

	if !called.Load() {
		t.Error("expected UseTimeout callback to be executed")
	}

	// Now test unmount cleanup
	var called2 atomic.Bool
	TimeoutApp2 := FC("TimeoutApp2", func(_ NoProps) Node {
		UseTimeout(func() {
			called2.Store(true)
		}, 20*time.Millisecond)

		return Box(BoxProps{}, Text("Timeout app 2"))
	})

	Render(TimeoutApp2(), container)
	testScheduler.flushMacrotasks()

	// Immediately unmount to trigger cleanup
	Render(nil, container)
	testScheduler.flushMacrotasks()

	// Wait for the timeout duration
	time.Sleep(40 * time.Millisecond)

	if called2.Load() {
		t.Error("expected UseTimeout callback NOT to be executed after unmount")
	}
}

func TestUseInterval(t *testing.T) {
	doc := dom.NewDocument()
	container := doc.CreateElement("container", nil)

	var ticks atomic.Int32
	IntervalApp := FC("IntervalApp", func(_ NoProps) Node {
		UseInterval(func() {
			ticks.Add(1)
		}, 10*time.Millisecond)

		return Box(BoxProps{}, Text("Interval app"))
	})

	Render(IntervalApp(), container)
	testScheduler.flushMacrotasks()

	// Wait for a couple of ticks
	time.Sleep(35 * time.Millisecond)
	testScheduler.flushMacrotasks()

	currentTicks := ticks.Load()
	if currentTicks < 2 {
		t.Errorf("expected at least 2 ticks, got %d", currentTicks)
	}

	// Unmount to stop interval
	Render(nil, container)
	testScheduler.flushMacrotasks()

	// Wait more time
	time.Sleep(20 * time.Millisecond)

	if ticks.Load() != currentTicks {
		t.Errorf("ticks continued after unmount: was %d, now %d", currentTicks, ticks.Load())
	}
}

type mockViewportView struct {
	dom.View // stub
	size     geom.Size
}

func (v *mockViewportView) ViewportSize() geom.Size { return v.size }

func TestUseViewportSize(t *testing.T) {
	doc := dom.NewDocument()
	view := &mockViewportView{size: geom.Size{Width: 80, Height: 24}}
	if setter, ok := doc.(interface{ SetDefaultView(dom.View) }); ok {
		setter.SetDefaultView(view)
	}

	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var size geom.Size
	comp := FC("ViewportComp", func(_ NoProps) Node {
		size = UseViewportSize()
		return Box(BoxProps{})
	})

	// 1. Initial render
	Render(comp(), container)
	testScheduler.flushMacrotasks() // Run UseEffect which calls setSize
	testScheduler.flushMacrotasks() // Run re-render with the new size

	if size.Width != 80 || size.Height != 24 {
		t.Errorf("expected viewport size 80x24 on initial render, got %dx%d", size.Width, size.Height)
	}

	// 2. Change viewport size and dispatch EventResize
	view.size = geom.Size{Width: 100, Height: 40}
	doc.DispatchTo(event.NewResizeEvent(100, 40))
	testScheduler.flushMacrotasks() // Run resize listener which calls setSize
	testScheduler.flushMacrotasks() // Run re-render with the new size

	if size.Width != 100 || size.Height != 40 {
		t.Errorf("expected viewport size to update to 100x40, got %dx%d", size.Width, size.Height)
	}

	// 3. Unmount component
	Render(nil, container)
	testScheduler.flushMacrotasks()
}

type mockTerminal struct {
	terminal.Terminal
	title  string
	resume *terminal.ProgramResume
}

func (m *mockTerminal) SetTitle(title string)                           { m.title = title }
func (m *mockTerminal) SetProgramResume(resume *terminal.ProgramResume) { m.resume = resume }
func (m *mockTerminal) Scheduler() terminal.Scheduler                   { return testScheduler }
func (m *mockTerminal) OnClose(fn func())                               {}
func (m *mockTerminal) IsClosed() bool                                  { return false }

func TestUseTerminal(t *testing.T) {
	doc := dom.NewDocument()
	mockTerm := &mockTerminal{}
	doc.SetTerminal(mockTerm)

	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var term terminal.Terminal
	comp := FC("TerminalComp", func(_ NoProps) Node {
		getTerm := UseTerminal()
		term = getTerm()
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	if term == nil {
		t.Fatalf("expected UseTerminal to return a terminal object")
	}

	term.SetTitle("Test Title")
	if mockTerm.title != "Test Title" {
		t.Errorf("expected title to be 'Test Title', got %q", mockTerm.title)
	}

	// Unmount
	Render(nil, container)
	testScheduler.flushMacrotasks()
}

type mockTitleBellTerminal struct {
	terminal.Terminal
	title           string
	bell            int
	progressState   terminal.ProgressBarState
	progressPercent int
	resume          *terminal.ProgramResume
}

func (m *mockTitleBellTerminal) SetTitle(title string) { m.title = title }
func (m *mockTitleBellTerminal) Bell()                 { m.bell++ }
func (m *mockTitleBellTerminal) SetProgressBar(state terminal.ProgressBarState, percentage int) {
	m.progressState = state
	m.progressPercent = percentage
}
func (m *mockTitleBellTerminal) SetProgramResume(resume *terminal.ProgramResume) { m.resume = resume }
func (m *mockTitleBellTerminal) Scheduler() terminal.Scheduler                   { return testScheduler }
func (m *mockTitleBellTerminal) OnClose(fn func())                               {}
func (m *mockTitleBellTerminal) IsClosed() bool                                  { return false }

func TestUseTitleAndUseBell(t *testing.T) {
	doc := dom.NewDocument()
	mockTerm := &mockTitleBellTerminal{}
	doc.SetTerminal(mockTerm)

	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var ringBell func()
	var ringBell2 func()
	var setTitle func(string)
	var setProgress func(terminal.ProgressBarState, int)
	var setProgress2 func(terminal.ProgressBarState, int)

	comp := FC("TitleBellComp", func(_ NoProps) Node {
		_, setTitleSetter := UseTitle("Initial Title")
		setTitle = setTitleSetter

		ringBell = UseBell()
		ringBell2 = UseBell()

		setProgress = UseProgressBar()
		setProgress2 = UseProgressBar()

		return Box(BoxProps{})
	})

	// 1. Initial render
	Render(comp(), container)
	testScheduler.flushMacrotasks() // runs UseEffect which sets the initial title

	if mockTerm.title != "Initial Title" {
		t.Errorf("expected title to be 'Initial Title', got %q", mockTerm.title)
	}

	// Verify that UseBell returns a memoized function
	if reflect.ValueOf(ringBell).Pointer() != reflect.ValueOf(ringBell2).Pointer() {
		t.Errorf("expected UseBell to return memoized function")
	}

	// Verify that UseProgressBar returns a memoized function
	if reflect.ValueOf(setProgress).Pointer() != reflect.ValueOf(setProgress2).Pointer() {
		t.Errorf("expected UseProgressBar to return memoized function")
	}

	// 2. Update title state
	setTitle("Updated Title")
	testScheduler.flushMacrotasks() // runs the re-render and UseEffect with the new title

	if mockTerm.title != "Updated Title" {
		t.Errorf("expected title to be updated to 'Updated Title', got %q", mockTerm.title)
	}

	// 3. Ring bell
	if mockTerm.bell != 0 {
		t.Errorf("expected bell to not have rung yet, got %d", mockTerm.bell)
	}
	ringBell()
	if mockTerm.bell != 1 {
		t.Errorf("expected bell to have rung once, got %d", mockTerm.bell)
	}

	// 4. Update progress bar
	if mockTerm.progressState != terminal.ProgressBarHide {
		t.Errorf("expected initial progress bar state to be Hide, got %v", mockTerm.progressState)
	}
	setProgress(terminal.ProgressBarNormal, 80)
	if mockTerm.progressState != terminal.ProgressBarNormal || mockTerm.progressPercent != 80 {
		t.Errorf("expected progress bar Normal at 80%%, got %v at %d%%", mockTerm.progressState, mockTerm.progressPercent)
	}

	// Unmount
	Render(nil, container)
	testScheduler.flushMacrotasks()
}

func TestUseWindowFocus(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var isFocused bool
	comp := FC("WindowFocusComp", func(_ NoProps) Node {
		isFocused = UseWindowFocus()
		return Box(BoxProps{})
	})

	// 1. Initial render
	Render(comp(), container)
	testScheduler.flushMacrotasks()

	// Initially, it should be true
	if !isFocused {
		t.Errorf("expected window focus to be initially true")
	}

	// 2. Dispatch blur event
	doc.DispatchTo(event.NewWindowFocusEvent(event.EventWindowBlur))
	testScheduler.flushMacrotasks() // runs setFocused(false)
	testScheduler.flushMacrotasks() // runs the re-render

	if isFocused {
		t.Errorf("expected window focus to update to false on EventWindowBlur")
	}

	// 3. Dispatch focus event
	doc.DispatchTo(event.NewWindowFocusEvent(event.EventWindowFocus))
	testScheduler.flushMacrotasks() // runs setFocused(true)
	testScheduler.flushMacrotasks() // runs the re-render

	if !isFocused {
		t.Errorf("expected window focus to update to true on EventWindowFocus")
	}

	// Unmount
	Render(nil, container)
	testScheduler.flushMacrotasks()
}

func TestUseAsyncEffect(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	type runRecord struct {
		dep       int
		cancelled chan struct{}
		started   chan struct{}
	}

	var mu sync.Mutex
	var runs []*runRecord

	comp := FC("AsyncComp", func(props struct{ Dep int }) Node {
		dep := props.Dep
		UseAsyncEffect(func(ctx context.Context) {
			rec := &runRecord{
				dep:       dep,
				cancelled: make(chan struct{}),
				started:   make(chan struct{}),
			}
			mu.Lock()
			runs = append(runs, rec)
			mu.Unlock()
			close(rec.started)

			<-ctx.Done()
			close(rec.cancelled)
		}, dep)
		return Box(BoxProps{})
	})

	// Mount with Dep=1
	Render(comp.Props(struct{ Dep int }{Dep: 1}), container)
	testScheduler.flushMacrotasks()

	var run1 *runRecord
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		if len(runs) >= 1 {
			run1 = runs[0]
		}
		mu.Unlock()
		if run1 != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if run1 == nil {
		t.Fatal("expected initial async effect to start")
	}
	<-run1.started

	select {
	case <-run1.cancelled:
		t.Fatal("did not expect initial async effect ctx to be cancelled while mounted")
	default:
	}

	// Re-render with same Dep=1: must not cancel or re-run
	Render(comp.Props(struct{ Dep int }{Dep: 1}), container)
	testScheduler.flushMacrotasks()

	mu.Lock()
	countSameDep := len(runs)
	mu.Unlock()
	if countSameDep != 1 {
		t.Fatalf("expected 1 run when dep is unchanged, got %d", countSameDep)
	}

	// Re-render with Dep=2: cancels run1 and starts run2
	Render(comp.Props(struct{ Dep int }{Dep: 2}), container)
	testScheduler.flushMacrotasks()

	select {
	case <-run1.cancelled:
	case <-time.After(time.Second):
		t.Fatal("expected run1 ctx to be cancelled when dep changed")
	}

	var run2 *runRecord
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		if len(runs) >= 2 {
			run2 = runs[1]
		}
		mu.Unlock()
		if run2 != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if run2 == nil {
		t.Fatal("expected second async effect to start on dep change")
	}
	<-run2.started

	// Unmount: cancels run2
	Render(nil, container)
	testScheduler.flushMacrotasks()

	select {
	case <-run2.cancelled:
	case <-time.After(time.Second):
		t.Fatal("expected run2 ctx to be cancelled on unmount")
	}
}

func TestUseAsyncEffect_NilEffect(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	comp := FC("NilAsyncComp", func(_ NoProps) Node {
		UseAsyncEffect(nil)
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()
	Render(nil, container)
	testScheduler.flushMacrotasks()
}

func TestUseProgramResume(t *testing.T) {
	doc := dom.NewDocument()
	mockTerm := &mockTerminal{}
	doc.SetTerminal(mockTerm)

	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	var setResume1 func(*terminal.ProgramResume)
	var setResume2 func(*terminal.ProgramResume)

	comp := FC("ResumeComp", func(_ NoProps) Node {
		setResume1 = UseProgramResume()
		setResume2 = UseProgramResume()
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	if reflect.ValueOf(setResume1).Pointer() != reflect.ValueOf(setResume2).Pointer() {
		t.Fatal("expected UseProgramResume to return a memoized callback")
	}

	want := &terminal.ProgramResume{Args: []string{"kite-app", "--resume"}, Dir: "/tmp"}
	setResume1(want)
	if mockTerm.resume == nil || len(mockTerm.resume.Args) != 2 || mockTerm.resume.Dir != "/tmp" {
		t.Fatalf("expected mockTerm.resume to be set, got %+v", mockTerm.resume)
	}

	setResume1(nil)
	if mockTerm.resume != nil {
		t.Fatalf("expected mockTerm.resume to be cleared, got %+v", mockTerm.resume)
	}
}

func TestStateSetter_DirectAndFunctionalUpdate(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	var (
		getCount func() int
		setCount StateSetter[int]
	)

	comp := FC("DirectUpdateComp", func(_ NoProps) Node {
		g, s := UseState(10)
		getCount = g
		setCount = s
		return Textf("count:%d", g())
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	if got := getCount(); got != 10 {
		t.Fatalf("initial count = %d, want 10", got)
	}

	// Direct call set(val)
	setCount(25)
	if got := getCount(); got != 25 {
		t.Fatalf("after setCount(25), count = %d, want 25", got)
	}

	// Functional update set.Update(fn)
	setCount.Update(func(prev int) int { return prev * 2 })
	if got := getCount(); got != 50 {
		t.Fatalf("after Update(*2), count = %d, want 50", got)
	}
}

func TestStateSetter_ConcurrentAtomicUpdates(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	var (
		captureOnce sync.Once
		getCount    func() int
		setCount    StateSetter[int]
	)

	comp := FC("CounterComp", func(_ NoProps) Node {
		g, s := UseState(0)
		captureOnce.Do(func() {
			getCount = g
			setCount = s
		})
		return Textf("%d", g())
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	const workers = 100
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			setCount.Update(func(n int) int { return n + 1 })
		}()
	}
	wg.Wait()
	testScheduler.flushMacrotasks()

	if got := getCount(); got != workers {
		t.Fatalf("StateSetter.Update concurrent count = %d, want %d", got, workers)
	}
}

func TestStateSetter_NilSafety(t *testing.T) {
	var nilSetter StateSetter[int]
	// Calling Update on nil StateSetter must be a safe no-op.
	nilSetter.Update(func(n int) int { return n + 1 })

	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	var (
		getVal func() int
		setVal StateSetter[int]
	)
	comp := FC("NilUpdateComp", func(_ NoProps) Node {
		g, s := UseState(7)
		getVal = g
		setVal = s
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	// Calling set.Update(nil) must be a safe no-op.
	setVal.Update(nil)
	if got := getVal(); got != 7 {
		t.Fatalf("after Update(nil), value = %d, want 7", got)
	}
}

func TestStateSetter_AssignabilityAndStableIdentity(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer Render(nil, container)

	var (
		setters  []StateSetter[string]
		handlers []func(string)
	)

	comp := FC("AssignComp", func(props struct{ Step int }) Node {
		query, setQuery := UseState("")
		setters = append(setters, setQuery)

		// Verify direct assignability to func(string) without casts.
		inputNode := Input(InputProps{
			Value:         query(),
			OnValueChange: setQuery,
		})
		_ = inputNode
		var fn func(string) = setQuery
		handlers = append(handlers, fn)

		return Text(query())
	})

	Render(comp.Props(struct{ Step int }{Step: 1}), container)
	testScheduler.flushMacrotasks()
	Render(comp.Props(struct{ Step int }{Step: 2}), container)
	testScheduler.flushMacrotasks()

	if len(setters) != 2 || len(handlers) != 2 {
		t.Fatalf("expected 2 renders, got setters=%d handlers=%d", len(setters), len(handlers))
	}
	if !funcEquals(setters[0], setters[1]) {
		t.Errorf("expected StateSetter closure identity to remain stable across re-renders")
	}
	if !funcEquals(handlers[0], handlers[1]) {
		t.Errorf("expected assigned func(T) handler identity to remain stable across re-renders")
	}
}

func TestStateSetter_UnmountCleanup(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var (
		getVal func() int
		setVal StateSetter[int]
	)

	comp := FC("CleanupComp", func(_ NoProps) Node {
		g, s := UseState(5)
		getVal = g
		setVal = s
		return Textf("%d", g())
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	key := *(*uintptr)(unsafe.Pointer(&setVal))
	if _, ok := setterRegistry.Load(key); !ok {
		t.Fatalf("expected setterRegistry to contain mounted setter key %#x", key)
	}

	// Unmount triggers ComponentNode.Destroy(), which must clean up setterRegistry.
	Render(nil, container)
	testScheduler.flushMacrotasks()

	if _, ok := setterRegistry.Load(key); ok {
		t.Fatalf("expected setterRegistry to remove key %#x after ComponentNode.Destroy()", key)
	}

	// Subsequent calls after unmount must be safe no-ops.
	setVal(99)
	setVal.Update(func(n int) int { return n + 1 })
	if got := getVal(); got != 5 {
		t.Fatalf("expected unmounted state to remain 5, got %d", got)
	}
}

func BenchmarkUseState_Set(b *testing.B) {
	doc := dom.NewDocument()
	doc.SetTerminal(&mockTerminal{})
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer func() {
		Render(nil, container)
		testScheduler.flushMacrotasks()
	}()

	var setVal StateSetter[int]
	comp := FC("BenchSetComp", func(_ NoProps) Node {
		_, s := UseState(0)
		setVal = s
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setVal(i)
	}
	b.StopTimer()
}

func BenchmarkUseState_Update(b *testing.B) {
	doc := dom.NewDocument()
	doc.SetTerminal(&mockTerminal{})
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)
	defer func() {
		Render(nil, container)
		testScheduler.flushMacrotasks()
	}()

	var setVal StateSetter[int]
	comp := FC("BenchUpdateComp", func(_ NoProps) Node {
		_, s := UseState(0)
		setVal = s
		return Box(BoxProps{})
	})

	Render(comp(), container)
	testScheduler.flushMacrotasks()

	inc := func(n int) int { return n + 1 }

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setVal.Update(inc)
	}
	b.StopTimer()
}
