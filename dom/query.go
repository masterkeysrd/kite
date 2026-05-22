package dom

import "strings"

func (e *element) QuerySelector(selector string) Element {
	if selector == "" {
		return nil
	}
	return querySelector(e.self, selector)
}

func (d *document) QuerySelector(selector string) Element {
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

func querySelector(n Node, selector string) Element {
	if el, ok := n.(Element); ok {
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

func matches(el Element, selector string) bool {
	if strings.HasPrefix(selector, "#") {
		return el.ID() == selector[1:]
	}
	if strings.HasPrefix(selector, ".") {
		return el.Class() == selector[1:]
	}
	return el.TagName() == selector
}
