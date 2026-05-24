package devtools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
)

//go:embed static/index.html
var dashboardHTML []byte

type DevToolsServer struct {
	eng     *engine.Engine
	insp    *inspector.Inspector
	mu      sync.RWMutex
	clients []chan []byte
}

func NewDevToolsServer(eng *engine.Engine, insp *inspector.Inspector) *DevToolsServer {
	s := &DevToolsServer{eng: eng, insp: insp}
	return s
}

func (s *DevToolsServer) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/debug/trace/start", s.handleTraceStart)
	mux.HandleFunc("/debug/trace/stop", s.handleTraceStop)
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

func (s *DevToolsServer) handleTraceStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.eng.StartProfiling()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"started"}`))
}

func (s *DevToolsServer) handleTraceStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tracer := s.eng.StopProfiling()
	w.Header().Set("Content-Type", "application/json")
	if tracer == nil {
		w.Write([]byte(`[]`))
		return
	}
	if err := tracer.WriteJSON(w); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write trace: %v", err), http.StatusInternalServerError)
	}
}
