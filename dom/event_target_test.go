package dom_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func TestNode_EventTarget(t *testing.T) {
	doc := dom.NewDocument()

	t.Run("StandardNode", func(t *testing.T) {
		el := doc.CreateElement("div", nil)
		if target := el.EventTarget(); target != el {
			t.Errorf("Standard node: want target %p, got %p", el, target)
		}
	})

	t.Run("UASubtreeNode", func(t *testing.T) {
		host := doc.CreateElement("host", nil)
		uaRoot := doc.CreateElement("ua-root", nil)
		uaChild := doc.CreateElement("ua-child", nil)
		uaRoot.AppendChild(uaChild)

		host.AttachUARoot(uaRoot)

		if target := uaRoot.EventTarget(); target != host {
			t.Errorf("UA root: want target %p (host), got %p", host, target)
		}

		if target := uaChild.EventTarget(); target != host {
			t.Errorf("UA child: want target %p (host), got %p", host, target)
		}
	})

	t.Run("TextNodeInUASubtree", func(t *testing.T) {
		host := doc.CreateElement("host", nil)
		uaText := doc.CreateTextNode("hello", nil)

		host.AttachUARoot(uaText)

		if target := uaText.EventTarget(); target != host {
			t.Errorf("UA text node: want target %p (host), got %p", host, target)
		}
	})

	t.Run("DeeplyNestedUAInsertion", func(t *testing.T) {
		host := doc.CreateElement("host", nil)
		uaRoot := doc.CreateElement("ua-root", nil)
		host.AttachUARoot(uaRoot)

		// Insert node after AttachUARoot
		nested := doc.CreateElement("nested", nil)
		uaRoot.AppendChild(nested)

		if target := nested.EventTarget(); target != host {
			t.Errorf("Newly inserted UA child: want target %p (host), got %p", host, target)
		}
	})
}
