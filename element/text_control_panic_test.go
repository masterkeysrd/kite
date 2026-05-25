package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
)

// TestTextArea_Panic_DocumentMismatch verifies that adopting a TextArea into a
// new document does not cause a panic during selection updates due to
// mismatched owner documents on UA nodes.
func TestTextArea_Panic_DocumentMismatch(t *testing.T) {
	doc1 := dom.NewDocument()
	txa := element.NewTextArea(doc1, "Hello")

	// Move to doc2.
	doc2 := dom.NewDocument()
	doc2.AppendChild(txa)

	// Mutate buffer to trigger rebuild of UA subtree in doc2.
	txa.Buffer().Insert(" World")
	txa.SyncBuffer()

	// Trigger selection update. If it uses nodes from doc1, it will panic.
	txa.SetSelectionRange(0, 5)

	// If we reached here without panic, the fix is verified.
}

// TestTextArea_Panic_InvalidBROffset verifies that mapping a selection to a
// <br> element does not use an invalid offset (1), which would exceed the
// child count (0) of the void <br> element and cause a panic.
func TestTextArea_Panic_InvalidBROffset(t *testing.T) {
	doc := dom.NewDocument()
	txa := element.NewTextArea(doc, "Line One\nLine Two")
	root := element.Box(txa)
	doc.AppendChild(root)

	// Layout is needed for resolveOffset to work (it iterates ChildNodes,
	// but textControlBase.resolveOffset uses the uaDiv's children which are
	// created during rebuildUASubtree).

	// Offset 8 is the '\n' character.
	// We want to ensure that setting selection at or after this '\n' works.

	// Test setting selection exactly at the \n.
	txa.SetSelectionRange(8, 9)

	// Test setting selection starting exactly at the \n.
	txa.SetSelectionRange(8, 10)

	// If no panic, we are good.
}
