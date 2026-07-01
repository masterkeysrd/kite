package kitex

import (
	"bytes"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/log"
	"github.com/masterkeysrd/kite/style"
	"image/color"
	"slices"
)

func TestReconcilerMountAndUnmount(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// 1. Initial Mount
	rootVNode := Box(BoxProps{ID: "app"},
		Span(SpanProps{ID: "child1"}, Text("Hello")),
	)
	Render(rootVNode, container)

	if container.FirstChild() == nil {
		t.Fatalf("expected container to have a child after mount")
	}

	appReal := container.FirstChild().(dom.Element)
	if appReal.ID() != "app" {
		t.Errorf("expected mounted element ID to be 'app', got %s", appReal.ID())
	}
	if appReal.FirstChild() == nil {
		t.Fatalf("expected child1 to be mounted")
	}
	child1Real := appReal.FirstChild().(dom.Element)
	if child1Real.ID() != "child1" {
		t.Errorf("expected child1 ID to be 'child1', got %s", child1Real.ID())
	}
	if child1Real.FirstChild().TextContent() != "Hello" {
		t.Errorf("expected text content 'Hello', got %s", child1Real.FirstChild().TextContent())
	}

	// 2. Unmount (Render nil)
	Render(nil, container)
	if container.FirstChild() != nil {
		t.Errorf("expected container to be empty after unmount")
	}
}

func TestReconcilerTagMismatchReplacement(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	// Mount a Box
	Render(Box(BoxProps{ID: "node1"}), container)
	node1 := container.FirstChild().(dom.Element)
	if node1.TagName() != "box" {
		t.Errorf("expected box, got %s", node1.TagName())
	}

	// Reconcile into a Span
	Render(Span(SpanProps{ID: "node2"}), container)
	node2 := container.FirstChild().(dom.Element)
	if node2.TagName() != "span" {
		t.Errorf("expected span, got %s", node2.TagName())
	}
	if node2.ID() != "node2" {
		t.Errorf("expected replaced node ID to be 'node2'")
	}
}

func TestReconcilerStateUpdateDirtyReRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setCounter func(int)

	CounterComponent := FC("CounterComponent", func(props struct{}) Node {
		getVal, setVal := UseState(0)
		setCounter = setVal
		return Span(SpanProps{ID: "count-span"},
			Text(fmt.Sprintf("Count: %d", getVal())),
		)
	})

	// Initial render
	Render(CounterComponent(struct{}{}), container)

	compNode := container.FirstChild() // The ComponentNode's real DOM node (which is the Span)
	spanReal := compNode.(dom.Element)
	if spanReal.FirstChild().TextContent() != "Count: 0" {
		t.Errorf("expected 'Count: 0', got %s", spanReal.FirstChild().TextContent())
	}

	// Trigger state update
	setCounter(5)

	// Since state update is reactive (it schedules reconciliation in OnComponentDirty),
	// it should have updated the DOM automatically!
	if spanReal.FirstChild().TextContent() != "Count: 5" {
		t.Errorf("expected reactive update to 'Count: 5', got %s", spanReal.FirstChild().TextContent())
	}
}

func TestReconcilerKeyedListReordering(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	// Render list: A, B, C with keys
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
	), container)

	appReal := container.FirstChild().(dom.Element)
	var firstChildList []dom.Node
	for child := range appReal.ChildNodes() {
		firstChildList = append(firstChildList, child)
	}

	if len(firstChildList) != 3 {
		t.Fatalf("expected 3 children, got %d", len(firstChildList))
	}
	realA := firstChildList[0].(dom.Element)
	realB := firstChildList[1].(dom.Element)
	realC := firstChildList[2].(dom.Element)

	if realA.ID() != "id-a" || realB.ID() != "id-b" || realC.ID() != "id-c" {
		t.Errorf("initial list IDs mismatch")
	}

	// Reorder list: C, A, B
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
	), container)

	var secondChildList []dom.Node
	for child := range appReal.ChildNodes() {
		secondChildList = append(secondChildList, child)
	}

	if len(secondChildList) != 3 {
		t.Fatalf("expected 3 children, got %d", len(secondChildList))
	}

	// Verify order
	if secondChildList[0].(dom.Element).ID() != "id-c" ||
		secondChildList[1].(dom.Element).ID() != "id-a" ||
		secondChildList[2].(dom.Element).ID() != "id-b" {
		t.Errorf("list was not reordered correctly")
	}

	// CRITICAL: Verify that the DOM elements themselves were moved, not recreated!
	if secondChildList[0] != realC {
		t.Errorf("element C was recreated instead of moved")
	}
	if secondChildList[1] != realA {
		t.Errorf("element A was recreated instead of moved")
	}
	if secondChildList[2] != realB {
		t.Errorf("element B was recreated instead of moved")
	}
}

