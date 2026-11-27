package kitex

import (
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/masterkeysrd/kite/dom"
	kitelog "github.com/masterkeysrd/kite/log"
)

// --- Functional Components & Hooks --------------------------------------------

// componentRef holds a stable pointer to the active ComponentNode.
// This is used by hooks to always target the active component instance
// without needing to re-allocate getters/setters on every render cycle.
type componentRef struct {
	mu   sync.Mutex
	node componentInstance
}

type passthroughNode interface {
	setDOMParent(dom.Element)
	setVDOMParent(Node)
	getVDOMParent() Node
}

// componentInstance is the internal interface implemented by all ComponentNode instances
// to manage hooks state and updates.
type componentInstance interface {
	Node
	passthroughNode
	getHookState(index int) (any, bool)
	setHookState(index int, val any)
	incrementHookIndex() int
	MarkDirty()
	IsDirty() bool
	ClearDirty()
	getRef() *componentRef
	realNodes() []dom.Node
	setRefs([]dom.Node)
	Rendered() Node
	ReRender() Node
	Destroy()
	restoreContexts() func()
	getDOMParent() dom.Element
	setDOMAnchor(dom.Node)
	getDOMAnchor() dom.Node
}

// ComponentNode represents a declarative functional component in the VDOM tree.
// It implements the Node interface.
type ComponentNode[P any] struct {
	Name     string
	PropsVal P
	RenderFn func(P) Node

	// Internal state
	rendered     Node
	refs         []dom.Node
	hooks        []any
	hookIndex    int
	isFirst      bool
	dirty        atomic.Bool
	componentRef *componentRef
	key          string
	contextSnap  contextSnapshot
	domParent    dom.Element
	domAnchor    dom.Node
	vdomParent   Node

	// Memoization: complexityScore holds the node count of the last rendered
	// subtree. shouldMemo is true when complexityScore > memoComplexityThreshold,
	// indicating that a deep props comparison is cheaper than a full re-render.
	complexityScore int
	shouldMemo      bool

	declFile string
	declLine int
	instFile string
	instLine int

	pool *sync.Pool
}

func (c *ComponentNode[P]) setDOMParent(parent dom.Element) {
	c.domParent = parent
}

func (c *ComponentNode[P]) getDOMParent() dom.Element {
	return c.domParent
}

func (c *ComponentNode[P]) setDOMAnchor(anchor dom.Node) {
	c.domAnchor = anchor
}

func (c *ComponentNode[P]) getDOMAnchor() dom.Node {
	return c.domAnchor
}

func (c *ComponentNode[P]) setVDOMParent(parent Node) {
	c.vdomParent = parent
}

func (c *ComponentNode[P]) getVDOMParent() Node {
	return c.vdomParent
}

type componentNodeInspector interface {
	getHooks() []any
	getRendered() Node
	getDecl() (string, int)
	getInst() (string, int)
}

func (c *ComponentNode[P]) getHooks() []any        { return c.hooks }
func (c *ComponentNode[P]) getRendered() Node      { return c.rendered }
func (c *ComponentNode[P]) getDecl() (string, int) { return c.declFile, c.declLine }
func (c *ComponentNode[P]) getInst() (string, int) { return c.instFile, c.instLine }

func (c *ComponentNode[P]) getSource() (string, int, string, int) {
	return c.declFile, c.declLine, c.instFile, c.instLine
}

func (c *ComponentNode[P]) setSource(declFile string, declLine int, instFile string, instLine int) {
	c.declFile = declFile
	c.declLine = declLine
	c.instFile = instFile
	c.instLine = instLine
}

var _ Node = (*ComponentNode[struct{}])(nil)

func (c *ComponentNode[P]) TagName() string { return c.Name }
func (c *ComponentNode[P]) Props() any      { return c.PropsVal }
func (c *ComponentNode[P]) Key() string     { return c.key }
func (c *ComponentNode[P]) containsProvider() bool {
	if c.rendered != nil {
		if ni, ok := c.rendered.(nodeInternal); ok {
			return ni.containsProvider()
		}
	}
	return false
}
func (c *ComponentNode[P]) isProvider() bool        { return false }
func (c *ComponentNode[P]) hasDirectProvider() bool { return false }

func (c *ComponentNode[P]) Children() []Node {
	if c.rendered == nil {
		return nil
	}
	return []Node{c.rendered}
}

