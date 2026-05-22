package inspector

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/engine"
)

func TestInspector(t *testing.T) {
	// 1. Create a dummy engine
	b := mock.New(80, 24)
	eng := engine.New(b, engine.Options{})

	// Add some content
	div := eng.Document().CreateElement("div", nil)
	div.SetID("test-div")
	eng.Mount(div)

	// 2. Find a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	actualAddr := l.Addr().String()
	l.Close()

	// 3. Attach inspector
	if _, err := Attach(eng, actualAddr, Options{NoOpen: true}); err != nil {
		t.Fatal(err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// 4. Try to connect to /stream
	resp, err := http.Get("http://" + actualAddr + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// 5. Trigger a frame
	eng.Frame()

	// 6. Read from stream
	// The first message is the initial state sent on connection
	// The second message should be the one from eng.Frame()
	// Or actually, broadcast() is called on frame done.

	reader := bufio.NewReader(resp.Body)

	// Skip first message if it's there
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	// SSE format: "data: ...\n\n"
	if strings.HasPrefix(line, "data: ") {
		if !strings.Contains(line, "test-div") {
			// Try reading more
			for i := 0; i < 5; i++ {
				line, _ = reader.ReadString('\n')
				if strings.Contains(line, "test-div") {
					break
				}
			}
			if !strings.Contains(line, "test-div") {
				t.Fatalf("expected JSON to contain 'test-div', got %q", line)
			}
		}
		// Verify new fields exist
		if !strings.Contains(line, "\"dom\":") {
			t.Errorf("expected JSON to contain 'dom' field")
		}
		if !strings.Contains(line, "\"fragments\":") {
			t.Errorf("expected JSON to contain 'fragments' field")
		}
	}

	// 7. Test /dump endpoint
	dumpResp, err := http.Get("http://" + actualAddr + "/dump")
	if err != nil {
		t.Fatal(err)
	}
	defer dumpResp.Body.Close()

	if dumpResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /dump, got %d", dumpResp.StatusCode)
	}
	if dumpResp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got %q", dumpResp.Header.Get("Content-Type"))
	}
}
