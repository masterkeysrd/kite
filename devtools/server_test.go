package devtools_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
)

func TestDevToolsServer_Profiling(t *testing.T) {
	// Initialize Engine with a mock backend.
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Initialize Inspector and DevToolsServer.
	insp := inspector.New(eng)
	srv := devtools.NewDevToolsServer(eng, insp)

	// Set up HTTP ServeMux.
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	// Start httptest server.
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Start profiling.
	respStart, err := http.Post(ts.URL+"/debug/trace/start", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to POST /debug/trace/start: %v", err)
	}
	defer respStart.Body.Close()

	if respStart.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", respStart.StatusCode)
	}

	body, _ := io.ReadAll(respStart.Body)
	var startStatus map[string]string
	if err := json.Unmarshal(body, &startStatus); err != nil {
		t.Fatalf("failed to parse start response: %v", err)
	}
	if startStatus["status"] != "started" {
		t.Errorf("expected status 'started', got %s", startStatus["status"])
	}

	// 2. Submit a background task and run some frames.
	doneChan := make(chan struct{})
	eng.Scheduler().RunBackground(func(ctx context.Context) {
		time.Sleep(10 * time.Millisecond)
		close(doneChan)
	})

	// Run multiple frames to ensure we capture Sync, Tasks, Style, Layout, Paint.
	eng.Frame()

	// Wait for the background job to finish.
	select {
	case <-doneChan:
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for background job to run")
	}

	// Process job complete results on main thread.
	eng.Frame()

	// 3. Stop profiling and get trace events.
	respStop, err := http.Post(ts.URL+"/debug/trace/stop", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to POST /debug/trace/stop: %v", err)
	}
	defer respStop.Body.Close()

	if respStop.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", respStop.StatusCode)
	}

	var events []map[string]any
	if err := json.NewDecoder(respStop.Body).Decode(&events); err != nil {
		t.Fatalf("failed to decode stop response trace: %v", err)
	}

	// Verify we got some trace events.
	if len(events) == 0 {
		t.Errorf("expected non-empty trace events array")
	}

	// Assertions on the trace events format.
	hasFrame := false
	hasSync := false
	hasJobSubmit := false
	hasJobRun := false

	for _, ev := range events {
		name, _ := ev["name"].(string)
		ph, _ := ev["ph"].(string)
		tsVal, okTs := ev["ts"].(float64)
		tid, okTid := ev["tid"].(float64)

		if name == "" || ph == "" || !okTs || !okTid {
			t.Errorf("invalid trace event format: %v", ev)
		}

		_ = tsVal

		if name == "Frame" {
			hasFrame = true
		}
		if name == "Phase:Sync" {
			hasSync = true
		}
		if strings.HasPrefix(name, "JobSubmit:") {
			hasJobSubmit = true
			if tid != 1 {
				t.Errorf("expected JobSubmit to be on Tid 1, got %f", tid)
			}
		}
		if strings.HasPrefix(name, "JobRun:") {
			hasJobRun = true
			if tid <= 1 {
				t.Errorf("expected JobRun to be on worker Tid > 1, got %f", tid)
			}
		}
	}

	if !hasFrame {
		t.Errorf("expected trace to capture 'Frame' event")
	}
	if !hasSync {
		t.Errorf("expected trace to capture 'Phase:Sync' event")
	}
	if !hasJobSubmit {
		t.Errorf("expected trace to capture 'JobSubmit' event")
	}
	if !hasJobRun {
		t.Errorf("expected trace to capture 'JobRun' event")
	}
}
