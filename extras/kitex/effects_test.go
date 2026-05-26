package kitex

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

var (
	scheduledMacros []func()
)

func init() {
	SetPostMacroFn(func(fn func()) {
		scheduledMacros = append(scheduledMacros, fn)
	})
}

func TestUseEffect_RunsAfterFlush(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	effectCalled := false
	comp := SimpleFC("TestComp", func() Node {
		UseEffect(func() {
			effectCalled = true
		}, []any{})
		return Box(BoxProps{})
	})

	scheduledMacros = nil
	Render(comp(), container)

	if effectCalled {
		t.Error("expected effect to not have run immediately after render")
	}

	flushPendingEffects()

	if !effectCalled {
		t.Error("expected effect to have run after flushing")
	}
}

func TestUseEffect_DepsNil_RunsEveryRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setTrigger(1) // Trigger re-render
	flushPendingEffects()
	if runCount != 2 {
		t.Errorf("expected 2 runs after first re-render, got %d", runCount)
	}

	setTrigger(2) // Trigger another re-render
	flushPendingEffects()
	if runCount != 3 {
		t.Errorf("expected 3 runs after second re-render, got %d", runCount)
	}
}

func TestUseEffect_DepsEmpty_RunsOnce(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setTrigger(1) // Trigger re-render
	flushPendingEffects()
	if runCount != 1 {
		t.Errorf("expected still 1 run after re-render, got %d", runCount)
	}
}

func TestUseEffect_DepsChanged_Reruns(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()
	if runCount != 1 {
		t.Errorf("expected 1 run on mount, got %d", runCount)
	}

	setVal(0) // No change
	flushPendingEffects()
	if runCount != 1 {
		t.Errorf("expected 1 run when deps do not change, got %d", runCount)
	}

	setVal(1) // Change
	flushPendingEffects()
	if runCount != 2 {
		t.Errorf("expected 2 runs after deps change, got %d", runCount)
	}
}

func TestUseEffectCleanup_CleansUpBeforeRerun(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()

	setVal(1) // Change
	flushPendingEffects()

	expected := []string{"effect-0", "cleanup-0", "effect-1"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("unexpected execution order: %v, expected %v", order, expected)
	}
}

func TestUseEffectCleanup_CleansUpOnUnmount(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()

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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	// Initial render + 10 re-renders = 11 total renders (cap at 10 iterations)
	if renderCount != 11 {
		t.Errorf("expected renderCount to be capped at 11, got %d", renderCount)
	}
}

func TestDestroy_RunsAllCleanups(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()

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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	flushPendingEffects()

	Render(nil, container) // Destroy/unmount

	if !childCleanupCalled {
		t.Error("expected nested child component cleanup to run on unmount")
	}
}

func TestFlushBeforeRender_Guarantee(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
	// Do NOT call flushPendingEffects().
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
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	comp := SimpleFC("BenchComp", func() Node {
		// Queue up 100 effects
		for range 100 {
			UseEffect(func() {}, nil)
		}
		return Box(BoxProps{})
	})

	for b.Loop() {
		Render(comp(), container)
		flushPendingEffects()
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
		container := Div(BoxProps{}).Instantiate(doc).(dom.Element)
		Render(buildTree(5), container)
		flushPendingEffects()
		Render(nil, container) // Destroy
	}
}

func BenchmarkLayoutEffectReRender(b *testing.B) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

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
		flushPendingEffects()
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