// TestReconcilerKeyedListReverse is a regression test for the stale-snapshot
// bug in the double-ended reconciler: Cases 3 & 4 called InsertBefore but
// did not update oldRealChildren, causing subsequent index lookups to point
// at the wrong DOM nodes. Full reversal [A,B,C] → [C,B,A] exercises both
// cases in the same pass and is the minimal repro.
func TestReconcilerKeyedListReverse(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	Render(Box(BoxProps{},
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
	), container)

	appReal := container.FirstChild().(dom.Element)
	var before []dom.Node
	for child := range appReal.ChildNodes() {
		before = append(before, child)
	}
	realA, realB, realC := before[0], before[1], before[2]

	// Reverse: [C, B, A]
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
	), container)

	var after []dom.Node
	for child := range appReal.ChildNodes() {
		after = append(after, child)
	}

	if len(after) != 3 {
		t.Fatalf("expected 3 children after reverse, got %d", len(after))
	}

	wantOrder := []string{"id-c", "id-b", "id-a"}
	for i, n := range after {
		if got := n.(dom.Element).ID(); got != wantOrder[i] {
			t.Errorf("position %d: want %s, got %s", i, wantOrder[i], got)
		}
	}

	// Verify identity: nodes must be moved, not recreated.
	if after[0] != realC {
		t.Error("element C was recreated instead of moved")
	}
	if after[1] != realB {
		t.Error("element B was recreated instead of moved")
	}
	if after[2] != realA {
		t.Error("element A was recreated instead of moved")
	}
}

func TestReconcilerComponentListReverse(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	type ItemProps struct {
		Key string
		ID  string
	}
	ItemComp := FC("ItemComp", func(props ItemProps) Node {
		return Span(SpanProps{ID: "id-" + props.ID}, Text(props.ID))
	})

	// Render [A, B, C]
	Render(Box(BoxProps{},
		ItemComp(ItemProps{Key: "A", ID: "a"}),
		ItemComp(ItemProps{Key: "B", ID: "b"}),
		ItemComp(ItemProps{Key: "C", ID: "c"}),
	), container)

	appReal := container.FirstChild().(dom.Element)
	var before []dom.Node
	for child := range appReal.ChildNodes() {
		before = append(before, child)
	}
	realA, realB, realC := before[0], before[1], before[2]

	// Reverse: [C, B, A]
	Render(Box(BoxProps{},
		ItemComp(ItemProps{Key: "C", ID: "c"}),
		ItemComp(ItemProps{Key: "B", ID: "b"}),
		ItemComp(ItemProps{Key: "A", ID: "a"}),
	), container)

	var after []dom.Node
	for child := range appReal.ChildNodes() {
		after = append(after, child)
	}

	if len(after) != 3 {
		t.Fatalf("expected 3 children after reverse, got %d", len(after))
	}

	wantOrder := []string{"id-c", "id-b", "id-a"}
	for i, n := range after {
		if got := n.(dom.Element).ID(); got != wantOrder[i] {
			t.Errorf("position %d: want %s, got %s", i, wantOrder[i], got)
		}
	}

	// Verify identity: nodes must be moved, not recreated.
	if after[0] != realC {
		t.Error("element C was recreated instead of moved")
	}
	if after[1] != realB {
		t.Error("element B was recreated instead of moved")
	}
	if after[2] != realA {
		t.Error("element A was recreated instead of moved")
	}
}

func TestReconcilerInsertionsAndDeletions(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	// Render: A, B
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
	), container)

	appReal := container.FirstChild().(dom.Element)

	// Render: A, C, B (Insertion in middle)
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "A", ID: "id-a"}, Text("A")),
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
	), container)

	var list1 []dom.Node
	for child := range appReal.ChildNodes() {
		list1 = append(list1, child)
	}
	if len(list1) != 3 {
		t.Fatalf("expected 3 children, got %d", len(list1))
	}
	if list1[1].(dom.Element).ID() != "id-c" {
		t.Errorf("expected child at index 1 to be C, got %s", list1[1].(dom.Element).ID())
	}

	// Render: C, B (Deletion of A)
	Render(Box(BoxProps{},
		Span(SpanProps{Key: "C", ID: "id-c"}, Text("C")),
		Span(SpanProps{Key: "B", ID: "id-b"}, Text("B")),
	), container)

	var list2 []dom.Node
	for child := range appReal.ChildNodes() {
		list2 = append(list2, child)
	}
	if len(list2) != 2 {
		t.Fatalf("expected 2 children, got %d", len(list2))
	}
	if list2[0].(dom.Element).ID() != "id-c" || list2[1].(dom.Element).ID() != "id-b" {
		t.Errorf("unexpected elements after deletion")
	}
}

