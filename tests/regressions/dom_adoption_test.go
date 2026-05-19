package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func TestAdoption_GetElementByID(t *testing.T) {
	doc1 := dom.NewDocument()
	doc2 := dom.NewDocument()

	// Create a detached subtree in doc2.
	root := doc2.CreateElement("div", nil)
	child := doc2.CreateElement("span", nil)
	child.SetID("adopted-node")
	root.AppendChild(child)

	// doc2 should not have it yet because it's detached.
	if doc2.GetElementByID("adopted-node") != nil {
		t.Fatal("node should not be registered in doc2 while detached")
	}

	// Move the detached subtree to doc1.
	doc1.AppendChild(root)

	// Now doc1 should have it because root is connected to doc1.
	if got := doc1.GetElementByID("adopted-node"); got != child {
		t.Errorf("GetElementByID in doc1 = %v, want %v", got, child)
	}

	// And it should have the correct owner document.
	if child.OwnerDocument() != doc1 {
		t.Errorf("child owner document = %v, want %v", child.OwnerDocument(), doc1)
	}
}
