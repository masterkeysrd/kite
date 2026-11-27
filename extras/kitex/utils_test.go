package kitex

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

// --- Map ---

func TestMap_Basic(t *testing.T) {
	items := []string{"a", "b", "c"}
	node := Map(items, func(item string, i int) Node {
		return Text(fmt.Sprintf("%d:%s", i, item))
	})
	frag, ok := node.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", node)
	}
	nodes := frag.children
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	for j, n := range nodes {
		want := fmt.Sprintf("%d:%s", j, items[j])
		if n.Props().(string) != want {
			t.Errorf("nodes[%d]: want %q, got %q", j, want, n.Props().(string))
		}
	}
}

func TestMap_FiltersNil(t *testing.T) {
	items := []int{1, 2, 3, 4}
	node := Map(items, func(item int, _ int) Node {
		if item%2 == 0 {
			return nil // skip evens
		}
		return Text(fmt.Sprintf("%d", item))
	})
	frag, ok := node.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", node)
	}
	nodes := frag.children
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (odd items only), got %d", len(nodes))
	}
}

func TestMap_EmptySlice(t *testing.T) {
	node := Map([]string{}, func(s string, _ int) Node { return Text(s) })
	frag, ok := node.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", node)
	}
	nodes := frag.children
	if len(nodes) != 0 {
		t.Fatalf("expected empty slice, got %d nodes", len(nodes))
	}
}

func TestMap_IndexProvided(t *testing.T) {
	capturedIndices := make([]int, 0)
	Map([]string{"x", "y", "z"}, func(_ string, i int) Node {
		capturedIndices = append(capturedIndices, i)
		return Text("x")
	})
	for j, idx := range capturedIndices {
		if idx != j {
			t.Errorf("expected index %d, got %d", j, idx)
		}
	}
}

// --- Nodes ---

func TestNodes_MergesGroups(t *testing.T) {
	g1 := Fragment(Text("a"), Text("b"))
	g2 := Text("c")
	merged := Nodes(g1, g2)
	frag, ok := merged.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", merged)
	}
	nodes := frag.children
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestNodes_FiltersNil(t *testing.T) {
	g := Fragment(Text("a"), nil, Text("b"), nil)
	merged := Nodes(g)
	frag, ok := merged.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", merged)
	}
	nodes := frag.children
	if len(nodes) != 2 {
		t.Fatalf("expected 2 non-nil nodes, got %d", len(nodes))
	}
}

func TestNodes_Empty(t *testing.T) {
	merged := Nodes()
	if merged == nil {
		t.Fatal("Nodes() should return non-nil Node")
	}
	frag, ok := merged.(*fragmentNode)
	if !ok {
		t.Fatalf("expected Fragment, got %T", merged)
	}
	nodes := frag.children
	if len(nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(nodes))
	}
}

// --- If ---

func TestIf_True(t *testing.T) {
	n := If(true, func() Node { return Text("visible") })
	if n == nil {
		t.Fatal("If(true, fn) should return node")
	}
}

func TestIf_False(t *testing.T) {
	nFalse := If(false, func() Node { return Text("hidden") })
	if nFalse == nil || nFalse.TagName() != "#empty" {
		t.Errorf("If(false, fn) should return an emptyNode, got %v", nFalse)
	}
}

func TestIf_NilNodePassthrough(t *testing.T) {
	// Even if the returned node is nil, the function should still return it.
	n := If(true, func() Node { return nil })
	if n != nil {
		t.Fatal("If(true, returns nil) should return nil")
	}
}

// --- IfElse ---

func TestIfElse_True(t *testing.T) {
	n := IfElse(true, Text("then"), Text("else"))
	if n == nil || n.Props().(string) != "then" {
		t.Fatal("IfElse(true,...) should return then-node")
	}
}