func TestReconcilerStateUpdateListReverse(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	type ItemData struct {
		Key string
		ID  string
	}
	type ItemProps struct {
		Key string
		ID  string
	}
	ItemComp := FC("ItemComp", func(props ItemProps) Node {
		return Span(SpanProps{ID: "id-" + props.ID}, Text(props.ID))
	})

	var setItems func([]ItemData)

	AppComp := FC("AppComp", func(props struct{}) Node {
		getItems, setIt := UseState([]ItemData{
			{Key: "1", ID: "a"},
			{Key: "2", ID: "b"},
			{Key: "3", ID: "c"},
		})
		setItems = setIt
		renderItem := func(item ItemData, _ int) Node {
			return ItemComp(ItemProps(item))
		}
		return Box(BoxProps{}, Map(getItems(), renderItem))
	})

	Render(AppComp(struct{}{}), container)

	appReal := container.FirstChild().(dom.Element)
	var before []dom.Node
	for child := range appReal.ChildNodes() {
		before = append(before, child)
	}
	realA, realB, realC := before[0], before[1], before[2]

	// Reverse
	setItems([]ItemData{
		{Key: "3", ID: "c"},
		{Key: "2", ID: "b"},
		{Key: "1", ID: "a"},
	})

	var after []dom.Node
	for child := range appReal.ChildNodes() {
		after = append(after, child)
	}

	if len(after) != 3 {
		t.Fatalf("expected 3 children after reverse, got %d", len(after))
	}

	wantOrder := []string{"id-c", "id-b", "id-a"}
	for i, n := range after {
		if got := n.(dom.Element).ID(); got != wantOrder[i] {
			t.Errorf("position %d: want %s, got %s", i, wantOrder[i], got)
		}
	}

	// Verify identity: nodes must be moved, not recreated.
	if after[0] != realC {
		t.Error("element C was recreated instead of moved")
	}
	if after[1] != realB {
		t.Error("element B was recreated instead of moved")
	}
	if after[2] != realA {
		t.Error("element A was recreated instead of moved")
	}
}

func TestReconcilerConditionalNilChildren(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setCondition func(bool)

	Comp := FC("ConditionalComp", func(props struct{}) Node {
		cond, setCond := UseState(true)
		setCondition = setCond

		return Box(BoxProps{ID: "parent"},
			Span(SpanProps{ID: "always-here"}),
			If(cond(), func() Node { return Span(SpanProps{ID: "conditional-span"}) }),
			Span(SpanProps{ID: "also-always-here"}),
		)
	})

	// 1. Initial Render (conditional is true)
	Render(Comp(struct{}{}), container)

	parent := container.FirstChild().(dom.Element)
	if parent.FirstChild() == nil {
		t.Fatalf("expected parent to have children")
	}

	// Should have always-here, conditional-span, and also-always-here
	childCount := 0
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		childCount++
	}
	if childCount != 3 {
		t.Errorf("expected 3 children initially, got %d", childCount)
	}

	// 2. Toggle condition to false (conditional-span unmounts, leaves nil child in new children)
	setCondition(false)

	childCount = 0
	var childIDs []string
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		childCount++
		childIDs = append(childIDs, child.(dom.Element).ID())
	}
	if childCount != 2 {
		t.Errorf("expected 2 children after condition false, got %d", childCount)
	}
	expectedIDs := []string{"always-here", "also-always-here"}
	if !reflect.DeepEqual(childIDs, expectedIDs) {
		t.Errorf("expected remaining children %v, got %v", expectedIDs, childIDs)
	}

	// 3. Toggle back to true (conditional-span remounts)
	setCondition(true)

	childCount = 0
	childIDs = nil
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		childCount++
		childIDs = append(childIDs, child.(dom.Element).ID())
	}
	if childCount != 3 {
		t.Errorf("expected 3 children after condition true, got %d", childCount)
	}
	expectedIDs2 := []string{"always-here", "conditional-span", "also-always-here"}
	if !reflect.DeepEqual(childIDs, expectedIDs2) {
		t.Errorf("expected children %v, got %v", expectedIDs2, childIDs)
	}
}

// BenchmarkReconcilerMount measures the cost of mounting a medium-sized VDOM tree.
func BenchmarkReconcilerMount(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()

	for b.Loop() {
		container := Box(BoxProps{}).Instantiate(doc)[0].(dom.Element)
		root := Box(BoxProps{ID: "app"},
			Span(SpanProps{ID: "s1"}, Text("text 1")),
			Span(SpanProps{ID: "s2"}, Text("text 2")),
			Span(SpanProps{ID: "s3"}, Text("text 3")),
			Span(SpanProps{ID: "s4"}, Text("text 4")),
			Span(SpanProps{ID: "s5"}, Text("text 5")),
		)
		Render(root, container)
		Render(nil, container)
	}
}

