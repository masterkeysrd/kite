package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// --- complexity() tests -------------------------------------------------------

// TestComplexityTextNode verifies that textNode always returns 1.
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
	myComp := FC("richComp", func(props struct{}) Node {
		return Box(BoxProps{},
			Text("1"), Text("2"), Text("3"),
			Text("4"), Text("5"), Text("6"),
		)
	})

	node := myComp(struct{}{})
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

	myComp := FC("smallComp", func(props struct{}) Node {
		return Box(BoxProps{}, Text("hi")) // score=2
	})

	node := myComp(struct{}{})
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
	node1 := myComp(RichProps{Title: "hello"})
	realNode := node1.Instantiate(doc)
	if renderCount != 1 {
		t.Fatalf("expected renderCount=1 after Instantiate, got %d", renderCount)
	}

	comp1 := node1.(*ComponentNode[RichProps])
	if !comp1.shouldMemo {
		t.Fatalf("shouldMemo should be true (score=%d)", comp1.complexityScore)
	}

	oldRendered := comp1.rendered

	// Update with identical props → memoization should kick in, no RenderFn call.
	node2 := myComp(RichProps{Title: "hello"})
	node2.Update(realNode, node1)
	if renderCount != 1 {
		t.Errorf("expected renderCount=1 after memo hit, got %d (RenderFn was called unexpectedly)", renderCount)
	}

	comp2 := node2.(*ComponentNode[RichProps])
	if comp2.rendered != oldRendered {
		t.Errorf("memoized component should reuse the old rendered node")
	}

	// Update with changed props → RenderFn must be called.
	node3 := myComp(RichProps{Title: "world"})
	node3.Update(realNode, node2)
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

	node1 := myComp(P{V: "a"})
	realNode := node1.Instantiate(doc)

	comp := node1.(*ComponentNode[P])
	if comp.shouldMemo {
		t.Fatalf("shouldMemo should be false for small trees (score=%d)", comp.complexityScore)
	}

	node2 := myComp(P{V: "a"}) // identical props but small tree
	node2.Update(realNode, node1)
	if renderCount != 2 {
		t.Errorf("small tree should always re-render, expected renderCount=2, got %d", renderCount)
	}
}

// --- UseMemo hook tests -------------------------------------------------------

// TestUseMemoCallsFactoryOnFirstRender ensures factory is called on the initial render.
func TestUseMemoCallsFactoryOnFirstRender(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0

	myComp := FC("MemoComp", func(props struct{}) Node {
		val := UseMemo(func() int {
			callCount++
			return 42
		}, []any{})
		_ = val
		return Box(BoxProps{})
	})

	node := myComp(struct{}{})
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
		}, []any{props.Dep})
		lastVal = val
		return Box(BoxProps{})
	})

	// Initial render: dep=5
	node1 := myComp(P{Dep: 5})
	realNode := node1.Instantiate(doc)
	if callCount != 1 || lastVal != 50 {
		t.Fatalf("initial render: callCount=%d lastVal=%d, want 1 / 50", callCount, lastVal)
	}

	// Re-render with same dep → factory should NOT be called.
	node2 := myComp(P{Dep: 5})
	node2.Update(realNode, node1)
	if callCount != 1 {
		t.Errorf("same deps: factory should not be called again, got callCount=%d", callCount)
	}
	if lastVal != 50 {
		t.Errorf("expected cached value 50, got %d", lastVal)
	}

	// Re-render with changed dep → factory MUST be called.
	node3 := myComp(P{Dep: 7})
	node3.Update(realNode, node2)
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
		}, []any{props.A, props.B})
		return Box(BoxProps{})
	})

	node1 := myComp(P{A: 1, B: "x"})
	realNode := node1.Instantiate(doc)
	if callCount != 1 {
		t.Fatalf("initial: callCount=%d, want 1", callCount)
	}

	// Same deps → no re-eval.
	node2 := myComp(P{A: 1, B: "x"})
	node2.Update(realNode, node1)
	if callCount != 1 {
		t.Errorf("same deps: callCount=%d, want 1", callCount)
	}

	// Changing B only → re-eval.
	node3 := myComp(P{A: 1, B: "y"})
	node3.Update(realNode, node2)
	if callCount != 2 {
		t.Errorf("changed B: callCount=%d, want 2", callCount)
	}
}

// TestUseMemoEmptyDeps verifies that an empty dep slice never re-triggers the factory
// (equivalent to "run once" semantics).
func TestUseMemoEmptyDeps(t *testing.T) {
	doc := dom.NewDocument()
	callCount := 0

	myComp := FC("EmptyDeps", func(props struct{}) Node {
		_ = UseMemo(func() int {
			callCount++
			return 99
		}, []any{})
		return Box(BoxProps{})
	})

	node1 := myComp(struct{}{})
	realNode := node1.Instantiate(doc)
	if callCount != 1 {
		t.Fatalf("initial: callCount=%d, want 1", callCount)
	}

	node2 := myComp(struct{}{})
	node2.Update(realNode, node1)
	node3 := myComp(struct{}{})
	node3.Update(realNode, node2)
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
	UseMemo(func() int { return 0 }, nil)
}
