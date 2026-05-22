package inspector

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

//go:embed dashboard.html
var dashboardHTML []byte

type Inspector struct {
	eng     *engine.Engine
	mu      sync.RWMutex
	clients []chan []byte
}

// Options configures the inspector.
type Options struct {
	// NoOpen prevents the inspector from automatically opening the browser.
	NoOpen bool
}

func Attach(eng *engine.Engine, addr string, opts Options) error {
	insp := &Inspector{
		eng: eng,
	}

	eng.OnFrameRendered(insp.broadcast)

	mux := http.NewServeMux()
	mux.HandleFunc("/", insp.handleIndex)
	mux.HandleFunc("/stream", insp.handleStream)
	mux.HandleFunc("/dump", insp.handleDump)

	// Try to listen on the requested address.
	// If the port is 0 or taken, net.Listen will handle it if we are smart.
	// If it fails with "address already in use", we try port 0.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Warn("inspector: requested address failed, trying random port", "addr", addr, "err", err)
		l, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("inspector: failed to find any available port: %w", err)
		}
	}

	actualAddr := l.Addr().String()
	slog.Info("inspector: server starting", "addr", actualAddr)

	// Start server in a background goroutine
	go func() {
		if err := http.Serve(l, mux); err != nil {
			slog.Error("inspector: server error", "err", err)
		}
	}()

	// Auto-open browser
	if !opts.NoOpen {
		go openBrowser("http://" + actualAddr)
	}

	return nil
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

func (i *Inspector) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(dashboardHTML)
}

func (i *Inspector) handleDump(w http.ResponseWriter, r *http.Request) {
	tmpFile := "kite-inspector-dump.json"
	if err := i.eng.Dump(tmpFile); err != nil {
		http.Error(w, fmt.Sprintf("failed to create dump: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile)

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		http.Error(w, "failed to read dump", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=kite-dump.json")
	w.Write(data)
}

func (i *Inspector) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 1)
	i.mu.Lock()
	i.clients = append(i.clients, ch)
	i.mu.Unlock()

	defer func() {
		i.mu.Lock()
		for idx, c := range i.clients {
			if c == ch {
				i.clients = append(i.clients[:idx], i.clients[idx+1:]...)
				break
			}
		}
		i.mu.Unlock()
	}()

	// Send initial state
	i.broadcast()

	for {
		select {
		case data := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (i *Inspector) broadcast() {
	payload := i.takeSnapshot()
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, ch := range i.clients {
		select {
		case ch <- data:
		default:
			// Client slow, skip or drop
		}
	}
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

func (i *Inspector) takeSnapshot() *InspectorPayload {
	doc := i.eng.Document()
	rv := i.eng.RenderView()

	boundsMap := make(map[layout.Node]layout.Rect)
	i.computeAllBounds(rv.Fragment(), layout.Point{X: 0, Y: 0}, boundsMap)

	// Also compute bounds for overlays
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

	s := &FragmentSnapshot{
		Name:   name,
		Offset: offset,
		Size:   f.Size,
	}

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
		s.BreakToken = &BreakTokenSnapshot{
			ChildIndex: f.BreakToken.ChildIndex,
		}
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
			// Union of fragments for the same node
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
		childOrigin := layout.Point{
			X: origin.X + child.Offset.X,
			Y: origin.Y + child.Offset.Y,
		}
		i.computeAllBounds(child.Fragment, childOrigin, m)
	}
}

func (i *Inspector) snapshotNode(n dom.Node, boundsMap map[layout.Node]layout.Rect) *NodeSnapshot {
	s := &NodeSnapshot{
		Kind:        n.Kind().String(),
		Name:        n.NodeName(),
		TextContent: n.TextContent(),
	}

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
