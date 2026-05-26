package kitex

import (
	"sync"

	"github.com/masterkeysrd/kite/dom"
)

var (
	renderMutex sync.Mutex
	activeRoots = make(map[dom.Element]Node)
)

func init() {
	OnComponentDirty = func(node Node) {
		renderMutex.Lock()
		defer renderMutex.Unlock()

		compInstance, ok := node.(componentInstance)
		if !ok {
			return
		}
		realNode := compInstance.realNode()
		if realNode == nil {
			return
		}
		parent := realNode.Parent()
		if parent == nil {
			return
		}
		parentEl, ok := parent.(dom.Element)
		if !ok {
			return
		}

		oldRendered := compInstance.Rendered()
		newRendered := compInstance.ReRender()
		compInstance.ClearDirty()

		reconcile(parentEl, oldRendered, newRendered, realNode)
	}
}

// Render mounts or reconciles a Virtual DOM root node into the specified host container.
func Render(root Node, container dom.Element) {
	renderMutex.Lock()
	defer renderMutex.Unlock()

	if container == nil {
		return
	}

	oldRoot := activeRoots[container]
	if root == nil {
		delete(activeRoots, container)
		if oldRoot != nil {
			firstChild := container.FirstChild()
			if firstChild != nil {
				container.RemoveChild(firstChild)
				ClearAllSubscriptions(firstChild)
			}
		}
		return
	}

	activeRoots[container] = root
	if oldRoot != nil {
		reconcile(container, oldRoot, root, container.FirstChild())
	} else {
		realNode := root.Instantiate(container.OwnerDocument())
		if realNode != nil {
			container.AppendChild(realNode)
		}
	}
}

func reconcile(parent dom.Element, oldNode, newNode Node, realNode dom.Node) dom.Node {
	if oldNode == nil && newNode == nil {
		return nil
	}

	// 1. Mount
	if oldNode == nil && newNode != nil {
		newReal := newNode.Instantiate(parent.OwnerDocument())
		if newReal != nil {
			parent.AppendChild(newReal)
		}
		return newReal
	}

	// 2. Unmount
	if oldNode != nil && newNode == nil {
		if realNode != nil {
			parent.RemoveChild(realNode)
			ClearAllSubscriptions(realNode)
		}
		return nil
	}

	// 3. Replace on tag mismatch
	if oldNode.TagName() != newNode.TagName() {
		newReal := newNode.Instantiate(parent.OwnerDocument())
		if realNode != nil {
			parent.ReplaceChild(newReal, realNode)
			ClearAllSubscriptions(realNode)
		} else {
			parent.AppendChild(newReal)
		}
		return newReal
	}

	// 4. Update in place
	// Component Node:
	if oldComp, ok := oldNode.(componentInstance); ok {
		newComp := newNode.(componentInstance)

		newNode.Update(realNode, oldNode)
		newComp.ClearDirty()

		reconcile(parent, oldComp.Rendered(), newComp.Rendered(), realNode)
		return realNode
	}

	// Text Node:
	if oldNode.TagName() == "#text" {
		newNode.Update(realNode, oldNode)
		return realNode
	}

	// Element Node:
	newNode.Update(realNode, oldNode)
	reconcileChildren(realNode.(dom.Element), oldNode.Children(), newNode.Children())
	return realNode
}

var keyMapPool = sync.Pool{
	New: func() any {
		return make(map[string]int)
	},
}

var nodeSlicePool = sync.Pool{
	New: func() any {
		s := make([]Node, 128)
		return &s
	},
}

var nodeRealMapPool = sync.Pool{
	New: func() any {
		return make(map[Node]dom.Node, 32)
	},
}

