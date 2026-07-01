package kitex

import (
	"slices"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	kitelog "github.com/masterkeysrd/kite/log"
	"github.com/masterkeysrd/kite/promise"
)

var (
	renderMutex        sync.Mutex
	activeRoots        = make(map[dom.Element]Node)
	inOnComponentDirty bool
	dirtyComponents    []componentInstance
	dirtyBuffer        []componentInstance
	updateScheduled    bool
	updateScheduledMu  sync.Mutex
)

func scheduleDirtyFlush(comp componentInstance) {
	parent := comp.getDOMParent()
	if parent != nil && parent.OwnerDocument() != nil && parent.OwnerDocument().Terminal() != nil && scheduler != nil {
		scheduler.QueueMacrotask(flushDirtyComponents)
	} else {
		flushDirtyComponents()
	}
}

func flushDirtyComponents() {
	updateScheduledMu.Lock()
	updateScheduled = false
	updateScheduledMu.Unlock()

	renderMutex.Lock()
	if inOnComponentDirty {
		renderMutex.Unlock()
		return
	}
	inOnComponentDirty = true
	defer func() {
		inOnComponentDirty = false
		renderMutex.Unlock()
	}()

	flushPendingEffects()
	processDirtyLoop()
}

func init() {
	dirtyComponents = make([]componentInstance, 0, 16)
	dirtyBuffer = make([]componentInstance, 0, 16)

	OnComponentDirty = func(node Node) {
		compInstance, ok := node.(componentInstance)
		if !ok {
			return
		}
		if compInstance.getDOMParent() == nil {
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

		updateScheduledMu.Lock()
		if updateScheduled {
			updateScheduledMu.Unlock()
			return
		}
		updateScheduled = true
		updateScheduledMu.Unlock()

		scheduleDirtyFlush(compInstance)
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
		if cap(dirtyBuffer) < cap(currentDirty) {
			dirtyBuffer = make([]componentInstance, 0, cap(currentDirty))
		}
		dirtyComponents = dirtyBuffer[:0]
		dirtyBuffer = currentDirty
		effectsMutex.Unlock()

		for _, comp := range currentDirty {
			if !comp.IsDirty() {
				continue
			}
			ref := comp.getRef()
			if ref == nil {
				continue
			}
			if ref.node != comp {
				continue
			}
			parentEl := comp.getDOMParent()
			if parentEl == nil {
				continue
			}
			realNodes := comp.realNodes()

			oldRendered := comp.Rendered()
			restoreCleanup := comp.restoreContexts()
			newRendered := comp.ReRender()
			comp.ClearDirty()

			newReals := reconcile(parentEl, oldRendered, newRendered, realNodes, comp.getDOMAnchor())
			restoreCleanup()
			comp.setRefs(newReals)
		}
		for i := range currentDirty {
			currentDirty[i] = nil
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
			realNodes := oldRoot.(nodeInternal).realNodes()
			destroyNode(oldRoot)
			for _, realNode := range realNodes {
				if realNode != nil {
					if isParent(realNode, container) {
						container.RemoveChild(realNode)
					}
					ClearAllSubscriptions(realNode)
				}
			}
		}
		return
	}

	activeRoots[container] = root
	setDOMParent(root, container)
	if oldRoot != nil {
		reconcile(container, oldRoot, root, oldRoot.(nodeInternal).realNodes(), nil)
	} else {
		realNodes := root.Instantiate(container.OwnerDocument())
		for _, realNode := range realNodes {
			if realNode != nil {
				container.AppendChild(realNode)
			}
		}
	}

	drainLayoutEffects()
	processDirtyLoop()
}

func reconcile(parent dom.Element, oldNode, newNode Node, realNodes []dom.Node, anchor dom.Node) []dom.Node {
	if oldNode == nil && newNode == nil {
		return nil
	}

	setDOMParent(newNode, parent)

	if oldNode == newNode {
		if comp, ok := oldNode.(componentInstance); !ok || !comp.IsDirty() {
			return realNodes
		}
	}

	// 1. Mount
	if oldNode == nil && newNode != nil {
		newReals := newNode.Instantiate(parent.OwnerDocument())
		for _, newReal := range newReals {
			safeInsertBefore(parent, newReal, anchor)
		}
		return newReals
	}

	// 2. Unmount
	if oldNode != nil && newNode == nil {
		destroyNode(oldNode)
		for _, realNode := range realNodes {
			if realNode != nil {
				if isParent(realNode, parent) {
					parent.RemoveChild(realNode)
				}
				ClearAllSubscriptions(realNode)
			}
		}
		return nil
	}

	// 3. Replace on tag mismatch
	if oldNode.TagName() != newNode.TagName() {
		if EnableDevMode && oldNode.Key() == "" && newNode.Key() == "" {
			kitelog.Warn("Keyless type mismatch replacement detected. This can cause visual ordering swap or cell position leaks in dynamic lists. Consider adding unique keys.",
				"parent", parent.TagName(),
				"oldTag", oldNode.TagName(),
				"newTag", newNode.TagName(),
			)
		}
		destroyNode(oldNode)
		if len(realNodes) > 0 {
			nextSibling := realNodes[len(realNodes)-1].NextSibling()
			for _, oldReal := range realNodes {
				if oldReal != nil {
					if isParent(oldReal, parent) {
						parent.RemoveChild(oldReal)
					}
					ClearAllSubscriptions(oldReal)
				}
			}
			newReals := newNode.Instantiate(parent.OwnerDocument())
			for _, newReal := range newReals {
				safeInsertBefore(parent, newReal, nextSibling)
			}
			return newReals
		} else {
			newReals := newNode.Instantiate(parent.OwnerDocument())
			for _, newReal := range newReals {
				safeInsertBefore(parent, newReal, anchor)
			}
			return newReals
		}
	}

	// 4. Update in place
	// Provider Node:
	if oldProv, ok := oldNode.(providerInstance); ok {
		newProv := newNode.(providerInstance)

		newProv.updateFrom(oldProv)
		newProv.pushEntry()

		reconcileChildren(parent, oldProv, newProv, oldProv.Children(), newProv.Children(), anchor)
		newProv.popEntry()

		count := 0
		for _, child := range newProv.Children() {
			if child != nil {
				count += len(child.(nodeInternal).realNodes())
			}
		}
		var reals []dom.Node
		if count > 0 {
			reals = make([]dom.Node, 0, count)
			for _, child := range newProv.Children() {
				if child != nil {
					reals = append(reals, child.(nodeInternal).realNodes()...)
				}
			}
		}
		newProv.setRefs(reals)
		return reals
	}

	// Fragment Node:
	if oldNode.TagName() == "#fragment" {
		reconcileChildren(parent, oldNode, newNode, oldNode.Children(), newNode.Children(), anchor)
		count := 0
		for _, child := range newNode.Children() {
			if child != nil {
				count += len(child.(nodeInternal).realNodes())
			}
		}
		var reals []dom.Node
		if count > 0 {
			reals = make([]dom.Node, 0, count)
			for _, child := range newNode.Children() {
				if child != nil {
					reals = append(reals, child.(nodeInternal).realNodes()...)
				}
			}
		}
		newNode.(nodeInternal).setRefs(reals)
		return reals
	}

	// Component Node:
	if oldComp, ok := oldNode.(componentInstance); ok {
		newComp := newNode.(componentInstance)

		oldRendered := oldComp.Rendered()
		newNode.Update(realNodes, oldNode)
		newComp.ClearDirty()
		newComp.setDOMAnchor(anchor)

		newReals := reconcile(parent, oldRendered, newComp.Rendered(), realNodes, anchor)
		newComp.setRefs(newReals)
		return newReals
	}

	// Text Node:
	if oldNode.TagName() == "#text" {
		newNode.Update(realNodes, oldNode)
		return realNodes
	}

	// Element Node:
	newNode.Update(realNodes, oldNode)
	if len(realNodes) > 0 && realNodes[0] != nil {
		el := realNodes[0].(dom.Element)
		for _, child := range newNode.Children() {
			setDOMParent(child, el)
		}
		reconcileChildren(el, oldNode, newNode, oldNode.Children(), newNode.Children(), nil)
	}
	return realNodes
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
		} else if n.TagName() == "#fragment" {
			count += countFlatNodes(n.Children())
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
		} else if n.TagName() == "#fragment" {
			out = flattenNodesRec(n.Children(), activeProviders, out)
		} else {
			out = append(out, flatNode{node: n, providers: activeProviders})
		}
	}
	return out
}

