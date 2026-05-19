package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
)

func TestDeclarative_NestedSlices(t *testing.T) {
	// Nested variadic slices: Box([]any{Box("a"), []any{Box("b"), Box("c")}})
	tree := element.Box(
		[]any{
			element.Box("a"),
			[]any{
				element.Box("b"),
				element.Box("c"),
			},
		},
	)

	// Count children
	count := 0
	for range tree.ChildNodes() {
		count++
	}

	if count != 3 {
		t.Errorf("expected 3 children, got %d", count)
	}

	// Verify child content
	expected := []string{"a", "b", "c"}
	i := 0
	for child := range tree.ChildNodes() {
		if child.TextContent() != expected[i] {
			t.Errorf("child %d text = %q, want %q", i, child.TextContent(), expected[i])
		}
		i++
	}
}