func reconcileChildren(parent dom.Element, oldChildren, newChildren []Node) {
	// Build a stable map from VDOM node pointer → its current live DOM node.
	// This is immune to DOM-move invalidation: when InsertBefore moves a node,
	// the mapping (vdom → dom) remains valid regardless of sibling-list changes.
	n := len(oldChildren)
	nodeMap := nodeRealMapPool.Get().(map[Node]dom.Node)

	for i := range n {
		nodeMap[oldChildren[i]] = oldChildren[i].(nodeInternal).realNode()
	}

	// Working copy of the old VDOM list so we can nil-out consumed entries.
	var oldS []Node
	var pooledOldS *[]Node
	if n <= 128 {
		pooledOldS = nodeSlicePool.Get().(*[]Node)
		oldS = (*pooledOldS)[:n]
	} else {
		oldS = make([]Node, n)
	}
	copy(oldS, oldChildren)

	oldStartIdx := 0
	newStartIdx := 0
	oldEndIdx := n - 1
	newEndIdx := len(newChildren) - 1

	// insertBeforeRef tracks the DOM node that new trailing nodes should be
	// inserted before. Updated when Case 2 consumes an old-end match.
	var insertBeforeRef dom.Node

	for oldStartIdx <= oldEndIdx && newStartIdx <= newEndIdx {
		for oldStartIdx <= oldEndIdx && oldS[oldStartIdx] == nil {
			oldStartIdx++
		}
		for oldEndIdx >= oldStartIdx && oldS[oldEndIdx] == nil {
			oldEndIdx--
		}
		if oldStartIdx > oldEndIdx {
			break
		}

		oldStartNode := oldS[oldStartIdx]
		newStartNode := newChildren[newStartIdx]
		oldEndNode := oldS[oldEndIdx]
		newEndNode := newChildren[newEndIdx]

		// Case 1: Old Start == New Start (no move needed)
		if sameNode(oldStartNode, newStartNode) {
			reconcile(parent, oldStartNode, newStartNode, getRealNode(oldStartIdx, oldS, nodeMap))
			oldStartIdx++
			newStartIdx++
			continue
		}

		// Case 2: Old End == New End (no move needed)
		if sameNode(oldEndNode, newEndNode) {
			domNode := getRealNode(oldEndIdx, oldS, nodeMap)
			reconcile(parent, oldEndNode, newEndNode, domNode)
			// Any new nodes inserted between start and end must go before this.
			insertBeforeRef = domNode
			oldEndIdx--
			newEndIdx--
			continue
		}

		// Case 3: Old Start goes to New End → move it after current oldEnd
		if sameNode(oldStartNode, newEndNode) {
			domNode := getRealNode(oldStartIdx, oldS, nodeMap)
			reconcile(parent, oldStartNode, newEndNode, domNode)
			afterEnd := getRealNode(oldEndIdx, oldS, nodeMap)
			if afterEnd != nil {
				parent.InsertBefore(domNode, afterEnd.NextSibling())
			} else {
				parent.InsertBefore(domNode, nil)
			}
			oldS[oldStartIdx] = nil
			oldStartIdx++
			newEndIdx--
			continue
		}

		// Case 4: Old End goes to New Start → move it before current oldStart
		if sameNode(oldEndNode, newStartNode) {
			domNode := getRealNode(oldEndIdx, oldS, nodeMap)
			reconcile(parent, oldEndNode, newStartNode, domNode)
			parent.InsertBefore(domNode, getRealNode(oldStartIdx, oldS, nodeMap))
			oldS[oldEndIdx] = nil
			oldEndIdx--
			newStartIdx++
			continue
		}

		// Case 5: Complex lookup by key
		oldKeyMap := keyMapPool.Get().(map[string]int)
		for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
			if oldS[idx] != nil {
				if key := oldS[idx].Key(); key != "" {
					oldKeyMap[key] = idx
				}
			}
		}

		newKey := newStartNode.Key()
		matchedIdx := -1
		if newKey != "" {
			if idx, found := oldKeyMap[newKey]; found {
				matchedIdx = idx
			}
		} else {
			for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
				if oldS[idx] != nil && oldS[idx].Key() == "" && oldS[idx].TagName() == newStartNode.TagName() {
					matchedIdx = idx
					break
				}
			}
		}
		clear(oldKeyMap)
		keyMapPool.Put(oldKeyMap)

		if matchedIdx != -1 {
			matchedReal := nodeMap[oldS[matchedIdx]]
			reconcile(parent, oldS[matchedIdx], newStartNode, matchedReal)
			parent.InsertBefore(matchedReal, getRealNode(oldStartIdx, oldS, nodeMap))
			oldS[matchedIdx] = nil
		} else {
			newReal := newStartNode.Instantiate(parent.OwnerDocument())
			parent.InsertBefore(newReal, getRealNode(oldStartIdx, oldS, nodeMap))
		}
		newStartIdx++
	}

	// Insert remaining new nodes at the end
	if newStartIdx <= newEndIdx {
		var refNode dom.Node
		// First try: any surviving old node in range is the natural anchor.
		for i := oldStartIdx; i <= oldEndIdx; i++ {
			if oldS[i] != nil {
				refNode = nodeMap[oldS[i]]
				break
			}
		}
		// Fallback: use the Case-2 anchor saved above (e.g. [A,B]→[A,C,B]).
		if refNode == nil {
			refNode = insertBeforeRef
		}
		for newStartIdx <= newEndIdx {
			newReal := newChildren[newStartIdx].Instantiate(parent.OwnerDocument())
			parent.InsertBefore(newReal, refNode)
			newStartIdx++
		}
	}

	// Remove remaining unmatched old nodes
	for i := oldStartIdx; i <= oldEndIdx; i++ {
		if oldS[i] != nil {
			realNode := nodeMap[oldS[i]]
			if realNode != nil {
				parent.RemoveChild(realNode)
				ClearAllSubscriptions(realNode)
			}
		}
	}

	// Return slices and map to pools
	if pooledOldS != nil {
		for i := range *pooledOldS {
			(*pooledOldS)[i] = nil
		}
		nodeSlicePool.Put(pooledOldS)
	}
	clear(nodeMap)
	nodeRealMapPool.Put(nodeMap)
}

func sameNode(n1, n2 Node) bool {
	if n1 == nil || n2 == nil {
		return false
	}
	return n1.TagName() == n2.TagName() && n1.Key() == n2.Key()
}

func getRealNode(idx int, oldS []Node, nodeMap map[Node]dom.Node) dom.Node {
	if idx < 0 || idx >= len(oldS) || oldS[idx] == nil {
		return nil
	}
	return nodeMap[oldS[idx]]
}