func reconcileFlat(parent dom.Element, oldFlat, newFlat flatNode, realNodes []dom.Node, anchor dom.Node) {
	for i := 0; i < len(newFlat.providers); i++ {
		newProv := newFlat.providers[i]
		setDOMParent(newProv, parent)
		if i < len(oldFlat.providers) {
			oldProv := oldFlat.providers[i]
			newProv.updateFrom(oldProv)
		}
	}

	for _, prov := range newFlat.providers {
		prov.pushEntry()
	}

	reconcile(parent, oldFlat.node, newFlat.node, realNodes, anchor)

	for i := len(newFlat.providers) - 1; i >= 0; i-- {
		newFlat.providers[i].popEntry()
	}
}

func instantiateFlat(parent dom.Element, newFlat flatNode) []dom.Node {
	for _, prov := range newFlat.providers {
		setDOMParent(prov, parent)
		prov.initEntry()
		prov.pushEntry()
	}
	defer func() {
		for i := len(newFlat.providers) - 1; i >= 0; i-- {
			newFlat.providers[i].popEntry()
		}
	}()
	setDOMParent(newFlat.node, parent)
	return newFlat.node.Instantiate(parent.OwnerDocument())
}

var keyMapPool = sync.Pool{
	New: func() any {
		return make(map[string]int)
	},
}

