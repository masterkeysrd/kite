package kitex

import (
	"slices"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/promise"
)

var (
	renderMutex        sync.Mutex
	activeRoots        = make(map[dom.Element]Node)
	inOnComponentDirty bool
	dirtyComponents    []componentInstance
	dirtyBuffer        []componentInstance
)

func init() {
	dirtyComponents = make([]componentInstance, 0, 16)
	dirtyBuffer = make([]componentInstance, 0, 16)

	OnComponentDirty = func(node Node) {
		compInstance, ok := node.(componentInstance)
		if !ok {
			return
		}
		if compInstance.realNode() == nil {
			return
		}

		effectsMutex.Lock()
		exists := slices.Contains(dirtyComponents, compInstance)
		if !exists {
			dirtyComponents = append(dirtyComponents, compInstance)
		}
		effectsMutex.Unlock()

		if inOnComponentDirty {
			return
		}

		renderMutex.Lock()
		inOnComponentDirty = true
		defer func() {
			inOnComponentDirty = false
			renderMutex.Unlock()
		}()

		flushPendingEffects()
		processDirtyLoop()
	}
}

func processDirtyLoop() {
	for range 10 {
		if len(dirtyComponents) == 0 {
			break
		}
		effectsMutex.Lock()
		if len(dirtyComponents) == 0 {
			effectsMutex.Unlock()
			break
		}
		currentDirty := dirtyComponents
		if cap(dirtyBuffer) < len(currentDirty) {
			dirtyBuffer = make([]componentInstance, 0, len(currentDirty))
		}
		dirtyComponents = dirtyBuffer
		dirtyBuffer = currentDirty[:0]
		effectsMutex.Unlock()

		for _, comp := range currentDirty {
			if !comp.IsDirty() {
				continue
			}
			ref := comp.getRef()
			if ref == nil || ref.node != comp {
				continue
			}
			realNode := comp.realNode()
			if realNode == nil {
				continue
			}
			parent := realNode.Parent()
			if parent == nil {
				continue
			}
			parentEl, ok := parent.(dom.Element)
			if !ok {
				continue
			}

			oldRendered := comp.Rendered()
			newRendered := comp.ReRender()
			comp.ClearDirty()

			newReal := reconcile(parentEl, oldRendered, newRendered, realNode)
			if newReal != realNode {
				comp.setRef(newReal)
			}
		}

		drainLayoutEffects()
	}
}

// Render mounts or reconciles a Virtual DOM root node into the specified host container.
func Render(root Node, container dom.Element) {
	renderMutex.Lock()
	inOnComponentDirty = true
	defer func() {
		inOnComponentDirty = false
		renderMutex.Unlock()
	}()

	if container == nil {
		return
	}

	if doc := container.OwnerDocument(); doc != nil {
		if term := doc.Terminal(); term != nil {
			sched := term.Scheduler()
			setInternalScheduler(sched)
			promise.SetScheduler(sched)
		}
	}

	oldRoot := activeRoots[container]
	if root == nil {
		delete(activeRoots, container)
		if oldRoot != nil {
			firstChild := container.FirstChild()
			if firstChild != nil {
				destroyNode(oldRoot)
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

	drainLayoutEffects()
	processDirtyLoop()
}

func reconcile(parent dom.Element, oldNode, newNode Node, realNode dom.Node) dom.Node {
	if oldNode == nil && newNode == nil {
		return nil
	}

	if oldNode == newNode {
		return realNode
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
			destroyNode(oldNode)
			parent.RemoveChild(realNode)
			ClearAllSubscriptions(realNode)
		}
		return nil
	}

	// 3. Replace on tag mismatch
	if oldNode.TagName() != newNode.TagName() {
		newReal := newNode.Instantiate(parent.OwnerDocument())
		if realNode != nil {
			destroyNode(oldNode)
			parent.ReplaceChild(newReal, realNode)
			ClearAllSubscriptions(realNode)
		} else {
			parent.AppendChild(newReal)
		}
		return newReal
	}

	// 4. Update in place
	// Provider Node:
	if oldProv, ok := oldNode.(providerInstance); ok {
		newProv := newNode.(providerInstance)

		newProv.pushEntry()
		newProv.updateFrom(oldProv)

		newReal := realNode
		if len(oldProv.Children()) == 1 && len(newProv.Children()) == 1 {
			newReal = reconcile(parent, oldProv.Children()[0], newProv.Children()[0], realNode)
		}
		newProv.popEntry()
		return newReal
	}

	// Component Node:
	if oldComp, ok := oldNode.(componentInstance); ok {
		newComp := newNode.(componentInstance)

		newNode.Update(realNode, oldNode)
		newComp.ClearDirty()

		newReal := reconcile(parent, oldComp.Rendered(), newComp.Rendered(), realNode)
		if newReal != realNode {
			newComp.setRef(newReal)
		}
		return newReal
	}

	// Text Node:
	if oldNode.TagName() == "#text" {
		newNode.Update(realNode, oldNode)
		return realNode
	}

	// Element Node:
	newNode.Update(realNode, oldNode)
	reconcileChildren(realNode.(dom.Element), oldNode, newNode, oldNode.Children(), newNode.Children())
	return realNode
}

type providerInstance interface {
	nodeInternal
	pushEntry()
	popEntry()
	initEntry()
	updateFrom(old providerInstance)
}

type flatNode struct {
	node      Node
	providers []providerInstance
}

func countFlatNodes(nodes []Node) int {
	count := 0
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if prov, ok := n.(providerInstance); ok {
			count += countFlatNodes(prov.Children())
		} else {
			count++
		}
	}
	return count
}

func flattenNodes(nodes []Node, activeProviders []providerInstance, out []flatNode) []flatNode {
	return flattenNodesRec(nodes, activeProviders, out)
}

func flattenNodesRec(nodes []Node, activeProviders []providerInstance, out []flatNode) []flatNode {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if prov, ok := n.(providerInstance); ok {
			out = flattenNodesRec(prov.Children(), append(activeProviders, prov), out)
		} else {
			out = append(out, flatNode{node: n, providers: activeProviders})
		}
	}
	return out
}