// BenchmarkReconcilerNonKeyedUpdate measures list update performance when NO keys are used.
func BenchmarkReconcilerNonKeyedUpdate(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	// Pre-create some lists
	listA := make([]Node, 100)
	for i := range 100 {
		listA[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, listA...)

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		listB[i] = Span(SpanProps{ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, listB...)

	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}

// BenchmarkReconcilerKeyedUpdate measures list update and reordering performance when keys ARE used.
func BenchmarkReconcilerKeyedUpdate(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	listA := make([]Node, 100)
	for i := range 100 {
		key := fmt.Sprintf("key-%d", i)
		listA[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, listA...)

	listB := make([]Node, 100)
	for i := range 100 {
		idx := 99 - i
		key := fmt.Sprintf("key-%d", idx)
		listB[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, listB...)

	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}

// BenchmarkReconcilerKeyedShuffled measures list update performance when keys are shuffled, forcing Case 5 lookup.
func BenchmarkReconcilerKeyedShuffled(b *testing.B) {
	EnableDevMode = false
	doc := dom.NewDocument()
	container := Box(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	listA := make([]Node, 100)
	for i := range 100 {
		key := fmt.Sprintf("key-%d", i)
		listA[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", i)}, Text(fmt.Sprintf("Item %d", i)))
	}
	rootA := Box(BoxProps{}, listA...)

	listB := make([]Node, 100)
	for i := range 100 {
		idx := (i * 17) % 100
		key := fmt.Sprintf("key-%d", idx)
		listB[i] = Span(SpanProps{Key: key, ID: fmt.Sprintf("id-%d", idx)}, Text(fmt.Sprintf("Item %d-updated", idx)))
	}
	rootB := Box(BoxProps{}, listB...)

	b.ResetTimer()
	for b.Loop() {
		Render(rootA, container)
		Render(rootB, container)
	}
}

func TestFragmentComponentReconciliation(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	type MyFragmentCompProps struct {
		Items []string
	}

	MyFragmentComp := FC("MyFragmentComp", func(props MyFragmentCompProps) Node {
		var nodes []Node
		for _, item := range props.Items {
			nodes = append(nodes, Span(SpanProps{ID: item}, Text(item)))
		}
		return Fragment(nodes...)
	})

	// 1. Mount: A, B
	Render(MyFragmentComp(MyFragmentCompProps{Items: []string{"A", "B"}}), container)

	var childIDs []string
	for child := range container.ChildNodes() {
		if el, ok := child.(dom.Element); ok {
			childIDs = append(childIDs, el.ID())
		}
	}
	expectedIDs1 := []string{"A", "B"}
	if !reflect.DeepEqual(childIDs, expectedIDs1) {
		t.Fatalf("expected mounted children %v, got %v", expectedIDs1, childIDs)
	}

	// 2. Update: B, C, D (reordering, adding, removing)
	Render(MyFragmentComp(MyFragmentCompProps{Items: []string{"B", "C", "D"}}), container)

	childIDs = nil
	for child := range container.ChildNodes() {
		if el, ok := child.(dom.Element); ok {
			childIDs = append(childIDs, el.ID())
		}
	}
	expectedIDs2 := []string{"B", "C", "D"}
	if !reflect.DeepEqual(childIDs, expectedIDs2) {
		t.Fatalf("expected updated children %v, got %v", expectedIDs2, childIDs)
	}

	// 3. Unmount
	Render(nil, container)
	if container.FirstChild() != nil {
		t.Errorf("expected container to be empty after unmount, but has child %v", container.FirstChild())
	}
}

func TestReconciler_UnmountAlreadyDetachedNode(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// Mount a Box with two children
	Render(Box(BoxProps{ID: "app"},
		Span(SpanProps{ID: "child1"}, Text("Hello")),
		Span(SpanProps{ID: "child2"}, Text("World")),
	), container)

	appReal := container.FirstChild().(dom.Element)
	child1Real := appReal.FirstChild()

	// Manually remove child1 from appReal outside of kitex (direct DOM manipulation)
	appReal.RemoveChild(child1Real)

	// Now render with child1 removed in VDOM
	// This will trigger unmounting of child1 in reconciler. Since child1 is already detached,
	// it should NOT panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reconciler panicked when unmounting already detached node: %v", r)
		}
	}()

	Render(Box(BoxProps{ID: "app"},
		Span(SpanProps{ID: "child2"}, Text("World")),
	), container)
}

func TestReconciler_ComponentNilRenderUnmount(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	var setRenderNil func(bool)
	var triggerStateUpdate func()

	InnerComp := FC("InnerComp", func(props struct{}) Node {
		renderNil, setRN := UseState(false)
		setRenderNil = setRN

		// A dummy state update function to simulate async event on this component
		_, setDummy := UseState(0)
		triggerStateUpdate = func() {
			setDummy(1)
		}

		if renderNil() {
			return nil
		}
		return Span(SpanProps{ID: "child"})
	})

	var setShowParentComp func(bool)

	ParentComp := FC("ParentComp", func(props struct{}) Node {
		show, setShow := UseState(true)
		setShowParentComp = setShow
		if !show() {
			return Box(BoxProps{ID: "empty-box"})
		}
		return Box(BoxProps{ID: "app"}, InnerComp(struct{}{}))
	})

	// 1. Initial Render
	Render(ParentComp(struct{}{}), container)

	appReal := container.FirstChild().(dom.Element)
	if appReal.FirstChild() == nil {
		t.Fatalf("expected child to be mounted initially")
	}

	// 2. Make InnerComp render nil
	setRenderNil(true)

	if appReal.FirstChild() != nil {
		t.Fatalf("expected child to be unmounted when InnerComp renders nil")
	}

	// 3. Unmount the entire branch containing InnerComp
	setShowParentComp(false)

	// Verify that the tree updated
	boxReal := container.FirstChild().(dom.Element)
	if boxReal.ID() != "empty-box" {
		t.Fatalf("expected empty-box, got %s", boxReal.ID())
	}

	// 4. Trigger state update on the now-unmounted InnerComp.
	// Since destroyNode was called, InnerComp's componentRef has been cleared
	// and its parent DOM is cleared (so OnComponentDirty returns early or has no effect).
	// But even if processDirtyLoop runs, it should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reconciler panicked after state update on unmounted nil-rendering component: %v", r)
		}
	}()

	triggerStateUpdate()
}

func TestReconciler_Case2InsertBeforeRefOutdated(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setCondition func(bool)

	// Inner Component that returns either a Span or a Box based on a condition
	DynamicComp := FC("DynamicComp", func(props struct{}) Node {
		cond, setCond := UseState(true)
		setCondition = setCond
		if cond() {
			return Span(SpanProps{ID: "child-span"})
		}
		return Box(BoxProps{ID: "child-box"})
	})

	// Initial render:
	// A list: [DynamicComp]
	// DynamicComp renders to a Span.
	Render(Box(BoxProps{ID: "parent"}, DynamicComp(struct{}{})), container)

	parentReal := container.FirstChild().(dom.Element)
	if parentReal.FirstChild() == nil || parentReal.FirstChild().(dom.Element).TagName() != "span" {
		t.Fatalf("expected span child initially")
	}

	// Update render:
	// A list: [Span(ID: "new-span"), DynamicComp]
	// In the same pass, setCondition is false, so DynamicComp renders to a Box.
	// Since DynamicComp is at the end of both old and new lists, Case 2 matches:
	// oldEndNode (DynamicComp) == newEndNode (DynamicComp).
	// During Case 2's reconcile:
	// DynamicComp updates from rendering Span to Box. The Span is replaced with a Box (Case 3 inside the nested reconcile).
	// Thus, the old real node (the Span) is unmounted and removed.
	// The new real node is a Box.
	// Then, Case 2 finishes, and we have the remaining new child (new-span) to insert.
	// The reconciler inserts it before the first real node of DynamicComp.
	// If insertBeforeRef was not updated, it would try to insert before the old detached Span, causing a panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reconciler panicked during insertion before updated end node: %v", r)
		}
	}()

	// We trigger both: set condition to false AND prepend a new span.
	setCondition(false)
	Render(Box(BoxProps{ID: "parent"},
		Span(SpanProps{ID: "new-span"}),
		DynamicComp(struct{}{}),
	), container)

	// Verify the final DOM structure
	var childIDs []string
	for child := parentReal.FirstChild(); child != nil; child = child.NextSibling() {
		childIDs = append(childIDs, child.(dom.Element).ID())
	}
	expectedIDs := []string{"new-span", "child-box"}
	if !reflect.DeepEqual(childIDs, expectedIDs) {
		t.Errorf("expected child IDs %v, got %v", expectedIDs, childIDs)
	}
}

