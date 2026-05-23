package devtools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/masterkeysrd/kite/devtools/inspector"
)

//go:embed static/index.html
var dashboardHTML []byte

type DevToolsServer struct {
	insp    *inspector.Inspector
	mu      sync.RWMutex
	clients []chan []byte
}

func NewDevToolsServer(insp *inspector.Inspector) *DevToolsServer {
	s := &DevToolsServer{insp: insp}
	return s
}

func (s *DevToolsServer) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/stream", s.handleStream)
}

func (s *DevToolsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(dashboardHTML)
}

func (s *DevToolsServer) handleStream(w http.ResponseWriter, r *http.Request) {
	// ... (implementation remains same)
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
	s.mu.Lock()
	s.clients = append(s.clients, ch)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		for idx, c := range s.clients {
			if c == ch {
				s.clients = append(s.clients[:idx], s.clients[idx+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	s.broadcast()

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

func (s *DevToolsServer) Broadcast() {
	s.broadcast()
}

func (s *DevToolsServer) broadcast() {
	payload := s.insp.TakeSnapshot()
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.clients {
		select {
		case ch <- data:
		default:
		}
	}
}
