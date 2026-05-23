package inspector

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

type Inspector struct {
	eng *engine.Engine
}

func New(eng *engine.Engine) *Inspector {
	return &Inspector{eng: eng}
}

type InspectorPayload struct {
	DOM          *NodeSnapshot       `json:"dom"`
	Overlays     []*NodeSnapshot     `json:"overlays,omitempty"`
	Fragments    *FragmentSnapshot   `json:"fragments"`
	OverlayFrags []*FragmentSnapshot `json:"overlayFragments,omitempty"`
}

type FragmentSnapshot struct {
	Name       string              `json:"name"`
	Offset     layout.Point        `json:"offset"`
	Size       layout.Size         `json:"size"`
	Clusters   []ClusterSnapshot   `json:"clusters,omitempty"`
	BreakToken *BreakTokenSnapshot `json:"breakToken,omitempty"`
	Children   []*FragmentSnapshot `json:"children,omitempty"`
}

type ClusterSnapshot struct {
	Text       string `json:"text"`
	Width      int    `json:"width"`
	BreakClass string `json:"breakClass"`
}

type BreakTokenSnapshot struct {
	ChildIndex int `json:"childIndex"`
}

type NodeSnapshot struct {
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	ID          string          `json:"id,omitempty"`
	Class       string          `json:"class,omitempty"`
	Rect        layout.Rect     `json:"rect"`
	ScrollX     int             `json:"scrollX,omitempty"`
	ScrollY     int             `json:"scrollY,omitempty"`
	Disabled    bool            `json:"disabled,omitempty"`
	Text        string          `json:"text,omitempty"`
	TextContent string          `json:"textContent,omitempty"`
	Computed    *style.Computed `json:"computed,omitempty"`
	Default     style.Style     `json:"default,omitempty"`
	Raw         style.Style     `json:"raw,omitempty"`
	Intrinsic   style.Style     `json:"intrinsic,omitempty"`
	Children    []*NodeSnapshot `json:"children,omitempty"`
}

func (i *Inspector) TakeSnapshot() *InspectorPayload {
	doc := i.eng.Document()
	rv := i.eng.RenderView()

	boundsMap := make(map[layout.Node]layout.Rect)
	i.computeAllBounds(rv.Fragment(), layout.Point{X: 0, Y: 0}, boundsMap)

	for _, overlay := range rv.Overlays() {
		offset := layout.Point{}
		if cs := overlay.ComputedStyle(); cs != nil {
			offset.X = cs.Margin.Left
			offset.Y = cs.Margin.Top
		}
		i.computeAllBounds(overlay.Fragment(), offset, boundsMap)
	}

	payload := &InspectorPayload{
		DOM:       i.snapshotNode(doc, boundsMap),
		Fragments: i.snapshotFragment(rv.Fragment(), layout.Point{X: 0, Y: 0}),
	}

	for overlayEl := range doc.Overlays() {
		payload.Overlays = append(payload.Overlays, i.snapshotNode(overlayEl, boundsMap))
	}

	for _, overlayRO := range rv.Overlays() {
		offset := layout.Point{}
		if cs := overlayRO.ComputedStyle(); cs != nil {
			offset.X = cs.Margin.Left
			offset.Y = cs.Margin.Top
		}
		payload.OverlayFrags = append(payload.OverlayFrags, i.snapshotFragment(overlayRO.Fragment(), offset))
	}

	return payload
}

func (i *Inspector) snapshotFragment(f *layout.Fragment, offset layout.Point) *FragmentSnapshot {
	if f == nil {
		return nil
	}
	name := "Anonymous"
	if f.Node != nil {
		if ro, ok := f.Node.(render.Object); ok {
			if et := ro.EventTarget(); et != nil {
				if n, ok := et.(dom.Node); ok {
					name = n.NodeName()
					if el, ok := n.(dom.Element); ok {
						if id := el.ID(); id != "" {
							name += "#" + id
						}
					}
				}
			}
		}
	}
	s := &FragmentSnapshot{Name: name, Offset: offset, Size: f.Size}
	if len(f.Text) > 0 {
		for _, c := range f.Text {
			s.Clusters = append(s.Clusters, ClusterSnapshot{
				Text:       string(c.Bytes),
				Width:      c.CellWidth,
				BreakClass: formatBreakClass(c.BreakClass),
			})
		}
	}
	if f.BreakToken != nil {
		s.BreakToken = &BreakTokenSnapshot{ChildIndex: f.BreakToken.ChildIndex}
	}
	for _, child := range f.Children {
		s.Children = append(s.Children, i.snapshotFragment(child.Fragment, child.Offset))
	}
	return s
}

func (i *Inspector) computeAllBounds(frag *layout.Fragment, origin layout.Point, m map[layout.Node]layout.Rect) {
	if frag == nil {
		return
	}
	rect := layout.Rect{Origin: origin, Size: frag.Size}
	if frag.Node != nil {
		if _, ok := m[frag.Node]; !ok {
			m[frag.Node] = rect
		} else {
			existing := m[frag.Node]
			newRect := layout.Rect{
				Origin: layout.Point{
					X: min(existing.Origin.X, rect.Origin.X),
					Y: min(existing.Origin.Y, rect.Origin.Y),
				},
			}
			newRect.Size = layout.Size{
				Width:  max(existing.Origin.X+existing.Size.Width, rect.Origin.X+rect.Size.Width) - newRect.Origin.X,
				Height: max(existing.Origin.Y+existing.Size.Height, rect.Origin.Y+rect.Size.Height) - newRect.Origin.Y,
			}
			m[frag.Node] = newRect
		}
	}
	for _, child := range frag.Children {
		childOrigin := layout.Point{X: origin.X + child.Offset.X, Y: origin.Y + child.Offset.Y}
		i.computeAllBounds(child.Fragment, childOrigin, m)
	}
}

func (i *Inspector) snapshotNode(n dom.Node, boundsMap map[layout.Node]layout.Rect) *NodeSnapshot {
	s := &NodeSnapshot{Kind: n.Kind().String(), Name: n.NodeName(), TextContent: n.TextContent()}
	if el, ok := n.(dom.Element); ok {
		s.ID = el.ID()
		s.Class = el.Class()
		s.ScrollX, s.ScrollY = el.Scroll()
		if d, ok := el.(dom.Disableable); ok {
			s.Disabled = d.IsDisabled()
		}
	}
	if n.Kind() == dom.KindText {
		if tn, ok := n.(dom.TextNode); ok {
			s.Text = tn.Data()
		}
	}
	if ro := n.RenderObject(); ro != nil {
		s.Computed = ro.ComputedStyle()
		s.Default = ro.DefaultStyle()
		s.Raw = ro.RawStyle()
		s.Intrinsic = ro.IntrinsicStyle()
		if rect, ok := boundsMap[ro]; ok {
			s.Rect = rect
		}
	}
	for child := range n.ChildNodes() {
		s.Children = append(s.Children, i.snapshotNode(child, boundsMap))
	}
	return s
}

func formatBreakClass(c text.BreakClass) string {
	switch c {
	case text.BreakNone:
		return "None"
	case text.BreakSoft:
		return "Soft"
	case text.BreakMandatory:
		return "Mandatory"
	case text.BreakAnywhere:
		return "Anywhere"
	default:
		return "Unknown"
	}
}