func (c *ComponentNode[P]) Instantiate(doc dom.Document) []dom.Node {
	cr := &componentRef{}
	cr.node = c
	c.componentRef = cr

	c.contextSnap = captureContexts()

	pushCurrentComponent(c)
	c.isFirst = true
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	checkKeylessList(c.Name, c.rendered, c.declFile, c.declLine)
	c.isFirst = false
	popCurrentComponent()

	// Compute the complexity of the rendered subtree and decide whether future
	// updates should attempt memoization.
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold

	if c.rendered != nil {
		bindChild(c.rendered, c.domParent, c)
		c.refs = c.rendered.Instantiate(doc)
	}
	return c.refs
}

func (c *ComponentNode[P]) Update(els []dom.Node, old Node) {
	oldComp, ok := old.(*ComponentNode[P])
	if !ok {
		return
	}
	// Transfer state
	c.hooks = oldComp.hooks
	c.isFirst = false
	c.refs = oldComp.refs
	c.complexityScore = oldComp.complexityScore
	c.shouldMemo = oldComp.shouldMemo
	c.componentRef = oldComp.componentRef
	c.contextSnap = oldComp.contextSnap
	c.domParent = oldComp.domParent
	if c.vdomParent == nil {
		c.vdomParent = oldComp.vdomParent
	}
	if c.componentRef == nil {
		cr := &componentRef{}
		cr.node = c
		c.componentRef = cr
		oldComp.componentRef = c.componentRef
	} else {
		c.componentRef.mu.Lock()
		c.componentRef.node = c
		c.componentRef.mu.Unlock()
	}

	// Automatic memoization: when the rendered subtree is large enough and the
	// new props are deeply equal to the old props, skip the RenderFn entirely
	// and reuse the previously rendered subtree. The depth limit (3) caps the
	// worst-case reflection time so the frame budget is never blown by a single
	// pathological component.
	if c.shouldMemo && !oldComp.IsDirty() && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) {
		c.rendered = oldComp.rendered
		bindChild(c.rendered, c.domParent, c)
		return
	}

	pushCurrentComponent(c)
	c.hookIndex = 0
	newRendered := c.RenderFn(c.PropsVal)
	checkKeylessList(c.Name, newRendered, c.declFile, c.declLine)
	popCurrentComponent()

	c.rendered = newRendered
	bindChild(c.rendered, c.domParent, c)

	// Refresh the memoization score after re-rendering so it reflects the
	// current subtree shape.
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold
}

func (c *ComponentNode[P]) getHookState(index int) (any, bool) {
	if index < len(c.hooks) {
		return c.hooks[index], true
	}
	return nil, false
}

func (c *ComponentNode[P]) setHookState(index int, val any) {
	if index >= len(c.hooks) {
		newHooks := make([]any, index+1)
		copy(newHooks, c.hooks)
		c.hooks = newHooks
	}
	c.hooks[index] = val
}

func (c *ComponentNode[P]) incrementHookIndex() int {
	idx := c.hookIndex
	c.hookIndex++
	return idx
}

func (c *ComponentNode[P]) getRef() *componentRef {
	return c.componentRef
}

func (c *ComponentNode[P]) realNodes() []dom.Node {
	return c.refs
}

func (c *ComponentNode[P]) setRefs(els []dom.Node) {
	c.refs = els
}

func (c *ComponentNode[P]) Rendered() Node {
	return c.rendered
}

func (c *ComponentNode[P]) ReRender() Node {
	pushCurrentComponent(c)
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	checkKeylessList(c.Name, c.rendered, c.declFile, c.declLine)
	popCurrentComponent()
	bindChild(c.rendered, c.domParent, c)
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold
	return c.rendered
}

// complexity for a ComponentNode exposes the pre-computed score of its last
// rendered subtree so parent elements can accumulate it accurately.
func (c *ComponentNode[P]) complexity() int {
	return c.complexityScore
}

// IsDirty returns whether the component is dirty.
func (c *ComponentNode[P]) IsDirty() bool {
	return c.dirty.Load()
}

func (c *ComponentNode[P]) Release() {
	c.rendered = nil
	c.hooks = nil
	c.componentRef = nil
	c.contextSnap = contextSnapshot{}
	c.vdomParent = nil
}

func (c *ComponentNode[P]) restoreContexts() func() {
	if len(c.contextSnap.states) == 0 {
		return noopCleanup
	}
	prevSnap := captureContexts()
	restoreContexts(c.contextSnap)
	return func() {
		restoreContexts(prevSnap)
	}
}

