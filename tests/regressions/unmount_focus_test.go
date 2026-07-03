package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/testenv"
)

// TestUnmountFocusReset verifies that if the currently focused element is
// unmounted and no other element remains in the document, focus is cleared to nil.
func TestUnmountFocusReset(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	doc := env.Engine.Document()
	container := element.Box()
	btn := element.Button("Click Me")
	container.AppendChild(btn)
	doc.AppendChild(container)
	env.Flush()

	// 1. Focus the button
	btn.Focus()
	if doc.CurrentFocus() != btn {
		t.Fatalf("expected button to be focused, got %v", doc.CurrentFocus())
	}

	// 2. Remove the container (which contains the focused button)
	doc.RemoveChild(container)
	env.Flush()

	// 3. Verify that focus is cleared/reset to nil because no root element exists
	if doc.CurrentFocus() != nil {
		t.Fatalf("expected focus to be cleared/reset to nil after unmounting focused element with no root remaining, got %v", doc.CurrentFocus())
	}
}

// TestUnmountFocusResetsToNilIfNoFocusableAncestor verifies that if a focused element is unmounted
// and no focusable parent exists in its ancestry, focus is cleared to nil (allowing autofocus recovery).
func TestUnmountFocusResetsToNilIfNoFocusableAncestor(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	doc := env.Engine.Document()
	rootBox := element.Box()
	subContainer := element.Box()
	btn := element.Button("Click Me")

	subContainer.AppendChild(btn)
	rootBox.AppendChild(subContainer)
	doc.AppendChild(rootBox)
	env.Flush()

	// 1. Focus the button
	btn.Focus()
	if doc.CurrentFocus() != btn {
		t.Fatalf("expected button to be focused, got %v", doc.CurrentFocus())
	}

	// 2. Remove the subContainer containing the button (rootBox remains connected but is not focusable)
	rootBox.RemoveChild(subContainer)
	env.Flush()

	// 3. Verify that focus is cleared/reset to nil
	if doc.CurrentFocus() != nil {
		t.Fatalf("expected focus to be cleared/reset to nil, got %v", doc.CurrentFocus())
	}
}

// TestUnmountFocusFallbackToFocusableAncestor verifies that if a focused element is unmounted
// but a parent container in the unmounted path's parent chain is focusable, focus falls back
// to that nearest focusable ancestor instead of the root Box.
func TestUnmountFocusFallbackToFocusableAncestor(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	doc := env.Engine.Document()
	rootBox := element.Box()

	// Create an ancestor container that is focusable (TabIndex = 0)
	focusableAncestor := element.Box()
	focusableAncestor.SetTabIndex(0)

	subContainer := element.Box()
	btn := element.Button("Click Me")

	subContainer.AppendChild(btn)
	focusableAncestor.AppendChild(subContainer)
	rootBox.AppendChild(focusableAncestor)
	doc.AppendChild(rootBox)
	env.Flush()

	// 1. Focus the button
	btn.Focus()
	if doc.CurrentFocus() != btn {
		t.Fatalf("expected button to be focused, got %v", doc.CurrentFocus())
	}

	// 2. Remove the subContainer containing the button (focusableAncestor remains connected)
	focusableAncestor.RemoveChild(subContainer)
	env.Flush()

	// 3. Verify that focus falls back to the nearest focusable ancestor (focusableAncestor)
	t.Logf("rootBox: %p, focusableAncestor: %p, got: %p", rootBox, focusableAncestor, doc.CurrentFocus())
	if doc.CurrentFocus() != focusableAncestor {
		t.Fatalf("expected focus to fall back to the focusable ancestor, got %v", doc.CurrentFocus())
	}
}