func reconcileFlat(parent dom.Element, oldFlat, newFlat flatNode, realNode dom.Node) {
	for i := 0; i < len(newFlat.providers); i++ {
		newProv := newFlat.providers[i]
		if i < len(oldFlat.providers) {
			oldProv := oldFlat.providers[i]
			newProv.updateFrom(oldProv)
		}
	}

	for _, prov := range newFlat.providers {
		prov.pushEntry()
	}

	reconcile(parent, oldFlat.node, newFlat.node, realNode)

	for i := len(newFlat.providers) - 1; i >= 0; i-- {
		newFlat.providers[i].popEntry()
	}
}

func instantiateFlat(parent dom.Element, newFlat flatNode) dom.Node {
	for _, prov := range newFlat.providers {
		prov.initEntry()
		prov.pushEntry()
	}
	defer func() {
		for i := len(newFlat.providers) - 1; i >= 0; i-- {
			newFlat.providers[i].popEntry()
		}
	}()
	return newFlat.node.Instantiate(parent.OwnerDocument())
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

var flatNodeSlicePool = sync.Pool{
	New: func() any {
		s := make([]flatNode, 0, 128)
		return &s
	},
}

func reconcileChildren(parent dom.Element, oldParent, newParent Node, oldChildren, newChildren []Node) {
	hasP := false
	if op, ok := oldParent.(nodeInternal); ok && op.hasDirectProvider() {
		hasP = true
	} else if np, ok := newParent.(nodeInternal); ok && np.hasDirectProvider() {
		hasP = true
	}

	if hasP {
		reconcileChildrenFlat(parent, oldChildren, newChildren)
		return
	}

	if len(oldChildren) == 0 {
		for i := range newChildren {
			if newChildren[i] != nil {
				newReal := newChildren[i].Instantiate(parent.OwnerDocument())
				if newReal != nil {
					parent.AppendChild(newReal)
				}
			}
		}
		return
	}

	if len(newChildren) == 0 {
		for i := range oldChildren {
			if oldChildren[i] != nil {
				realNode := oldChildren[i].(nodeInternal).realNode()
				if realNode != nil {
					destroyNode(oldChildren[i])
					parent.RemoveChild(realNode)
					ClearAllSubscriptions(realNode)
				}
			}
		}
		return
	}

	if len(oldChildren) == 1 && len(newChildren) == 1 {
		oldChild := oldChildren[0]
		newChild := newChildren[0]
		if sameNode(oldChild, newChild) {
			var realNode dom.Node
			if oldChild != nil {
				realNode = oldChild.(nodeInternal).realNode()
			}
			reconcile(parent, oldChild, newChild, realNode)
			return
		}
	}

	n := len(oldChildren)
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

	var insertBeforeRef dom.Node
	var nodeMap map[Node]dom.Node

	for oldStartIdx <= oldEndIdx && newStartIdx <= newEndIdx {
		for oldStartIdx <= oldEndIdx && oldS[oldStartIdx] == nil {
			oldStartIdx++
		}
		for oldEndIdx >= oldStartIdx && oldS[oldEndIdx] == nil {
			oldEndIdx--
		}
		for newStartIdx <= newEndIdx && newChildren[newStartIdx] == nil {
			newStartIdx++
		}
		for newEndIdx >= newStartIdx && newChildren[newEndIdx] == nil {
			newEndIdx--
		}
		if oldStartIdx > oldEndIdx || newStartIdx > newEndIdx {
			break
		}

		oldStartNode := oldS[oldStartIdx]
		newStartNode := newChildren[newStartIdx]
		oldEndNode := oldS[oldEndIdx]
		newEndNode := newChildren[newEndIdx]

		// Case 1: Old Start == New Start (no move needed)
		if sameNode(oldStartNode, newStartNode) {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldStartNode]
			} else {
				realNode = oldChildren[oldStartIdx].(nodeInternal).realNode()
			}
			reconcile(parent, oldStartNode, newStartNode, realNode)
			oldStartIdx++
			newStartIdx++
			continue
		}

		// Case 2: Old End == New End (no move needed)
		if sameNode(oldEndNode, newEndNode) {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldEndNode]
			} else {
				realNode = oldChildren[oldEndIdx].(nodeInternal).realNode()
			}
			reconcile(parent, oldEndNode, newEndNode, realNode)
			insertBeforeRef = realNode
			oldEndIdx--
			newEndIdx--
			continue
		}

		// Case 3-5: Move needed. Ensure map is populated.
		if nodeMap == nil {
			nodeMap = nodeRealMapPool.Get().(map[Node]dom.Node)
			for i := range n {
				if oldChildren[i] != nil {
					nodeMap[oldChildren[i]] = oldChildren[i].(nodeInternal).realNode()
				}
			}
		}

		// Case 3: Old Start goes to New End -> move it after current oldEnd
		if sameNode(oldStartNode, newEndNode) {
			domNode := nodeMap[oldStartNode]
			reconcile(parent, oldStartNode, newEndNode, domNode)
			afterEnd := nodeMap[oldEndNode]
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

		// Case 4: Old End goes to New Start -> move it before current oldStart
		if sameNode(oldEndNode, newStartNode) {
			domNode := nodeMap[oldEndNode]
			reconcile(parent, oldEndNode, newStartNode, domNode)
			parent.InsertBefore(domNode, nodeMap[oldStartNode])
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
			parent.InsertBefore(matchedReal, nodeMap[oldStartNode])
			oldS[matchedIdx] = nil
		} else {
			newReal := newStartNode.Instantiate(parent.OwnerDocument())
			parent.InsertBefore(newReal, nodeMap[oldStartNode])
		}
		newStartIdx++
	}

	// Insert remaining new nodes at the end
	if newStartIdx <= newEndIdx {
		var refNode dom.Node
		for i := oldStartIdx; i <= oldEndIdx; i++ {
			if oldS[i] != nil {
				if nodeMap != nil {
					refNode = nodeMap[oldS[i]]
				} else {
					refNode = oldChildren[i].(nodeInternal).realNode()
				}
				break
			}
		}
		if refNode == nil {
			refNode = insertBeforeRef
		}
		for newStartIdx <= newEndIdx {
			if newChildren[newStartIdx] != nil {
				newReal := newChildren[newStartIdx].Instantiate(parent.OwnerDocument())
				parent.InsertBefore(newReal, refNode)
			}
			newStartIdx++
		}
	}

	// Remove remaining unmatched old nodes
	for i := oldStartIdx; i <= oldEndIdx; i++ {
		if oldS[i] != nil {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldS[i]]
			} else {
				realNode = oldChildren[i].(nodeInternal).realNode()
			}
			if realNode != nil {
				destroyNode(oldS[i])
				parent.RemoveChild(realNode)
				ClearAllSubscriptions(realNode)
			}
		}
	}

	if pooledOldS != nil {
		for i := range *pooledOldS {
			(*pooledOldS)[i] = nil
		}
		nodeSlicePool.Put(pooledOldS)
	}
	if nodeMap != nil {
		clear(nodeMap)
		nodeRealMapPool.Put(nodeMap)
	}
}

