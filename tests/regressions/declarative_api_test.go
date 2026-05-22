package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
)

func TestDeclarative_NestedSlices(t *testing.T) {
	e := testenv.Default(10, 5)
	defer e.Close()

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

	e.Mount(tree)
	e.RenderFrame()

	// Assertions
	testenv.Expect(t, tree).
		ToHaveChildCount(3).
		ToHaveChildrenText([]string{"a", "b", "c"})
}
