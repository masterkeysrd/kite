// Package marker provides a way to mark nodes so we can identify them without importing the render package. This is used by the layout engine to identify nodes that are
// participating in layout without creating an import cycle.
package marker

type Node interface {
	isKiteNode()
}

type NodeMarker struct{}

func (NodeMarker) isKiteNode() {}