var keylessMapPool = sync.Pool{
	New: func() any {
		return make(map[string][]int)
	},
}

var keylessIterPool = sync.Pool{
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

var nodeSliceMapPool = sync.Pool{
	New: func() any {
		return make(map[Node][]dom.Node, 32)
	},
}

func reconcileChildren(parent dom.Element, oldParent, newParent Node, oldChildren, newChildren []Node, anchor dom.Node) {
	hasP := false
	if op, ok := oldParent.(nodeInternal); ok && op.hasDirectProvider() {
		hasP = true
	} else if np, ok := newParent.(nodeInternal); ok && np.hasDirectProvider() {
		hasP = true
	}

	if hasP {
		reconcileChildrenFlat(parent, oldChildren, newChildren, anchor)
		return
	}

	if len(oldChildren) == 0 {
		for i := range newChildren {
			if newChildren[i] != nil {
				setDOMParent(newChildren[i], parent)
				newReals := newChildren[i].Instantiate(parent.OwnerDocument())
				for _, newReal := range newReals {
					safeInsertBefore(parent, newReal, anchor)
				}
			}
		}
		return
	}

	if len(newChildren) == 0 {
		for i := range oldChildren {
			if oldChildren[i] != nil {
				realNodes := oldChildren[i].(nodeInternal).realNodes()
				destroyNode(oldChildren[i])
				for _, realNode := range realNodes {
					if realNode != nil {
						if isParent(realNode, parent) {
							parent.RemoveChild(realNode)
						}
						ClearAllSubscriptions(realNode)
					}
				}
			}
		}
		return
	}

	if len(oldChildren) == 1 && len(newChildren) == 1 {
		oldChild := oldChildren[0]
		newChild := newChildren[0]
		if sameNode(oldChild, newChild) {
			var realNodes []dom.Node
			if oldChild != nil {
				realNodes = oldChild.(nodeInternal).realNodes()
			}
			reconcile(parent, oldChild, newChild, realNodes, nil)
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

	var oldKeyMap map[string]int
	var oldKeylessMap map[string][]int
	var oldKeylessIter map[string]int
	defer func() {
		if oldKeyMap != nil {
			clear(oldKeyMap)
			keyMapPool.Put(oldKeyMap)
		}
		if oldKeylessMap != nil {
			for k := range oldKeylessMap {
				oldKeylessMap[k] = oldKeylessMap[k][:0]
				delete(oldKeylessMap, k)
			}
			keylessMapPool.Put(oldKeylessMap)
		}
		if oldKeylessIter != nil {
			clear(oldKeylessIter)
			keylessIterPool.Put(oldKeylessIter)
		}
	}()

	insertBeforeRef := anchor
	var nodeMap map[Node][]dom.Node

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
			var realNodes []dom.Node
			if nodeMap != nil {
				realNodes = nodeMap[oldStartNode]
			} else {
				realNodes = oldChildren[oldStartIdx].(nodeInternal).realNodes()
			}
			refNode := findNextDOMAnchor(oldS, oldStartIdx+1, oldEndIdx, nodeMap, oldChildren, insertBeforeRef)
			reconcile(parent, oldStartNode, newStartNode, realNodes, refNode)
			oldStartIdx++
			newStartIdx++
			continue
		}

		// Case 2: Old End == New End (no move needed)
		if sameNode(oldEndNode, newEndNode) {
			var realNodes []dom.Node
			if nodeMap != nil {
				realNodes = nodeMap[oldEndNode]
			} else {
				realNodes = oldChildren[oldEndIdx].(nodeInternal).realNodes()
			}
			refNode := findNextDOMAnchor(oldS, oldEndIdx+1, n-1, nodeMap, oldChildren, insertBeforeRef)
			reconcile(parent, oldEndNode, newEndNode, realNodes, refNode)
			newReals := newEndNode.(nodeInternal).realNodes()
			if len(newReals) > 0 {
				insertBeforeRef = newReals[0]
			}
			oldEndIdx--
			newEndIdx--
			continue
		}

		// Case 3-5: Move needed. Ensure map is populated.
		if nodeMap == nil {
			nodeMap = nodeSliceMapPool.Get().(map[Node][]dom.Node)
			for i := range n {
				if oldChildren[i] != nil {
					nodeMap[oldChildren[i]] = oldChildren[i].(nodeInternal).realNodes()
				}
			}
		}

		// Case 3: Old Start goes to New End -> move it after current oldEnd
		if sameNode(oldStartNode, newEndNode) {
			domNodes := nodeMap[oldStartNode]
			var afterEndRef dom.Node
			endNodes := nodeMap[oldEndNode]
			if len(endNodes) > 0 {
				afterEndRef = endNodes[len(endNodes)-1].NextSibling()
			}
			reconcile(parent, oldStartNode, newEndNode, domNodes, afterEndRef)
			for _, node := range domNodes {
				safeInsertBefore(parent, node, afterEndRef)
			}
			newReals := newEndNode.(nodeInternal).realNodes()
			if len(newReals) > 0 {
				insertBeforeRef = newReals[0]
			}
			oldS[oldStartIdx] = nil
			oldStartIdx++
			newEndIdx--
			continue
		}

		// Case 4: Old End goes to New Start -> move it before current oldStart
		if sameNode(oldEndNode, newStartNode) {
			domNodes := nodeMap[oldEndNode]
			var refNode dom.Node
			for i := oldStartIdx; i < oldEndIdx; i++ {
				if oldS[i] != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i]]
					} else {
						oldReals = oldChildren[i].(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			reconcile(parent, oldEndNode, newStartNode, domNodes, refNode)
			for _, node := range domNodes {
				safeInsertBefore(parent, node, refNode)
			}
			oldS[oldEndIdx] = nil
			oldEndIdx--
			newStartIdx++
			continue
		}

		// Case 5: Complex lookup by key
		if oldKeyMap == nil {
			oldKeyMap = keyMapPool.Get().(map[string]int)
			for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
				if oldS[idx] != nil {
					if key := oldS[idx].Key(); key != "" {
						oldKeyMap[key] = idx
					}
				}
			}
		}

		newKey := newStartNode.Key()
		matchedIdx := -1
		if newKey != "" {
			if idx, found := oldKeyMap[newKey]; found && oldS[idx] != nil {
				matchedIdx = idx
			}
		} else {
			if oldKeylessMap == nil {
				oldKeylessMap = keylessMapPool.Get().(map[string][]int)
				oldKeylessIter = keylessIterPool.Get().(map[string]int)
				for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
					if oldS[idx] != nil && oldS[idx].Key() == "" {
						tag := oldS[idx].TagName()
						oldKeylessMap[tag] = append(oldKeylessMap[tag], idx)
					}
				}
			}
			tag := newStartNode.TagName()
			indices := oldKeylessMap[tag]
			iter := oldKeylessIter[tag]
			for iter < len(indices) {
				idx := indices[iter]
				oldKeylessIter[tag] = iter + 1
				if idx >= oldStartIdx && idx <= oldEndIdx && oldS[idx] != nil {
					matchedIdx = idx
					break
				}
				iter++
			}
		}

		if matchedIdx != -1 {
			if EnableDevMode && newKey == "" {
				kitelog.Warn("Keyless node shifting detected during reconciliation. This can cause layout corruption and visual ordering issues. Consider adding a unique 'Key' property.",
					"parent", parent.TagName(),
					"child", newStartNode.TagName(),
				)
			}
			matchedNode := oldS[matchedIdx]
			matchedReal := nodeMap[matchedNode]
			var refNode dom.Node
			for i := oldStartIdx; i <= oldEndIdx; i++ {
				if i == matchedIdx {
					continue
				}
				if oldS[i] != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i]]
					} else {
						oldReals = oldChildren[i].(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			reconcile(parent, matchedNode, newStartNode, matchedReal, refNode)
			for _, node := range matchedReal {
				safeInsertBefore(parent, node, refNode)
			}
			if key := matchedNode.Key(); key != "" && oldKeyMap != nil {
				delete(oldKeyMap, key)
			}
			oldS[matchedIdx] = nil
		} else {
			setDOMParent(newStartNode, parent)
			newReals := newStartNode.Instantiate(parent.OwnerDocument())
			var refNode dom.Node
			// We cannot just blindly use `insertBeforeRef` here. If the previous `oldStartNode`
			// was an `emptyNode` (which generates no real DOM nodes), it would fail to provide
			// a valid DOM anchor and default to `nil`, incorrectly appending the new nodes at the end.
			// Instead, we scan forward through the remaining old nodes to find the first one
			// that has actual real DOM nodes attached to it, and use that as our anchor.
			for i := oldStartIdx; i <= oldEndIdx; i++ {
				if oldS[i] != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i]]
					} else {
						oldReals = oldChildren[i].(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			for _, node := range newReals {
				safeInsertBefore(parent, node, refNode)
			}
		}
		newStartIdx++
	}

	// Insert remaining new nodes at the end
	if newStartIdx <= newEndIdx {
		var refNode dom.Node
		for i := oldStartIdx; i <= oldEndIdx; i++ {
			if oldS[i] != nil {
				var oldReals []dom.Node
				if nodeMap != nil {
					oldReals = nodeMap[oldS[i]]
				} else {
					oldReals = oldChildren[i].(nodeInternal).realNodes()
				}
				if len(oldReals) > 0 {
					refNode = oldReals[0]
					break
				}
			}
		}
		if refNode == nil {
			refNode = insertBeforeRef
		}
		for newStartIdx <= newEndIdx {
			if newChildren[newStartIdx] != nil {
				setDOMParent(newChildren[newStartIdx], parent)
				newReals := newChildren[newStartIdx].Instantiate(parent.OwnerDocument())
				for _, newReal := range newReals {
					safeInsertBefore(parent, newReal, refNode)
				}
			}
			newStartIdx++
		}
	}

	// Remove remaining unmatched old nodes
	for i := oldStartIdx; i <= oldEndIdx; i++ {
		if oldS[i] != nil {
			var oldReals []dom.Node
			if nodeMap != nil {
				oldReals = nodeMap[oldS[i]]
			} else {
				oldReals = oldChildren[i].(nodeInternal).realNodes()
			}
			destroyNode(oldS[i])
			for _, oldReal := range oldReals {
				if oldReal != nil {
					if isParent(oldReal, parent) {
						parent.RemoveChild(oldReal)
					}
					ClearAllSubscriptions(oldReal)
				}
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
		nodeSliceMapPool.Put(nodeMap)
	}
}

func reconcileChildrenFlat(parent dom.Element, oldChildren, newChildren []Node, anchor dom.Node) {
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
			for i := range *flatOldPtr {
				(*flatOldPtr)[i] = flatNode{}
			}
			*flatOldPtr = (*flatOldPtr)[:0]
			flatNodeSlicePool.Put(flatOldPtr)
		}
		if flatNewPtr != nil {
			for i := range *flatNewPtr {
				(*flatNewPtr)[i] = flatNode{}
			}
			*flatNewPtr = (*flatNewPtr)[:0]
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
			newReals := instantiateFlat(parent, flatNew[i])
			for _, newReal := range newReals {
				safeInsertBefore(parent, newReal, anchor)
			}
		}
		return
	}

	if len(flatNew) == 0 {
		for i := range flatOld {
			realNodes := flatOld[i].node.(nodeInternal).realNodes()
			destroyNode(flatOld[i].node)
			for _, realNode := range realNodes {
				if realNode != nil {
					if isParent(realNode, parent) {
						parent.RemoveChild(realNode)
					}
					ClearAllSubscriptions(realNode)
				}
			}
		}
		return
	}

	if len(flatOld) == 1 && len(flatNew) == 1 {
		oldChild := flatOld[0]
		newChild := flatNew[0]
		if sameNode(oldChild.node, newChild.node) {
			var realNodes = oldChild.node.(nodeInternal).realNodes()
			reconcileFlat(parent, oldChild, newChild, realNodes, nil)
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

	var oldKeyMap map[string]int
	var oldKeylessMap map[string][]int
	var oldKeylessIter map[string]int
	defer func() {
		if oldKeyMap != nil {
			clear(oldKeyMap)
			keyMapPool.Put(oldKeyMap)
		}
		if oldKeylessMap != nil {
			for k := range oldKeylessMap {
				oldKeylessMap[k] = oldKeylessMap[k][:0]
				delete(oldKeylessMap, k)
			}
			keylessMapPool.Put(oldKeylessMap)
		}
		if oldKeylessIter != nil {
			clear(oldKeylessIter)
			keylessIterPool.Put(oldKeylessIter)
		}
	}()

	insertBeforeRef := anchor
	var nodeMap map[Node][]dom.Node

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
			var realNodes []dom.Node
			if nodeMap != nil {
				realNodes = nodeMap[oldStartNode.node]
			} else {
				realNodes = flatOld[oldStartIdx].node.(nodeInternal).realNodes()
			}
			refNode := findNextDOMAnchorFlat(oldS, oldStartIdx+1, oldEndIdx, nodeMap, flatOld, insertBeforeRef)
			reconcileFlat(parent, oldStartNode, newStartNode, realNodes, refNode)
			oldStartIdx++
			newStartIdx++
			continue
		}

		// Case 2: Old End == New End (no move needed)
		if sameNode(oldEndNode.node, newEndNode.node) {
			var realNodes []dom.Node
			if nodeMap != nil {
				realNodes = nodeMap[oldEndNode.node]
			} else {
				realNodes = flatOld[oldEndIdx].node.(nodeInternal).realNodes()
			}
			refNode := findNextDOMAnchorFlat(oldS, oldEndIdx+1, n-1, nodeMap, flatOld, insertBeforeRef)
			reconcileFlat(parent, oldEndNode, newEndNode, realNodes, refNode)
			newReals := newEndNode.node.(nodeInternal).realNodes()
			if len(newReals) > 0 {
				insertBeforeRef = newReals[0]
			}
			oldEndIdx--
			newEndIdx--
			continue
		}

		// Case 3-5: Move needed. Ensure map is populated.
		if nodeMap == nil {
			nodeMap = nodeSliceMapPool.Get().(map[Node][]dom.Node)
			for i := range n {
				nodeMap[flatOld[i].node] = flatOld[i].node.(nodeInternal).realNodes()
			}
		}

		// Case 3: Old Start goes to New End -> move it after current oldEnd
		if sameNode(oldStartNode.node, newEndNode.node) {
			domNodes := nodeMap[oldStartNode.node]
			var afterEndRef dom.Node
			endNodes := nodeMap[oldEndNode.node]
			if len(endNodes) > 0 {
				afterEndRef = endNodes[len(endNodes)-1].NextSibling()
			}
			reconcileFlat(parent, oldStartNode, newEndNode, domNodes, afterEndRef)
			for _, node := range domNodes {
				safeInsertBefore(parent, node, afterEndRef)
			}
			newReals := newEndNode.node.(nodeInternal).realNodes()
			if len(newReals) > 0 {
				insertBeforeRef = newReals[0]
			}
			oldS[oldStartIdx] = flatNode{}
			oldStartIdx++
			newEndIdx--
			continue
		}

		// Case 4: Old End goes to New Start -> move it before current oldStart
		if sameNode(oldEndNode.node, newStartNode.node) {
			domNodes := nodeMap[oldEndNode.node]
			var refNode dom.Node
			for i := oldStartIdx; i < oldEndIdx; i++ {
				if oldS[i].node != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i].node]
					} else {
						oldReals = flatOld[i].node.(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			reconcileFlat(parent, oldEndNode, newStartNode, domNodes, refNode)
			for _, node := range domNodes {
				safeInsertBefore(parent, node, refNode)
			}
			oldS[oldEndIdx] = flatNode{}
			oldEndIdx--
			newStartIdx++
			continue
		}

		// Case 5: Complex lookup by key
		if oldKeyMap == nil {
			oldKeyMap = keyMapPool.Get().(map[string]int)
			for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
				if oldS[idx].node != nil {
					if key := oldS[idx].node.Key(); key != "" {
						oldKeyMap[key] = idx
					}
				}
			}
		}

		newKey := newStartNode.node.Key()
		matchedIdx := -1
		if newKey != "" {
			if idx, found := oldKeyMap[newKey]; found && oldS[idx].node != nil {
				matchedIdx = idx
			}
		} else {
			if oldKeylessMap == nil {
				oldKeylessMap = keylessMapPool.Get().(map[string][]int)
				oldKeylessIter = keylessIterPool.Get().(map[string]int)
				for idx := oldStartIdx; idx <= oldEndIdx; idx++ {
					if oldS[idx].node != nil && oldS[idx].node.Key() == "" {
						tag := oldS[idx].node.TagName()
						oldKeylessMap[tag] = append(oldKeylessMap[tag], idx)
					}
				}
			}
			tag := newStartNode.node.TagName()
			indices := oldKeylessMap[tag]
			iter := oldKeylessIter[tag]
			for iter < len(indices) {
				idx := indices[iter]
				oldKeylessIter[tag] = iter + 1
				if idx >= oldStartIdx && idx <= oldEndIdx && oldS[idx].node != nil {
					matchedIdx = idx
					break
				}
				iter++
			}
		}

		if matchedIdx != -1 {
			if EnableDevMode && newKey == "" {
				kitelog.Warn("Keyless node shifting detected during flat reconciliation. This can cause layout corruption and visual ordering issues. Consider adding a unique 'Key' property.",
					"parent", parent.TagName(),
					"child", newStartNode.node.TagName(),
				)
			}
			matchedNode := oldS[matchedIdx]
			matchedReal := nodeMap[matchedNode.node]
			var refNode dom.Node
			for i := oldStartIdx; i <= oldEndIdx; i++ {
				if i == matchedIdx {
					continue
				}
				if oldS[i].node != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i].node]
					} else {
						oldReals = flatOld[i].node.(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			reconcileFlat(parent, matchedNode, newStartNode, matchedReal, refNode)
			for _, node := range matchedReal {
				safeInsertBefore(parent, node, refNode)
			}
			if key := matchedNode.node.Key(); key != "" && oldKeyMap != nil {
				delete(oldKeyMap, key)
			}
			oldS[matchedIdx] = flatNode{}
		} else {
			newReals := instantiateFlat(parent, newStartNode)
			var refNode dom.Node
			// We cannot just blindly use `insertBeforeRef` here. If the previous `oldStartNode`
			// was an `emptyNode` (which generates no real DOM nodes), it would fail to provide
			// a valid DOM anchor and default to `nil`, incorrectly appending the new nodes at the end.
			// Instead, we scan forward through the remaining old nodes to find the first one
			// that has actual real DOM nodes attached to it, and use that as our anchor.
			for i := oldStartIdx; i <= oldEndIdx; i++ {
				if oldS[i].node != nil {
					var oldReals []dom.Node
					if nodeMap != nil {
						oldReals = nodeMap[oldS[i].node]
					} else {
						oldReals = flatOld[i].node.(nodeInternal).realNodes()
					}
					if len(oldReals) > 0 {
						refNode = oldReals[0]
						break
					}
				}
			}
			if refNode == nil {
				refNode = insertBeforeRef
			}
			for _, node := range newReals {
				safeInsertBefore(parent, node, refNode)
			}
		}
		newStartIdx++
	}

	// Insert remaining new nodes at the end
	if newStartIdx <= newEndIdx {
		var refNode dom.Node
		for i := oldStartIdx; i <= oldEndIdx; i++ {
			if oldS[i].node != nil {
				var oldReals []dom.Node
				if nodeMap != nil {
					oldReals = nodeMap[oldS[i].node]
				} else {
					oldReals = flatOld[i].node.(nodeInternal).realNodes()
				}
				if len(oldReals) > 0 {
					refNode = oldReals[0]
					break
				}
			}
		}
		if refNode == nil {
			refNode = insertBeforeRef
		}
		for newStartIdx <= newEndIdx {
			newReals := instantiateFlat(parent, flatNew[newStartIdx])
			for _, node := range newReals {
				safeInsertBefore(parent, node, refNode)
			}
			newStartIdx++
		}
	}

	// Remove remaining unmatched old nodes
	for i := oldStartIdx; i <= oldEndIdx; i++ {
		if oldS[i].node != nil {
			var oldReals []dom.Node
			if nodeMap != nil {
				oldReals = nodeMap[oldS[i].node]
			} else {
				oldReals = flatOld[i].node.(nodeInternal).realNodes()
			}
			destroyNode(oldS[i].node)
			for _, oldReal := range oldReals {
				if oldReal != nil {
					if isParent(oldReal, parent) {
						parent.RemoveChild(oldReal)
					}
					ClearAllSubscriptions(oldReal)
				}
			}
		}
	}

	if nodeMap != nil {
		clear(nodeMap)
		nodeSliceMapPool.Put(nodeMap)
	}
}

