package kitex

import (
	"fmt"
	"reflect"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
)

type VDOMSnapshot struct {
	Name        string          `json:"name"`
	Key         string          `json:"key,omitempty"`
	Props       any             `json:"props,omitempty"`
	State       []any           `json:"state,omitempty"`
	DeclFile    string          `json:"declFile,omitempty"`
	DeclLine    int             `json:"declLine,omitempty"`
	InstFile    string          `json:"instFile,omitempty"`
	InstLine    int             `json:"instLine,omitempty"`
	DomID       string          `json:"domId,omitempty"`
	DomUniqueID string          `json:"domUniqueId,omitempty"`
	UniqueID    string          `json:"uniqueId,omitempty"`
	Rect        geom.Rect       `json:"rect"`
	Children    []*VDOMSnapshot `json:"children,omitempty"`
}

func BuildDevToolsSnapshot(eng *engine.Engine) any {
	renderMutex.Lock()
	defer renderMutex.Unlock()

	boundsMap := make(map[layout.Node]geom.Rect)
	if eng != nil {
		rv := eng.RenderView()
		if rv != nil {
			computeAllBounds(rv.Fragment(), geom.Point{X: 0, Y: 0}, boundsMap)
			for _, overlay := range rv.Overlays() {
				offset := geom.Point{}
				if cs := overlay.ComputedStyle(); cs != nil {
					offset.X = cs.Margin.Left
					offset.Y = cs.Margin.Top
				}
				computeAllBounds(overlay.Fragment(), offset, boundsMap)
			}
		}
	}

	var roots []*VDOMSnapshot
	counter := 0
	for container, root := range activeRoots {
		if root != nil {
			snap := snapshotVDOMNode(eng, root, container, boundsMap, &counter)
			if snap != nil {
				roots = append(roots, snap)
			}
		}
	}
	return roots
}

func computeAllBounds(frag *layout.Fragment, origin geom.Point, m map[layout.Node]geom.Rect) {
	if frag == nil {
		return
	}
	rect := geom.Rect{Origin: origin, Size: frag.Size}
	if frag.Node != nil {
		if _, ok := m[frag.Node]; !ok {
			m[frag.Node] = rect
		} else {
			existing := m[frag.Node]
			newRect := geom.Rect{
				Origin: geom.Point{
					X: min(existing.Origin.X, rect.Origin.X),
					Y: min(existing.Origin.Y, rect.Origin.Y),
				},
			}
			newRect.Size = geom.Size{
				Width:  max(existing.Origin.X+existing.Size.Width, rect.Origin.X+rect.Size.Width) - newRect.Origin.X,
				Height: max(existing.Origin.Y+existing.Size.Height, rect.Origin.Y+rect.Size.Height) - newRect.Origin.Y,
			}
			m[frag.Node] = newRect
		}
	}
	for _, child := range frag.Children {
		childOrigin := geom.Point{X: origin.X + child.Offset.X, Y: origin.Y + child.Offset.Y}
		computeAllBounds(child.Fragment, childOrigin, m)
	}
}

func snapshotVDOMNode(eng *engine.Engine, node Node, container dom.Element, boundsMap map[layout.Node]geom.Rect, counter *int) *VDOMSnapshot {
	if node == nil {
		return nil
	}

	*counter++
	snap := &VDOMSnapshot{
		Name:     node.TagName(),
		Key:      node.Key(),
		UniqueID: fmt.Sprintf("vdom-%d", *counter),
	}

	var refNode dom.Node
	if ni, ok := node.(nodeInternal); ok {
		refNode = ni.realNode()
	}

	if refNode != nil {
		if el, ok := refNode.(dom.Element); ok {
			snap.DomID = el.ID()
		}
		if eng != nil {
			if ro := eng.RenderObject(refNode); ro != nil {
				if r, ok := boundsMap[ro]; ok {
					snap.Rect = r
					snap.DomUniqueID = fmt.Sprintf("%s:%s:%s:%d,%d", refNode.Kind().String(), refNode.NodeName(), snap.DomID, r.Origin.X, r.Origin.Y)
				}
			}
		}
	}

	snap.Props = cleanProps(node.Props())

	if st, ok := node.(sourceTracker); ok {
		snap.DeclFile, snap.DeclLine, snap.InstFile, snap.InstLine = st.getSource()
	}

	if ci, ok := node.(componentNodeInspector); ok {
		hooks := ci.getHooks()
		for _, h := range hooks {
			if hv, ok := h.(hookValuer); ok {
				snap.State = append(snap.State, cleanProps(hv.getValue()))
			} else {
				snap.State = append(snap.State, cleanProps(h))
			}
		}

		rendered := ci.getRendered()
		if rendered != nil {
			childSnap := snapshotVDOMNode(eng, rendered, container, boundsMap, counter)
			if childSnap != nil {
				snap.Children = append(snap.Children, childSnap)
			}
		}
	} else {
		for _, child := range node.Children() {
			childSnap := snapshotVDOMNode(eng, child, container, boundsMap, counter)
			if childSnap != nil {
				snap.Children = append(snap.Children, childSnap)
			}
		}
	}

	return snap
}

func cleanProps(val any) any {
	if val == nil {
		return nil
	}
	v := reflect.ValueOf(val)
	return cleanValue(v)
}

func cleanValue(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return nil
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return cleanValue(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return cleanValue(v.Elem())
	case reflect.Struct:
		// Check if this struct implements dom.Node to avoid recursive DOM serialization
		if v.CanInterface() {
			if n, ok := v.Interface().(dom.Node); ok {
				return fmt.Sprintf("Node(%s)", n.NodeName())
			}
		}
		m := make(map[string]any)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // skip unexported fields
			}
			fv := v.Field(i)
			cleaned := cleanValue(fv)
			if cleaned != nil {
				m[f.Name] = cleaned
			}
		}
		return m
	case reflect.Map:
		m := make(map[string]any)
		for _, k := range v.MapKeys() {
			kv := v.MapIndex(k)
			cleaned := cleanValue(kv)
			if cleaned != nil {
				m[fmt.Sprint(k.Interface())] = cleaned
			}
		}
		return m
	case reflect.Slice, reflect.Array:
		var list []any
		for i := 0; i < v.Len(); i++ {
			cleaned := cleanValue(v.Index(i))
			if cleaned != nil {
				list = append(list, cleaned)
			}
		}
		return list
	default:
		if v.IsValid() && v.CanInterface() {
			return v.Interface()
		}
		return nil
	}
}
