package trace

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// EventType represents the phase of a trace event.
type EventType string

const (
	Begin EventType = "B"
	End   EventType = "E"
)

// Event represents a single Chrome Trace Event.
type Event struct {
	Name string         `json:"name"`
	Ph   EventType      `json:"ph"`
	Ts   int64          `json:"ts"` // Microseconds
	Pid  int            `json:"pid"`
	Tid  int            `json:"tid"`
	Args map[string]any `json:"args,omitempty"`
}

// Tracer captures trace events and formats them for Chrome Trace Viewer.
type Tracer struct {
	mu     sync.Mutex
	events []Event
	start  time.Time
}

// NewTracer creates a new Tracer.
func NewTracer() *Tracer {
	return &Tracer{
		start: time.Now(),
	}
}

// Begin starts a new synchronous trace event.
// It returns a function that, when called, records the end of the event.
//
// Usage: defer tracer.Begin("Layout")()
func (t *Tracer) Begin(name string) func() {
	if t == nil {
		return Noop()
	}

	t.mu.Lock()
	t.events = append(t.events, Event{
		Name: name,
		Ph:   Begin,
		Ts:   time.Since(t.start).Microseconds(),
		Pid:  1,
		Tid:  1,
	})
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		t.events = append(t.events, Event{
			Name: name,
			Ph:   End,
			Ts:   time.Since(t.start).Microseconds(),
			Pid:  1,
			Tid:  1,
		})
		t.mu.Unlock()
	}
}

// BeginThread starts a new synchronous trace event on a specific thread.
// It returns a function that, when called, records the end of the event.
func (t *Tracer) BeginThread(name string, tid int) func() {
	return t.BeginThreadWithArgs(name, tid, nil)
}

// BeginWithArgs starts a new synchronous trace event with optional arguments.
func (t *Tracer) BeginWithArgs(name string, args map[string]any) func() {
	return t.BeginThreadWithArgs(name, 1, args)
}

// BeginThreadWithArgs starts a new synchronous trace event on a specific thread with optional arguments.
func (t *Tracer) BeginThreadWithArgs(name string, tid int, args map[string]any) func() {
	if t == nil {
		return Noop()
	}

	t.mu.Lock()
	t.events = append(t.events, Event{
		Name: name,
		Ph:   Begin,
		Ts:   time.Since(t.start).Microseconds(),
		Pid:  1,
		Tid:  tid,
		Args: args,
	})
	t.mu.Unlock()

	return func() {
		t.mu.Lock()
		t.events = append(t.events, Event{
			Name: name,
			Ph:   End,
			Ts:   time.Since(t.start).Microseconds(),
			Pid:  1,
			Tid:  tid,
			Args: args,
		})
		t.mu.Unlock()
	}
}

// WriteJSON writes the captured events as a JSON array to w.
func (t *Tracer) WriteJSON(w io.Writer) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return json.NewEncoder(w).Encode(t.events)
}

// Noop returns a function that does nothing.
func Noop() func() { return func() {} }

func noop() {}
