package trace

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTracer(t *testing.T) {
	tr := NewTracer()

	func() {
		defer tr.Begin("test-event")()
	}()

	var buf bytes.Buffer
	if err := tr.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var events []Event
	if err := json.Unmarshal(buf.Bytes(), &events); err != nil {
		t.Fatalf("Failed to unmarshal events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].Name != "test-event" || events[0].Ph != Begin {
		t.Errorf("Unexpected first event: %+v", events[0])
	}

	if events[1].Name != "test-event" || events[1].Ph != End {
		t.Errorf("Unexpected second event: %+v", events[1])
	}

	t.Run("Args", func(t *testing.T) {
		tr := NewTracer()
		args := map[string]any{"key": "value"}
		func() {
			defer tr.BeginWithArgs("args-event", args)()
		}()

		var buf bytes.Buffer
		tr.WriteJSON(&buf)
		var events []Event
		json.Unmarshal(buf.Bytes(), &events)

		if len(events) != 2 {
			t.Fatalf("Expected 2 events, got %d", len(events))
		}
		if events[0].Args["key"] != "value" {
			t.Errorf("Expected Arg key=value, got %v", events[0].Args["key"])
		}
		if events[1].Args["key"] != "value" {
			t.Errorf("Expected Arg key=value in End event, got %v", events[1].Args["key"])
		}
	})
}

func TestNilTracer(t *testing.T) {
	var tr *Tracer

	// Should not panic
	done := tr.Begin("test")
	done()

	var buf bytes.Buffer
	if err := tr.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("Expected empty buffer for nil tracer, got %q", buf.String())
	}
}
