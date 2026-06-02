package kitex

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
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
			If(cond(), Span(SpanProps{ID: "conditional-span"})),
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
