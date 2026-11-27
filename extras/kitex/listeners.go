package kitex

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
)

// --- Event Subscription Registry ----------------------------------------------

const eventPressSubKey event.EventType = "__kitex_button_press"

var (
	subMutex      sync.Mutex
	subscriptions = make(map[dom.Node]map[event.EventType]event.Subscription)
)

func setSubscription(node dom.Node, typ event.EventType, sub event.Subscription) {
	subMutex.Lock()
	defer subMutex.Unlock()
	m, ok := subscriptions[node]
	if !ok {
		m = make(map[event.EventType]event.Subscription)
		subscriptions[node] = m
	}
	if oldSub, ok := m[typ]; ok {
		oldSub.Cancel()
	}
	m[typ] = sub
}

func clearSubscription(node dom.Node, typ event.EventType) {
	subMutex.Lock()
	defer subMutex.Unlock()
	if m, ok := subscriptions[node]; ok {
		if oldSub, ok := m[typ]; ok {
			oldSub.Cancel()
			delete(m, typ)
		}
	}
}

// ClearAllSubscriptions cancels all event subscriptions associated with a DOM node
// and removes the node (and all its descendants) from the tracking map. Reconcilers should invoke this when deleting a node.
func ClearAllSubscriptions(node dom.Node) {
	subMutex.Lock()
	defer subMutex.Unlock()
	clearAllSubscriptionsRec(node)
}

func clearAllSubscriptionsRec(node dom.Node) {
	if node == nil {
		return
	}
	if m, ok := subscriptions[node]; ok {
		for _, sub := range m {
			sub.Cancel()
		}
		delete(subscriptions, node)
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		clearAllSubscriptionsRec(child)
	}
}

type eface struct {
	_type unsafe.Pointer
	data  unsafe.Pointer
}

func funcEquals(f1, f2 any) bool {
	n1 := isNilFunc(f1)
	n2 := isNilFunc(f2)
	if n1 || n2 {
		return n1 == n2
	}

	// Two functions are only equal if both their code (entry point) and
	// closure context (captured variables data pointer) are identical.
	// Since reflect.ValueOf(f).Pointer() returns the entry point of the function,
	// and the eface data pointer points to the closure block containing captured
	// values, we must compare both to detect stale/updated closures.
	d1 := (*eface)(unsafe.Pointer(&f1)).data
	d2 := (*eface)(unsafe.Pointer(&f2)).data
	if d1 != d2 {
		return false
	}

	return reflect.ValueOf(f1).Pointer() == reflect.ValueOf(f2).Pointer()
}

func isNilFunc(f any) bool {
	if f == nil {
		return true
	}
	v := reflect.ValueOf(f)
	return v.Kind() == reflect.Func && v.IsNil()
}

func updateTypedListener[E event.Event](el element.Element, typ event.EventType, oldFn, newFn func(E)) {
	if oldFn == nil && newFn == nil {
		return
	}
	if oldFn != nil && newFn != nil && funcEquals(oldFn, newFn) {
		return
	}
	clearSubscription(el, typ)
	if newFn != nil {
		sub := el.AddEventListener(typ, func(ev event.Event) {
			if typed, ok := ev.(E); ok {
				newFn(typed)
			}
		})
		setSubscription(el, typ, sub)
	}
}

func updatePressListener(el element.Element, oldFn, newFn func()) {
	if oldFn == nil && newFn == nil {
		return
	}
	if oldFn != nil && newFn != nil && funcEquals(oldFn, newFn) {
		return
	}
	clearSubscription(el, eventPressSubKey)
	if newFn != nil {
		sub := el.AddEventListener(event.EventClick, func(ev event.Event) {
			if me, ok := ev.(*event.MouseEvent); ok {
				if me.Button == event.ButtonLeft || me.Button == event.ButtonNone {
					newFn()
				}
				return
			}
			newFn()
		})
		setSubscription(el, eventPressSubKey, sub)
	}
}

func setDisabled(el element.Element, d bool) {
	if dis, ok := el.(dom.Disableable); ok {
		dis.SetDisabled(d)
	}
}

var emptyElementProps ElementProps

// updateElementBase syncs core style, identity and listeners on any element.
func updateElementBase(el element.Element, old, new *ElementProps) {
	if old == nil {
		old = &emptyElementProps
	}
	if old.ID != new.ID {
		el.SetID(new.ID)
	}
	if old.Class != new.Class {
		el.SetClass(new.Class)
	}
	if old.Hidden != new.Hidden {
		el.SetHidden(new.Hidden)
	}
	if old.Disabled != new.Disabled {
		setDisabled(el, new.Disabled)
	}
	if old.CursorNavigable != new.CursorNavigable {
		el.SetCursorNavigable(new.CursorNavigable)
	}
	el.SetStyle(new.Style)

	// Reconcile attributes
	if len(old.Attributes) > 0 || len(new.Attributes) > 0 {
		for k := range old.Attributes {
			if _, ok := new.Attributes[k]; !ok {
				el.RemoveAttribute(k)
			}
		}
		for k, v := range new.Attributes {
			if oldVal, ok := old.Attributes[k]; !ok || oldVal != v {
				el.SetAttribute(k, v)
			}
		}
	}

	// Listeners
	updateTypedListener(el, event.EventKeyDown, old.OnKeyDown, new.OnKeyDown)
	updateTypedListener(el, event.EventKeyUp, old.OnKeyUp, new.OnKeyUp)
	updateTypedListener(el, event.EventKeyPress, old.OnKeyPress, new.OnKeyPress)
	updateTypedListener(el, event.EventMouseDown, old.OnMouseDown, new.OnMouseDown)
	updateTypedListener(el, event.EventMouseUp, old.OnMouseUp, new.OnMouseUp)
	updateTypedListener(el, event.EventMouseMove, old.OnMouseMove, new.OnMouseMove)
	updateTypedListener(el, event.EventClick, old.OnClick, new.OnClick)
	updateTypedListener(el, event.EventDrag, old.OnDrag, new.OnDrag)
	updateTypedListener(el, event.EventWheel, old.OnWheel, new.OnWheel)
	updateTypedListener(el, event.EventFocus, old.OnFocus, new.OnFocus)
	updateTypedListener(el, event.EventBlur, old.OnBlur, new.OnBlur)
	updateTypedListener(el, event.EventChange, old.OnChange, new.OnChange)
	updateTypedListener(el, event.EventInput, old.OnInput, new.OnInput)
	updateTypedListener(el, event.EventScroll, old.OnScroll, new.OnScroll)
	updateTypedListener(el, event.EventMouseEnter, old.OnMouseEnter, new.OnMouseEnter)
	updateTypedListener(el, event.EventMouseLeave, old.OnMouseLeave, new.OnMouseLeave)
	updateTypedListener(el, event.EventMouseOver, old.OnMouseOver, new.OnMouseOver)
	updateTypedListener(el, event.EventMouseOut, old.OnMouseOut, new.OnMouseOut)
}