func TestReconciler_ClearAllSubscriptionsRecursive(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	innerRef := CreateRef[dom.Node]()
	childRef := CreateRef[dom.Node]()

	root := Box(BoxProps{},
		Box(BoxProps{ID: "inner-box", OnClick: func(e event.Event) {}, Ref: innerRef},
			Span(SpanProps{ID: "child-span", OnClick: func(e event.Event) {}, Ref: childRef}),
		),
	)

	Render(root, container)

	innerBoxReal := innerRef.Current
	childSpanReal := childRef.Current

	if innerBoxReal == nil || childSpanReal == nil {
		t.Fatalf("expected real DOM elements to be captured via Ref")
	}

	// Lock the subMutex to safely read subscriptions
	subMutex.Lock()
	_, hasInner := subscriptions[innerBoxReal]
	_, hasChild := subscriptions[childSpanReal]
	subMutex.Unlock()

	if !hasInner {
		t.Error("expected inner box to have registered event subscriptions")
	}
	if !hasChild {
		t.Error("expected child span to have registered event subscriptions")
	}

	// Now unmount the root
	Render(nil, container)

	// Verify that both have been removed from the global subscriptions map (no memory leak)
	subMutex.Lock()
	_, hasInnerAfter := subscriptions[innerBoxReal]
	_, hasChildAfter := subscriptions[childSpanReal]
	subMutex.Unlock()

	if hasInnerAfter {
		t.Error("expected inner box subscriptions to be cleaned up after unmount")
	}
	if hasChildAfter {
		t.Error("expected child span subscriptions to be cleaned up after unmount")
	}
}

