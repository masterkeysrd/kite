package kitex

import (
	"context"
	"sync"
)

type mockScheduler struct {
	mu         sync.Mutex
	macrotasks []func()
}

func (m *mockScheduler) RunBackground(task func(ctx context.Context)) { go task(context.Background()) }
func (m *mockScheduler) QueueMicrotask(task func())                   { task() }
func (m *mockScheduler) QueueMacrotask(task func()) {
	m.mu.Lock()
	m.macrotasks = append(m.macrotasks, task)
	m.mu.Unlock()
}

func (m *mockScheduler) flushMacrotasks() {
	for {
		m.mu.Lock()
		if len(m.macrotasks) == 0 {
			m.mu.Unlock()
			break
		}
		tasks := m.macrotasks
		m.macrotasks = nil
		m.mu.Unlock()
		for _, t := range tasks {
			t()
		}
	}
}