func reconcileChildrenFlat(parent dom.Element, oldChildren, newChildren []Node) {
	capacityOld := countFlatNodes(oldChildren)
	capacityNew := countFlatNodes(newChildren)

	var flatOldPtr, flatNewPtr, oldSPtr *[]flatNode
	var flatOld, flatNew, oldS []flatNode

	if capacityOld <= 128 {
		flatOldPtr = flatNodeSlicePool.Get().(*[]flatNode)
		flatOld = flattenNodes(oldChildren, nil, (*flatOldPtr)[:0])
	} else {
		flatOld = flattenNodes(oldChildren, nil, make([]flatNode, 0, capacityOld))
	}

	if capacityNew <= 128 {
		flatNewPtr = flatNodeSlicePool.Get().(*[]flatNode)
		flatNew = flattenNodes(newChildren, nil, (*flatNewPtr)[:0])
	} else {
		flatNew = flattenNodes(newChildren, nil, make([]flatNode, 0, capacityNew))
	}

	defer func() {
		if flatOldPtr != nil {
			*flatOldPtr = flatOld[:0]
			flatNodeSlicePool.Put(flatOldPtr)
		}
		if flatNewPtr != nil {
			*flatNewPtr = flatNew[:0]
			flatNodeSlicePool.Put(flatNewPtr)
		}
		if oldSPtr != nil {
			for i := range *oldSPtr {
				(*oldSPtr)[i] = flatNode{}
			}
			flatNodeSlicePool.Put(oldSPtr)
		}
	}()

	if len(flatOld) == 0 {
		for i := range flatNew {
			newReal := instantiateFlat(parent, flatNew[i])
			if newReal != nil {
				parent.AppendChild(newReal)
			}
		}
		return
	}

	if len(flatNew) == 0 {
		for i := range flatOld {
			realNode := flatOld[i].node.(nodeInternal).realNode()
			if realNode != nil {
				destroyNode(flatOld[i].node)
				parent.RemoveChild(realNode)
				ClearAllSubscriptions(realNode)
			}
		}
		return
	}

	if len(flatOld) == 1 && len(flatNew) == 1 {
		oldChild := flatOld[0]
		newChild := flatNew[0]
		if sameNode(oldChild.node, newChild.node) {
			var realNode = oldChild.node.(nodeInternal).realNode()
			reconcileFlat(parent, oldChild, newChild, realNode)
			return
		}
	}

	n := len(flatOld)
	if n <= 128 {
		oldSPtr = flatNodeSlicePool.Get().(*[]flatNode)
		oldS = (*oldSPtr)[:n]
	} else {
		oldS = make([]flatNode, n)
	}
	copy(oldS, flatOld)

	oldStartIdx := 0
	newStartIdx := 0
	oldEndIdx := n - 1
	newEndIdx := len(flatNew) - 1

	var insertBeforeRef dom.Node
	var nodeMap map[Node]dom.Node

	for oldStartIdx <= oldEndIdx && newStartIdx <= newEndIdx {
		for oldStartIdx <= oldEndIdx && oldS[oldStartIdx].node == nil {
			oldStartIdx++
		}
		for oldEndIdx >= oldStartIdx && oldS[oldEndIdx].node == nil {
			oldEndIdx--
		}
		if oldStartIdx > oldEndIdx || newStartIdx > newEndIdx {
			break
		}

		oldStartNode := oldS[oldStartIdx]
		newStartNode := flatNew[newStartIdx]
		oldEndNode := oldS[oldEndIdx]
		newEndNode := flatNew[newEndIdx]

		// Case 1: Old Start == New Start (no move needed)
		if sameNode(oldStartNode.node, newStartNode.node) {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldStartNode.node]
			} else {
				realNode = flatOld[oldStartIdx].node.(nodeInternal).realNode()
			}
			reconcileFlat(parent, oldStartNode, newStartNode, realNode)
			oldStartIdx++
			newStartIdx++
			continue
		}

		// Case 2: Old End == New End (no move needed)
		if sameNode(oldEndNode.node, newEndNode.node) {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldEndNode.node]
			} else {
				realNode = flatOld[oldEndIdx].node.(nodeInternal).realNode()
			}
			reconcileFlat(parent, oldEndNode, newEndNode, realNode)
			insertBeforeRef = realNode
			oldEndIdx--
			newEndIdx--
			continue
		}

		// Case 3-5: Move needed. Ensure map is populated.
		if nodeMap == nil {
			nodeMap = nodeRealMapPool.Get().(map[Node]dom.Node)
			for i := range n {
				nodeMap[flatOld[i].node] = flatOld[i].node.(nodeInternal).realNode()
			}
		}

		// Case 3: Old Start goes to New End -> move it after current oldEnd
		if sameNode(oldStartNode.node, newEndNode.node) {
			domNode := nodeMap[oldStartNode.node]
			reconcileFlat(parent, oldStartNode, newEndNode, domNode)
			afterEnd := nodeMap[oldEndNode.node]
			if afterEnd != nil {
				parent.InsertBefore(domNode, afterEnd.NextSibling())
			} else {
				parent.InsertBefore(domNode, nil)
			}
			oldS[oldStartIdx] = flatNode{}
			oldStartIdx++
			newEndIdx--
			continue
		}

		// Case 4: Old End goes to New Start -> move it before current oldStart
		if sameNode(oldEndNode.node, newStartNode.node) {
			domNode := nodeMap[oldEndNode.node]
			reconcileFlat(parent, oldEndNode, newStartNode, domNode)
			parent.InsertBefore(domNode, nodeMap[oldStartNode.node])
			oldS[oldEndIdx] = flatNode{}
			oldEndIdx--
			newStartIdx++
			continue
		}

		// Case 5: Complex lookup by key
		oldKeyMap := keyMapPool.Get().(map[string]int)
		for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
			if oldS[idx].node != nil {
				if key := oldS[idx].node.Key(); key != "" {
					oldKeyMap[key] = idx
				}
			}
		}

		newKey := newStartNode.node.Key()
		matchedIdx := -1
		if newKey != "" {
			if idx, found := oldKeyMap[newKey]; found {
				matchedIdx = idx
			}
		} else {
			for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
				if oldS[idx].node != nil && oldS[idx].node.Key() == "" && oldS[idx].node.TagName() == newStartNode.node.TagName() {
					matchedIdx = idx
					break
				}
			}
		}
		clear(oldKeyMap)
		keyMapPool.Put(oldKeyMap)

		if matchedIdx != -1 {
			matchedReal := nodeMap[oldS[matchedIdx].node]
			reconcileFlat(parent, oldS[matchedIdx], newStartNode, matchedReal)
			parent.InsertBefore(matchedReal, nodeMap[oldStartNode.node])
			oldS[matchedIdx] = flatNode{}
		} else {
			newReal := instantiateFlat(parent, newStartNode)
			parent.InsertBefore(newReal, nodeMap[oldStartNode.node])
		}
		newStartIdx++
	}

	// Insert remaining new nodes at the end
	if newStartIdx <= newEndIdx {
		var refNode dom.Node
		for i := oldStartIdx; i <= oldEndIdx; i++ {
			if oldS[i].node != nil {
				if nodeMap != nil {
					refNode = nodeMap[oldS[i].node]
				} else {
					refNode = flatOld[i].node.(nodeInternal).realNode()
				}
				break
			}
		}
		if refNode == nil {
			refNode = insertBeforeRef
		}
		for newStartIdx <= newEndIdx {
			newReal := instantiateFlat(parent, flatNew[newStartIdx])
			parent.InsertBefore(newReal, refNode)
			newStartIdx++
		}
	}

	// Remove remaining unmatched old nodes
	for i := oldStartIdx; i <= oldEndIdx; i++ {
		if oldS[i].node != nil {
			var realNode dom.Node
			if nodeMap != nil {
				realNode = nodeMap[oldS[i].node]
			} else {
				realNode = flatOld[i].node.(nodeInternal).realNode()
			}
			if realNode != nil {
				destroyNode(oldS[i].node)
				parent.RemoveChild(realNode)
				ClearAllSubscriptions(realNode)
			}
		}
	}

	if nodeMap != nil {
		clear(nodeMap)
		nodeRealMapPool.Put(nodeMap)
	}
}

func sameNode(n1, n2 Node) bool {
	if n1 == nil || n2 == nil {
		return false
	}
	return n1.TagName() == n2.TagName() && n1.Key() == n2.Key()
}