func TestReconciler_Case4InsertBeforeRefFallback(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	EmptyComp := FC("EmptyComp", func(props struct{}) Node {
		return nil
	})

	// Initial render: [EmptyComp, EndNode, FixedEnd]
	Render(Box(BoxProps{ID: "parent"},
		EmptyComp(struct{}{}),
		Span(SpanProps{ID: "end"}),
		Span(SpanProps{ID: "fixed-end"}),
	), container)

	parentReal := container.FirstChild().(dom.Element)
	var initialIDs []string
	for child := parentReal.FirstChild(); child != nil; child = child.NextSibling() {
		initialIDs = append(initialIDs, child.(dom.Element).ID())
	}
	expectedInit := []string{"end", "fixed-end"}
	if !reflect.DeepEqual(initialIDs, expectedInit) {
		t.Fatalf("expected initial IDs %v, got %v", expectedInit, initialIDs)
	}

	// Reconcile to: [EndNode, NewNode, FixedEnd]
	// This triggers Case 2 for FixedEnd (updating insertBeforeRef to FixedEnd).
	// Then Case 4 for EndNode (oldEndNode == newStartNode).
	// EndNode is moved before EmptyComp (which has no real DOM nodes).
	// It must fall back to insertBeforeRef (FixedEnd).
	// If it doesn't, EndNode is inserted at the end (after FixedEnd).
	Render(Box(BoxProps{ID: "parent"},
		Span(SpanProps{ID: "end"}),
		Span(SpanProps{ID: "new-node"}),
		Span(SpanProps{ID: "fixed-end"}),
	), container)

	var finalIDs []string
	for child := parentReal.FirstChild(); child != nil; child = child.NextSibling() {
		finalIDs = append(finalIDs, child.(dom.Element).ID())
	}
	expectedFinal := []string{"end", "new-node", "fixed-end"}
	if !reflect.DeepEqual(finalIDs, expectedFinal) {
		t.Errorf("expected final IDs %v, got %v", expectedFinal, finalIDs)
	}
}

func TestReconciler_Case5InsertBeforeRefFallback(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	EmptyComp := FC("EmptyComp", func(props struct{}) Node {
		return nil
	})

	// Initial render: [EmptyComp, MatchedNode, FixedEnd]
	Render(Box(BoxProps{ID: "parent"},
		EmptyComp(struct{}{}),
		Span(SpanProps{ID: "matched", Key: "m"}),
		Span(SpanProps{ID: "fixed-end"}),
	), container)

	parentReal := container.FirstChild().(dom.Element)
	var initialIDs []string
	for child := parentReal.FirstChild(); child != nil; child = child.NextSibling() {
		initialIDs = append(initialIDs, child.(dom.Element).ID())
	}
	t.Logf("Initial DOM IDs: %v", initialIDs)

	// Reconcile to: [MatchedNode, NewNode, FixedEnd]
	Render(Box(BoxProps{ID: "parent"},
		Span(SpanProps{ID: "matched", Key: "m"}),
		Span(SpanProps{ID: "new-node"}),
		Span(SpanProps{ID: "fixed-end"}),
	), container)

	var finalIDs []string
	for child := parentReal.FirstChild(); child != nil; child = child.NextSibling() {
		finalIDs = append(finalIDs, child.(dom.Element).ID())
	}
	t.Logf("Final DOM IDs: %v", finalIDs)

	expectedFinal := []string{"matched", "new-node", "fixed-end"}
	if !reflect.DeepEqual(finalIDs, expectedFinal) {
		t.Errorf("expected final IDs %v, got %v", expectedFinal, finalIDs)
	}
}

func TestReconcilerTagMismatchOrder(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// Frame 1: Box
	Render(Box(BoxProps{ID: "target-box"}), container)
	parentReal := container
	if parentReal.FirstChild() == nil || parentReal.FirstChild().(dom.Element).TagName() != "box" {
		t.Fatalf("expected box child initially")
	}

	// Frame 2: Span (tag mismatch). Replaces Box in place.
	Render(Span(SpanProps{ID: "target-span"}), container)
	if parentReal.FirstChild() == nil || parentReal.FirstChild().(dom.Element).TagName() != "span" {
		t.Fatalf("expected span child after tag mismatch replacement")
	}
}

func TestReconcilerKeylessWarnings(t *testing.T) {
	EnableDevMode = true
	defer func() { EnableDevMode = false }()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)

	oldLogger := log.Logger()
	log.SetLogger(logger)
	defer log.SetLogger(oldLogger)

	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// Test 1: Fragment returned by component with keyless child elements
	KeylessComp := FC("KeylessComp", func(props struct{}) Node {
		return Fragment(
			Span(SpanProps{ID: "child1"}),
			Span(SpanProps{ID: "child2"}),
		)
	})

	Render(KeylessComp(struct{}{}), container)
	if !strings.Contains(buf.String(), "Component returned a fragment containing child nodes without explicit keys") {
		t.Errorf("expected warning for keyless fragment siblings, got log: %s", buf.String())
	}
	buf.Reset()

	// Test 2: Tag mismatch replacement on keyless root nodes
	Render(Span(SpanProps{ID: "keyless-span"}), container)
	buf.Reset()
	Render(Box(BoxProps{ID: "keyless-box"}), container)
	if !strings.Contains(buf.String(), "Keyless type mismatch replacement detected") {
		t.Errorf("expected warning for keyless type mismatch replacement, got log: %s", buf.String())
	}
	buf.Reset()

	// Test 3: Keyless node shifting in dynamic list reconciliation
	// We set up a list where keyless nodes shift.
	// Frame 1: [Span1, Box1, Span2]
	Render(Box(BoxProps{ID: "list-parent"},
		Span(SpanProps{ID: "s1"}),
		Box(BoxProps{ID: "b1"}),
		Span(SpanProps{ID: "s2"}),
	), container)
	buf.Reset()

	// Frame 2: [Box2, Span1, Box1, Box3] - this forces Box1 to shift
	Render(Box(BoxProps{ID: "list-parent"},
		Box(BoxProps{ID: "b2"}),
		Span(SpanProps{ID: "s1"}),
		Box(BoxProps{ID: "b1"}),
		Box(BoxProps{ID: "b3"}),
	), container)

	if !strings.Contains(buf.String(), "Keyless node shifting detected") {
		t.Errorf("expected warning for keyless node shifting, got log: %s", buf.String())
	}
}