func noopCleanup() {}

// ClearDirty clears the dirty flag of the component.
func (c *ComponentNode[P]) ClearDirty() {
	c.dirty.Store(false)
}

// MarkDirty flags the component as dirty.
//
// The flag is atomic because async sources (promise callbacks, stream
// iterators, timers) legitimately flag components from worker goroutines while
// the reconciler reads and clears it on the main thread. Every call notifies the
// reconciler: the flag alone does not prove the component is queued, since it
// can be raised while the component is already mid-render and therefore already
// removed from the dirty list.
func (c *ComponentNode[P]) MarkDirty() {
	c.dirty.Store(true)
	if OnComponentDirty != nil {
		OnComponentDirty(c)
	}
}

// Destroy cleans up component state, runs effect cleanups, and recursively destroys rendered subtree.
func (c *ComponentNode[P]) Destroy() {
	for _, h := range c.hooks {
		if eff, ok := h.(*effectHookState); ok {
			if eff.cleanup != nil {
				eff.cleanup()
				eff.cleanup = nil
			}
			eff.pending = false
		}
		if unsub, ok := h.(contextUnsubscriber); ok {
			unsub.unsubscribe(c)
		}
		if cleaner, ok := h.(hookCleaner); ok {
			cleaner.cleanHook()
		}
	}
	if c.componentRef != nil {
		c.componentRef.mu.Lock()
		c.componentRef.node = nil
		c.componentRef.mu.Unlock()
		c.componentRef = nil
	}
	if c.rendered != nil {
		destroyNode(c.rendered)
	}
}

// destroyNode recursively walks the subtree and destroys any component instances.
func destroyNode(n Node) {
	if n == nil {
		return
	}
	if comp, ok := n.(componentInstance); ok {
		comp.Destroy()
	} else {
		for _, child := range n.Children() {
			destroyNode(child)
		}
	}
	n.Release()
}

// OnComponentDirty is a package-level hook triggered when a component becomes dirty (e.g. state change).
var OnComponentDirty func(node Node)

// --- Execution Stack ----------------------------------------------------------

var (
	renderStackMutex sync.Mutex
	renderStack      = make([]any, 0, 32)
)

func pushCurrentComponent(c any) {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	renderStack = append(renderStack, c)
}

func popCurrentComponent() {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	if len(renderStack) > 0 {
		renderStack[len(renderStack)-1] = nil
		renderStack = renderStack[:len(renderStack)-1]
	}
}

func getCurrentComponent() any {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	if len(renderStack) == 0 {
		return nil
	}
	return renderStack[len(renderStack)-1]
}

// --- FC & FCC wrappers ---------------------------------------------------------

type structTypeInfo struct {
	keyFieldIdx          []int
	hasKey               bool
	elementPropsFieldIdx []int
	hasElementProps      bool
	elementPropsKeyIdx   []int
	hasElementPropsKey   bool
	childrenFieldIdx     []int
	hasChildren          bool
}

var typeInfoCache sync.Map

func getTypeInfo(t reflect.Type) *structTypeInfo {
	if val, ok := typeInfoCache.Load(t); ok {
		return val.(*structTypeInfo)
	}

	info := &structTypeInfo{}
	if t.Kind() == reflect.Struct {
		// Look for Key field
		if f, found := t.FieldByName("Key"); found {
			if f.Type.Kind() == reflect.String {
				info.keyFieldIdx = f.Index
				info.hasKey = true
			}
		}
		// Look for ElementProps field
		if ep, found := t.FieldByName("ElementProps"); found {
			if ep.Type.Kind() == reflect.Struct {
				info.elementPropsFieldIdx = ep.Index
				info.hasElementProps = true
				if k, foundKey := ep.Type.FieldByName("Key"); foundKey {
					if k.Type.Kind() == reflect.String {
						info.elementPropsKeyIdx = k.Index
						info.hasElementPropsKey = true
					}
				}
			}
		}
		// Look for Children field
		if ch, found := t.FieldByName("Children"); found {
			if ch.Type == reflect.TypeFor[[]Node]() {
				info.childrenFieldIdx = ch.Index
				info.hasChildren = true
			}
		}
	}

	typeInfoCache.Store(t, info)
	return info
}

