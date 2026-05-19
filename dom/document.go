package dom

// idRegistrar is an unexported interface implemented by *document.
// element uses it (via a type assertion on ownerDocument) to maintain
// the O(1) ID map without creating an import cycle or exposing internals.
type idRegistrar interface {
	registerID(id string, el Element)
	unregisterID(id string)
}

// document is the concrete, unexported implementation of Document.
type document struct {
	baseNode
	byID    map[string]Element
	anchors map[string]Element
	// mutating is the reentrancy guard. It is true while an attach or detach
	// walk is executing. Ancestor-mutation inside a lifecycle callback is
	// detected by checking whether the node being mutated is outside the
	// subtree currently being walked.
	mutating bool
}

// Compile-time assertion.
var _ Document = (*document)(nil)

// NewDocument creates and returns a new, empty Document.
// The document itself is always connected.
func NewDocument() Document {
	d := &document{
		byID:    make(map[string]Element),
		anchors: make(map[string]Element),
	}
	d.self = d
	d.ownerDocument = d // the document owns itself
	d.connected = true  // the document is always connected
	d.kind = KindDocument
	d.name = "#document"
	d.needsSync = true
	return d
}

// CreateElement returns a new, detached Element with the given tag name owned
// by this document.
func (d *document) CreateElement(tag string, self Node) Element {
	return newElement(tag, d, self)
}

// CreateTextNode returns a new, detached TextNode with the given data owned
// by this document.
func (d *document) CreateTextNode(data string, self Node) TextNode {
	return newTextNode(data, d, self)
}

// GetElementByID returns the Element whose ID equals id, or nil.
func (d *document) GetElementByID(id string) Element {
	return d.byID[id]
}

// FindAnchor returns the Element registered under name in the anchor registry,
// or nil.
func (d *document) FindAnchor(name string) Element {
	return d.anchors[name]
}

// RegisterAnchor adds el to the anchor registry under name.
func (d *document) RegisterAnchor(name string, el Element) {
	if name != "" {
		d.anchors[name] = el
	}
}

// UnregisterAnchor removes the entry for name from the anchor registry.
func (d *document) UnregisterAnchor(name string) {
	delete(d.anchors, name)
}

func (d *document) Body() Element {
	for child := range d.ChildNodes() {
		if el, ok := child.(Element); ok {
			return el
		}
	}
	return nil
}

// --- attach / detach walks --------------------------------------------------

func (d *document) attachWalk(parent Node, child Node) {
	parentConnected := parent.IsConnected()
	d.walkAttach(child, parentConnected)
}

func (d *document) walkAttach(n Node, parentConnected bool) {
	b := asBase(n)
	if b == nil {
		return
	}

	// Steps 2 & 3: connection + registry + lifecycle.
	if parentConnected {
		b.connected = true
		d.registerNode(n)
		if lc, ok := n.(Lifecycle); ok {
			lc.OnConnected()
		}
	}

	// Recurse into children (pre-order: parent handled before children).
	for child := range n.ChildNodes() {
		d.walkAttach(child, parentConnected)
	}
}

func (d *document) detachWalk(parent Node, child Node) {
	parentConnected := parent.IsConnected()
	d.walkDetach(child, parentConnected)
}

func (d *document) walkDetach(n Node, parentConnected bool) {
	b := asBase(n)
	if b == nil {
		return
	}

	// Recurse into children first (post-order: children before parent).
	for child := range n.ChildNodes() {
		d.walkDetach(child, parentConnected)
	}

	if parentConnected {
		// Step 1: lifecycle callback fires while IsConnected() is still true.
		if lc, ok := n.(Lifecycle); ok {
			lc.OnDisconnected()
		}
		// Step 2: unregister then flip connected flag.
		d.unregisterNode(n)
		b.connected = false
	}
}

// --- registry helpers -------------------------------------------------------

func (d *document) registerNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			// Identity registry uses the current self pointer.
			d.byID[id] = el
		}
	}
}

func (d *document) unregisterNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			delete(d.byID, id)
		}
	}
}

func (d *document) registerID(id string, el Element) {
	if id != "" {
		d.byID[id] = el
	}
}

func (d *document) unregisterID(id string) {
	delete(d.byID, id)
}

func (d *document) asBase() *baseNode { return &d.baseNode }