func TestReconcilerDirtyBufferSharing(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	var runCountA int
	var runCountB int
	var runCountC int
	var runCountD int
	var runCountE int

	var setA func(int)
	var setB func(int)
	var setC func(int)
	var setD func(int)
	var setE func(int)

	CompE := FC("CompE", func(props struct{}) Node {
		state, set := UseState(0)
		setE = set
		runCountE++
		return Span(SpanProps{ID: fmt.Sprintf("e-%d", state())})
	})

	CompD := FC("CompD", func(props struct{}) Node {
		state, set := UseState(0)
		setD = set
		runCountD++
		return Span(SpanProps{ID: fmt.Sprintf("d-%d", state())})
	})

	CompC := FC("CompC", func(props struct{}) Node {
		state, set := UseState(0)
		setC = set
		runCountC++
		return Span(SpanProps{ID: fmt.Sprintf("c-%d", state())})
	})

	CompB := FC("CompB", func(props struct{}) Node {
		state, set := UseState(0)
		setB = set
		runCountB++
		if state() > 0 {
			// Trigger two state updates to overwrite index 1 (CompA)
			setC(1)
			setE(1)
		}
		return Span(SpanProps{ID: fmt.Sprintf("b-%d", state())})
	})

	CompA := FC("CompA", func(props struct{}) Node {
		state, set := UseState(0)
		setA = set
		runCountA++
		return Span(SpanProps{ID: fmt.Sprintf("a-%d", state())})
	})

	// Root triggers queueing
	var setTrigger func(bool)

	Root := FC("Root", func(props struct{}) Node {
		state, set := UseState(false)
		setTrigger = set
		UseLayoutEffect(func() {
			if state() {
				// Queue B, A, D
				setB(1)
				setA(1)
				setD(1)
			}
		}, []any{state()})
		return Box(BoxProps{},
			CompB(struct{}{}),
			CompA(struct{}{}),
			CompD(struct{}{}),
			CompC(struct{}{}),
			CompE(struct{}{}),
		)
	})

	Render(Root(struct{}{}), container)

	// Reset counters
	runCountA = 0
	runCountB = 0
	runCountC = 0
	runCountD = 0
	runCountE = 0

	// Trigger layout effect to queue B, A, D
	setTrigger(true)

	// Kick off the dirty loop
	setB(2)

	if runCountA == 0 {
		t.Errorf("CompA was skipped during dirty component processing (likely due to dirty queue buffer sharing corruption)!")
	}
	if runCountB == 0 {
		t.Errorf("CompB was not processed!")
	}
	if runCountD == 0 {
		t.Errorf("CompD was not processed!")
	}
}

func TestReconciler_ComponentStyleUpdate(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setStyleFn func(style.Style)

	StyleComp := FC("StyleComp", func(props struct{}) Node {
		getStyle, setStyle := UseState(style.S())
		setStyleFn = setStyle
		return Span(SpanProps{ID: "target", Style: getStyle()})
	})

	Render(StyleComp(struct{}{}), container)

	spanReal := container.FirstChild().(dom.Element)
	if spanReal.ID() != "target" {
		t.Fatalf("expected span with ID 'target'")
	}

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	setStyleFn(style.S().Foreground(red))

	// Verify that the style on the DOM element was updated!
	var baseComputed style.Computed
	applied := spanReal.RawStyle().Apply(baseComputed)
	if applied.Foreground != red {
		t.Errorf("expected Foreground to be updated to red, got %v", applied.Foreground)
	}
}

func TestReconciler_ComponentEventListenerUpdate(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setValFn func(int)
	var calledWithVal int

	ClickComp := FC("ClickComp", func(props struct{}) Node {
		getVal, setVal := UseState(10)
		setValFn = setVal
		return Button(ButtonProps{
			OnClick: func(e event.Event) {
				calledWithVal = getVal()
			},
		})
	})

	Render(ClickComp(struct{}{}), container)

	btnReal := container.FirstChild().(*element.ButtonElement)

	// Trigger click on btnReal
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 10 {
		t.Errorf("expected calledWithVal to be 10, got %d", calledWithVal)
	}

	setValFn(20)

	// Trigger click on btnReal again
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 20 {
		t.Errorf("expected calledWithVal to be updated to 20, got %d (closure was not updated)", calledWithVal)
	}

}

