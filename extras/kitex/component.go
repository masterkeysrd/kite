package kitex

import (
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
	"weak"

	"github.com/masterkeysrd/kite/dom"
	kitelog "github.com/masterkeysrd/kite/log"
	"github.com/masterkeysrd/kite/style"
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
	Name              string
	PropsVal          P
	RenderFn          func(P) Node
	RenderContainerFn func(P, ...Node) Node
	PassedChildren    []Node

	// Internal state
	rendered         Node
	refs             []dom.Node
	hooks            []any
	hookIndex        int
	isFirst          bool
	dirty            atomic.Bool
	componentRef     *componentRef
	key              string
	contextSnap      contextSnapshot
	domParent        dom.Element
	domAnchor        dom.Node
	vdomParent       Node
	safelyComparable bool

	// Memoization: complexityScore holds the node count of the last rendered
	// subtree. shouldMemo is true when complexityScore > memoComplexityThreshold,
	// indicating that a props comparison is cheaper than a full re-render.
	complexityScore int
	shouldMemo      bool

	declFile string
	declLine int
	instFile string
	instLine int
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

func (c *ComponentNode[P]) invokeRender() Node {
	if c.RenderContainerFn != nil {
		return c.RenderContainerFn(c.PropsVal, c.PassedChildren...)
	}
	if c.RenderFn != nil {
		return c.RenderFn(c.PropsVal)
	}
	return nil
}

func (c *ComponentNode[P]) propsEqual(oldProps, newProps P) bool {
	if c.safelyComparable {
		return any(oldProps) == any(newProps)
	}
	return deepEqualProps(oldProps, newProps, 3)
}

func (c *ComponentNode[P]) Instantiate(doc dom.Document) []dom.Node {
	cr := &componentRef{}
	cr.node = c
	c.componentRef = cr

	c.contextSnap = captureContexts()

	pushCurrentComponent(c)
	c.isFirst = true
	c.hookIndex = 0
	c.rendered = c.invokeRender()
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
	// new props equal the old props, skip the render function entirely and reuse
	// the previously rendered subtree. Never skip when passed children exist,
	// because parent-supplied VDOM children must flow through RenderContainerFn
	// so child updates reconcile properly.
	if c.shouldMemo && !oldComp.IsDirty() &&
		len(c.PassedChildren) == 0 && len(oldComp.PassedChildren) == 0 &&
		c.propsEqual(oldComp.PropsVal, c.PropsVal) {
		c.rendered = oldComp.rendered
		bindChild(c.rendered, c.domParent, c)
		return
	}

	pushCurrentComponent(c)
	c.hookIndex = 0
	newRendered := c.invokeRender()
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
	c.rendered = c.invokeRender()
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
	c.PassedChildren = nil
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

// NoProps is the empty props type for components that accept no properties.
type NoProps = struct{}

// Component is a leaf functional component that does not accept children.
// Call comp() to render with zero-value P, or comp.Props(p) to pass explicit props.
type Component[P any] func() Node

// Props instantiates the component VDOM node with explicit properties.
func (c Component[P]) Props(props P) Node {
	if c == nil {
		return Empty()
	}
	regKey := *(*uintptr)(unsafe.Pointer(&c))
	val, ok := componentRegistry.Load(regKey)
	if !ok {
		return Empty()
	}
	wp, ok := val.(weak.Pointer[componentDescriptor[P]])
	if !ok {
		return Empty()
	}
	desc := wp.Value()
	runtime.KeepAlive(c)
	if desc == nil {
		return Empty()
	}

	var instFile string
	var instLine int
	if EnableDevMode {
		if _, file, line, ok := runtime.Caller(1); ok {
			instFile = file
			instLine = line
		}
	}
	return newComponentNode(desc, props, nil, instFile, instLine)
}

// Container is a functional component that accepts variadic VDOM children.
// Call comp(children...) to render with zero-value P, or comp.Props(p, children...) to pass explicit props.
type Container[P any] func(children ...Node) Node

// Props instantiates the container component VDOM node with explicit properties and children.
func (c Container[P]) Props(props P, children ...Node) Node {
	if c == nil {
		return Empty()
	}
	regKey := *(*uintptr)(unsafe.Pointer(&c))
	val, ok := componentRegistry.Load(regKey)
	if !ok {
		return Empty()
	}
	wp, ok := val.(weak.Pointer[componentDescriptor[P]])
	if !ok {
		return Empty()
	}
	desc := wp.Value()
	runtime.KeepAlive(c)
	if desc == nil {
		return Empty()
	}

	var instFile string
	var instLine int
	if EnableDevMode {
		if _, file, line, ok := runtime.Caller(1); ok {
			instFile = file
			instLine = line
		}
	}
	return newComponentNode(desc, props, children, instFile, instLine)
}

type componentDescriptor[P any] struct {
	name             string
	render           func(P) Node
	renderContainer  func(P, ...Node) Node
	declFile         string
	declLine         int
	keyFn            func(*P) string
	safelyComparable bool
	registryKey      uintptr
	cleanup          runtime.Cleanup
}

var componentRegistry sync.Map

type componentCleanupArg[P any] struct {
	key uintptr
	wp  weak.Pointer[componentDescriptor[P]]
}

func newComponentNode[P any](desc *componentDescriptor[P], props P, children []Node, instFile string, instLine int) *ComponentNode[P] {
	c := &ComponentNode[P]{
		Name:              desc.name,
		PropsVal:          props,
		RenderFn:          desc.render,
		RenderContainerFn: desc.renderContainer,
		PassedChildren:    children,
		isFirst:           true,
		safelyComparable:  desc.safelyComparable,
		declFile:          desc.declFile,
		declLine:          desc.declLine,
		instFile:          instFile,
		instLine:          instLine,
	}
	if desc.keyFn != nil {
		c.key = desc.keyFn(&c.PropsVal)
	}
	return c
}

var (
	styleType          = reflect.TypeFor[style.Style]()
	equatablePropsType = reflect.TypeFor[EquatableProps]()
)

func isSafelyComparable(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t == styleType || t.Implements(equatablePropsType) {
		return false
	}
	if !t.Comparable() {
		return false
	}
	switch t.Kind() {
	case reflect.Interface:
		return false
	case reflect.Array:
		return isSafelyComparable(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if !isSafelyComparable(t.Field(i).Type) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

func resolveStringFieldOffset(t reflect.Type, index []int) (uintptr, bool) {
	var offset uintptr
	cur := t
	for _, idx := range index {
		if cur.Kind() != reflect.Struct {
			return 0, false
		}
		f := cur.Field(idx)
		offset += f.Offset
		cur = f.Type
	}
	if cur.Kind() != reflect.String {
		return 0, false
	}
	return offset, true
}

func resolveStructFieldOffset(t reflect.Type, index []int) (uintptr, bool) {
	var offset uintptr
	cur := t
	for _, idx := range index {
		if cur.Kind() != reflect.Struct {
			return 0, false
		}
		f := cur.Field(idx)
		offset += f.Offset
		cur = f.Type
	}
	if cur.Kind() != reflect.Struct {
		return 0, false
	}
	return offset, true
}

func findKeyOffset(t reflect.Type) (uintptr, bool) {
	if f, found := t.FieldByName("Key"); found {
		if off, ok := resolveStringFieldOffset(t, f.Index); ok {
			return off, true
		}
	}
	if ep, found := t.FieldByName("ElementProps"); found && ep.Type.Kind() == reflect.Struct {
		if epOff, ok := resolveStructFieldOffset(t, ep.Index); ok {
			if k, foundKey := ep.Type.FieldByName("Key"); foundKey {
				if kOff, ok := resolveStringFieldOffset(ep.Type, k.Index); ok {
					return epOff + kOff, true
				}
			}
		}
	}
	return 0, false
}

func buildKeyFn[P any]() func(*P) string {
	t := reflect.TypeFor[P]()
	isPtr := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPtr = true
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	offset, ok := findKeyOffset(t)
	if !ok {
		return nil
	}
	if isPtr {
		return func(p *P) string {
			if p == nil {
				return ""
			}
			ptr := *(*unsafe.Pointer)(unsafe.Pointer(p))
			if ptr == nil {
				return ""
			}
			return *(*string)(unsafe.Add(ptr, offset))
		}
	}
	return func(p *P) string {
		if p == nil {
			return ""
		}
		return *(*string)(unsafe.Add(unsafe.Pointer(p), offset))
	}
}

// FC creates a leaf functional component that does not accept children.
func FC[P any](name string, render func(props P) Node) Component[P] {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	desc := &componentDescriptor[P]{
		name:             name,
		render:           render,
		declFile:         declFile,
		declLine:         declLine,
		keyFn:            buildKeyFn[P](),
		safelyComparable: isSafelyComparable(reflect.TypeFor[P]()),
	}

	var fn Component[P]
	fn = func() Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}
		var zero P
		return newComponentNode(desc, zero, nil, instFile, instLine)
	}

	regKey := *(*uintptr)(unsafe.Pointer(&fn))
	desc.registryKey = regKey
	wp := weak.Make(desc)
	componentRegistry.Store(regKey, wp)
	desc.cleanup = runtime.AddCleanup(desc, func(arg componentCleanupArg[P]) {
		componentRegistry.CompareAndDelete(arg.key, arg.wp)
	}, componentCleanupArg[P]{key: regKey, wp: wp})

	return fn
}

// FCC creates a container functional component that receives children directly as a parameter (zero reflection).
func FCC[P any](name string, render func(props P, children ...Node) Node) Container[P] {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	desc := &componentDescriptor[P]{
		name:             name,
		renderContainer:  render,
		declFile:         declFile,
		declLine:         declLine,
		keyFn:            buildKeyFn[P](),
		safelyComparable: isSafelyComparable(reflect.TypeFor[P]()),
	}

	var fn Container[P]
	fn = func(children ...Node) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}
		var zero P
		return newComponentNode(desc, zero, children, instFile, instLine)
	}

	regKey := *(*uintptr)(unsafe.Pointer(&fn))
	desc.registryKey = regKey
	wp := weak.Make(desc)
	componentRegistry.Store(regKey, wp)
	desc.cleanup = runtime.AddCleanup(desc, func(arg componentCleanupArg[P]) {
		componentRegistry.CompareAndDelete(arg.key, arg.wp)
	}, componentCleanupArg[P]{key: regKey, wp: wp})

	return fn
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
