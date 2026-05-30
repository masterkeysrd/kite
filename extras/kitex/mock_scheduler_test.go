package kitex

import "context"

type mockScheduler struct {
	macrotasks []func()
}

func (m *mockScheduler) RunBackground(task func(ctx context.Context)) { go task(context.Background()) }
func (m *mockScheduler) QueueMicrotask(task func())                   { task() }
func (m *mockScheduler) QueueMacrotask(task func()) {
	m.macrotasks = append(m.macrotasks, task)
}

func (m *mockScheduler) flushMacrotasks() {
	for len(m.macrotasks) > 0 {
		tasks := m.macrotasks
		m.macrotasks = nil
		for _, t := range tasks {
			t()
		}
	}
}