func TestReconciler_ComponentNestedEventListenerUpdate(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setValFn func(int)
	var calledWithVal int

	InnerComp := FC("InnerComp", func(props struct{ OnClick func(event.Event) }) Node {
		return Button(ButtonProps{
			OnClick: props.OnClick,
		})
	})

	ClickComp := FC("ClickComp", func(props struct{}) Node {
		getVal, setVal := UseState(10)
		setValFn = setVal
		return Fragment(
			InnerComp(struct{ OnClick func(event.Event) }{
				OnClick: func(e event.Event) {
					calledWithVal = getVal()
				},
			}),
		)
	})

	Render(ClickComp(struct{}{}), container)

	btnReal := container.FirstChild().(*element.ButtonElement)

	// Trigger click on btnReal
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 10 {
		t.Errorf("expected calledWithVal to be 10, got %d", calledWithVal)
	}

	setValFn(20)

	// Trigger click on btnReal again
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 20 {
		t.Errorf("expected calledWithVal to be updated to 20, got %d (closure was not updated)", calledWithVal)
	}

}

func TestReconciler_ComponentStaleClosureValue(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var setValFn func(int)
	var calledWithVal int

	ClickComp := FC("ClickComp", func(props struct{}) Node {
		getVal, setVal := UseState(10)
		setValFn = setVal
		val := getVal() // calls getter during render
		return Button(ButtonProps{
			OnClick: func(e event.Event) {
				calledWithVal = val // captures the value, not the getter!
			},
		})
	})

	Render(ClickComp(struct{}{}), container)

	btnReal := container.FirstChild().(*element.ButtonElement)

	// Trigger click on btnReal
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 10 {
		t.Errorf("expected calledWithVal to be 10, got %d", calledWithVal)
	}

	setValFn(20)

	// Trigger click on btnReal again
	btnReal.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
	if calledWithVal != 20 {
		t.Errorf("expected calledWithVal to be updated to 20, got %d (closure was not updated)", calledWithVal)
	}

	ClearAllSubscriptions(btnReal)
}

func TestReconcilerConditionalEndItem(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// Frame 1: cond is false, only span-1 is present
	Render(Box(BoxProps{},
		Span(SpanProps{ID: "span-1"}),
		If(false, func() Node { return Span(SpanProps{ID: "span-cond"}) }),
	), container)

	// Verify initial state
	realParent := container.FirstChild().(dom.Element)
	var ids []string
	for child := realParent.FirstChild(); child != nil; child = child.NextSibling() {
		ids = append(ids, child.(dom.Element).ID())
	}
	if len(ids) != 1 || ids[0] != "span-1" {
		t.Fatalf("expected initial children [span-1], got %v", ids)
	}

	// Frame 2: cond is true, new span-new is added before the conditional end item
	Render(Box(BoxProps{},
		Span(SpanProps{ID: "span-1"}),
		Span(SpanProps{ID: "span-new"}),
		If(true, func() Node { return Span(SpanProps{ID: "span-cond"}) }),
	), container)

	// Verify final state order
	ids = ids[:0]
	for child := realParent.FirstChild(); child != nil; child = child.NextSibling() {
		ids = append(ids, child.(dom.Element).ID())
	}
	expected := []string{"span-1", "span-new", "span-cond"}
	if !slices.Equal(ids, expected) {
		t.Errorf("expected children order %v, got %v", expected, ids)
	}
}

func TestReconcilerFragmentConditionalEndItem(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{ID: "container"}).Instantiate(doc)[0].(dom.Element)

	// Frame 1: Fragment has span-1, cond is false
	Render(Box(BoxProps{},
		Fragment(
			Span(SpanProps{ID: "span-1"}),
		),
		If(false, func() Node { return Span(SpanProps{ID: "span-cond"}) }),
	), container)

	realParent := container.FirstChild().(dom.Element)
	var ids []string
	for child := realParent.FirstChild(); child != nil; child = child.NextSibling() {
		ids = append(ids, child.(dom.Element).ID())
	}
	if len(ids) != 1 || ids[0] != "span-1" {
		t.Fatalf("expected initial children [span-1], got %v", ids)
	}

	// Frame 2: Fragment appends span-new, cond becomes true
	Render(Box(BoxProps{},
		Fragment(
			Span(SpanProps{ID: "span-1"}),
			Span(SpanProps{ID: "span-new"}),
		),
		If(true, func() Node { return Span(SpanProps{ID: "span-cond"}) }),
	), container)

	ids = ids[:0]
	for child := realParent.FirstChild(); child != nil; child = child.NextSibling() {
		ids = append(ids, child.(dom.Element).ID())
	}
	expected := []string{"span-1", "span-new", "span-cond"}
	if !slices.Equal(ids, expected) {
		t.Errorf("expected children order %v, got %v", expected, ids)
	}
}
