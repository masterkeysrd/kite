package dom

import (
	"strings"

	"github.com/masterkeysrd/kite/dom"
)

func (e *Element) QuerySelector(selector string) dom.Element {
	if selector == "" {
		return nil
	}
	return querySelector(e.self, selector)
}

func (d *Document) QuerySelector(selector string) dom.Element {
	if selector == "" {
		return nil
	}
	// Search main document tree
	if found := querySelector(d.self, selector); found != nil {
		return found
	}
	// Search overlays
	for _, o := range d.overlays {
		if found := querySelector(o.el, selector); found != nil {
			return found
		}
	}
	return nil
}

func querySelector(n dom.Node, selector string) dom.Element {
	if el, ok := n.(dom.Element); ok {
		if matches(el, selector) {
			return el
		}
	}

	for child := range n.ChildNodes() {
		if found := querySelector(child, selector); found != nil {
			return found
		}
	}
	return nil
}

func matches(el dom.Element, selector string) bool {
	if strings.HasPrefix(selector, "#") {
		return el.ID() == selector[1:]
	}
	if strings.HasPrefix(selector, ".") {
		return el.Class() == selector[1:]
	}
	return el.TagName() == selector
}