func TestIfElse_False(t *testing.T) {
	n := IfElse(false, Text("then"), Text("else"))
	if n == nil || n.Props().(string) != "else" {
		t.Fatal("IfElse(false,...) should return else-node")
	}
}

// --- IfElseFn ---

func TestIfElseFn_LazyEvaluation(t *testing.T) {
	t.Run("true branch only invokes thenFn", func(t *testing.T) {
		thenCalls, elseCalls := 0, 0
		n := IfElseFn(true,
			func() Node { thenCalls++; return Text("then") },
			func() Node { elseCalls++; return Text("else") },
		)
		if thenCalls != 1 || elseCalls != 0 {
			t.Fatalf("expected thenCalls=1, elseCalls=0; got %d, %d", thenCalls, elseCalls)
		}
		if n == nil || n.Props().(string) != "then" {
			t.Fatalf("expected 'then' node, got %v", n)
		}
	})

	t.Run("false branch only invokes elseFn", func(t *testing.T) {
		thenCalls, elseCalls := 0, 0
		n := IfElseFn(false,
			func() Node { thenCalls++; return Text("then") },
			func() Node { elseCalls++; return Text("else") },
		)
		if thenCalls != 0 || elseCalls != 1 {
			t.Fatalf("expected thenCalls=0, elseCalls=1; got %d, %d", thenCalls, elseCalls)
		}
		if n == nil || n.Props().(string) != "else" {
			t.Fatalf("expected 'else' node, got %v", n)
		}
	})

	t.Run("nil closure or nil return yields Empty", func(t *testing.T) {
		if n := IfElseFn(true, nil, func() Node { return Text("x") }); n == nil || n.TagName() != "#empty" {
			t.Errorf("expected #empty for nil thenFn, got %v", n)
		}
		if n := IfElseFn(false, func() Node { return Text("x") }, nil); n == nil || n.TagName() != "#empty" {
			t.Errorf("expected #empty for nil elseFn, got %v", n)
		}
		if n := IfElseFn(true, func() Node { return nil }, nil); n == nil || n.TagName() != "#empty" {
			t.Errorf("expected #empty when closure returns nil, got %v", n)
		}
	})
}

// --- Fragment ---

func TestFragment_Basic(t *testing.T) {
	nodes := Fragment(Text("a"), Text("b"), Text("c")).Children()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestFragment_FiltersNil(t *testing.T) {
	nodes := Fragment(Text("a"), nil, Text("b")).Children()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestFragment_Empty(t *testing.T) {
	nodes := Fragment().Children()
	if len(nodes) != 0 {
		t.Fatalf("expected empty fragment, got %d", len(nodes))
	}
}

// --- Integration: Map + spread into Box ---

func TestMap_SpreadIntoBox(t *testing.T) {
	doc := dom.NewDocument()

	type Row struct {
		Key  string
		Text string
	}
	rows := []Row{
		{Key: "1", Text: "First"},
		{Key: "2", Text: "Second"},
		{Key: "3", Text: "Third"},
	}

	vdom := Box(BoxProps{},
		Map(rows, func(r Row, _ int) Node {
			return Span(SpanProps{Key: r.Key}, Text(r.Text))
		}),
	)

	real := vdom.Instantiate(doc)[0].(dom.Element)
	count := 0
	for child := real.FirstChild(); child != nil; child = child.NextSibling() {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 rendered children, got %d", count)
	}
}

func TestIf_SpreadIntoBox(t *testing.T) {
	doc := dom.NewDocument()

	render := func(show bool) dom.Element {
		vdom := Box(BoxProps{},
			Text("always"),
			If(show, func() Node { return Text("conditional") }),
		)
		return vdom.Instantiate(doc)[0].(dom.Element)
	}

	// With show=true: 2 children
	el := render(true)
	count := 0
	for c := el.FirstChild(); c != nil; c = c.NextSibling() {
		count++
	}
	if count != 2 {
		t.Fatalf("show=true: expected 2 children, got %d", count)
	}
}