func getKey(props any) string {
	if props == nil {
		return ""
	}
	t := reflect.TypeOf(props)
	isPtr := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPtr = true
	}
	if t.Kind() != reflect.Struct {
		return ""
	}

	info := getTypeInfo(t)
	v := reflect.ValueOf(props)
	if isPtr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if info.hasKey {
		return v.FieldByIndex(info.keyFieldIdx).String()
	}
	if info.hasElementPropsKey {
		return v.FieldByIndex(info.elementPropsFieldIdx).FieldByIndex(info.elementPropsKeyIdx).String()
	}
	return ""
}

func getRefSetter(props any) refSetter {
	if props == nil {
		return nil
	}
	v := reflect.ValueOf(props)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	f := v.FieldByName("Ref")
	if f.IsValid() {
		switch f.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if !f.IsNil() {
				if setter, ok := reflect.TypeAssert[refSetter](f); ok {
					return setter
				}
			}
		}
	}

	ep := v.FieldByName("ElementProps")
	if ep.IsValid() && ep.Kind() == reflect.Struct {
		f = ep.FieldByName("Ref")
		if f.IsValid() {
			switch f.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
				if !f.IsNil() {
					if setter, ok := reflect.TypeAssert[refSetter](f); ok {
						return setter
					}
				}
			}
		}
	}
	return nil
}

// FC creates a functional component wrapper that does not take children.
func FC[P any](name string, render func(P) Node) func(P) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func(props P) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := &ComponentNode[P]{}
		*c = ComponentNode[P]{
			Name:     name,
			PropsVal: props,
			RenderFn: render,
			isFirst:  true,
			key:      getKey(&props),
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
		}
		return c
	}
}

// FCC creates a functional component wrapper that can accept children.
// If the Props type has a Children field of type []Node, the variadic children
// will be injected using reflection.
func FCC[P any](name string, render func(P) Node) func(P, ...Node) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func(props P, children ...Node) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		propsWithChildren := injectChildren(props, children)
		c := &ComponentNode[P]{}
		*c = ComponentNode[P]{
			Name:     name,
			PropsVal: propsWithChildren,
			RenderFn: render,
			isFirst:  true,
			key:      getKey(&propsWithChildren),
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
		}
		return c
	}
}

func injectChildren[P any](props P, children []Node) P {
	var val any = props
	if val == nil {
		return props
	}
	t := reflect.TypeOf(val)
	isPtr := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPtr = true
	}
	if t.Kind() != reflect.Struct {
		return props
	}

	info := getTypeInfo(t)
	if !info.hasChildren {
		return props
	}

	v := reflect.ValueOf(props)
	if isPtr {
		if v.IsNil() {
			return props
		}
		f := v.Elem().FieldByIndex(info.childrenFieldIdx)
		if f.CanSet() {
			f.Set(reflect.ValueOf(children))
		}
		return props
	}

	ptr := reflect.New(t)
	ptr.Elem().Set(v)
	f := ptr.Elem().FieldByIndex(info.childrenFieldIdx)
	if f.CanSet() {
		f.Set(reflect.ValueOf(children))
	}
	return ptr.Elem().Interface().(P)
}

// SimpleFC creates a functional component wrapper for components with no props.
func SimpleFC(name string, render func() Node) func() Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func() Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := &ComponentNode[struct{}]{}
		*c = ComponentNode[struct{}]{
			Name:     name,
			PropsVal: struct{}{},
			RenderFn: func(_ struct{}) Node { return render() },
			isFirst:  true,
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
		}
		return c
	}
}

// SimpleFCC creates a functional component wrapper for components that only take children.
func SimpleFCC(name string, render func([]Node) Node) func(...Node) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func(children ...Node) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := &ComponentNode[[]Node]{}
		*c = ComponentNode[[]Node]{
			Name:     name,
			PropsVal: children,
			RenderFn: func(p []Node) Node { return render(p) },
			isFirst:  true,
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
		}
		return c
	}
}

func checkKeylessList(compName string, node Node, file string, line int) {
	if !EnableDevMode || node == nil {
		return
	}
	if node.TagName() == "#fragment" {
		children := node.Children()
		if len(children) > 1 {
			hasKeyless := false
			for _, child := range children {
				if child != nil && child.Key() == "" && child.TagName() != "#empty" && child.TagName() != "#text" {
					hasKeyless = true
					break
				}
			}
			if hasKeyless {
				kitelog.Warn("Component returned a fragment containing child nodes without explicit keys. Dynamic lists should have unique keys to prevent reconciliation and layout corruption.",
					"component", compName,
					"file", file,
					"line", line,
				)
			}
		}
	}
}