func sameNode(n1, n2 Node) bool {
	if n1 == nil || n2 == nil {
		return false
	}
	return n1.TagName() == n2.TagName() && n1.Key() == n2.Key()
}

type componentDOMParentSetter interface {
	setDOMParent(dom.Element)
}

func setDOMParent(n Node, parent dom.Element) {
	if n == nil {
		return
	}
	if setter, ok := n.(componentDOMParentSetter); ok {
		setter.setDOMParent(parent)
	}
}

func unwrapNode(n dom.Node) dom.Node {
	if n == nil {
		return nil
	}
	curr := n
	for {
		if u := curr.Unwrap(); u != nil && u != curr {
			curr = u
		} else {
			break
		}
	}
	return curr
}

func isParent(child dom.Node, parent dom.Node) bool {
	if child == nil || parent == nil {
		return false
	}
	p := child.Parent()
	if p == nil {
		return false
	}
	return unwrapNode(p) == unwrapNode(parent)
}

func safeInsertBefore(parent dom.Element, newReal, refNode dom.Node) {
	if newReal == nil {
		return
	}
	if refNode != nil && !isParent(refNode, parent) {
		refNode = nil
	}
	parent.InsertBefore(newReal, refNode)
}

func findNextDOMAnchor(oldS []Node, startIdx, endIdx int, nodeMap map[Node][]dom.Node, oldChildren []Node, insertBeforeRef dom.Node) dom.Node {
	for i := startIdx; i <= endIdx; i++ {
		if i >= 0 && i < len(oldS) && oldS[i] != nil {
			var oldReals []dom.Node
			if nodeMap != nil {
				oldReals = nodeMap[oldS[i]]
			} else {
				oldReals = oldChildren[i].(nodeInternal).realNodes()
			}
			if len(oldReals) > 0 {
				return oldReals[0]
			}
		}
	}
	return insertBeforeRef
}

func findNextDOMAnchorFlat(oldS []flatNode, startIdx, endIdx int, nodeMap map[Node][]dom.Node, flatOld []flatNode, insertBeforeRef dom.Node) dom.Node {
	for i := startIdx; i <= endIdx; i++ {
		if i >= 0 && i < len(oldS) && oldS[i].node != nil {
			var oldReals []dom.Node
			if nodeMap != nil {
				oldReals = nodeMap[oldS[i].node]
			} else {
				oldReals = flatOld[i].node.(nodeInternal).realNodes()
			}
			if len(oldReals) > 0 {
				return oldReals[0]
			}
		}
	}
	return insertBeforeRef
}
