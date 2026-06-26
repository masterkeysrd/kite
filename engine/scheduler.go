package engine

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	kitelog "github.com/masterkeysrd/kite/log"
	"github.com/masterkeysrd/kite/terminal"
)

type defaultScheduler struct {
	taskMu     sync.Mutex
	macroQueue []func()
	microQueue []func()

	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWG     sync.WaitGroup
	workerSem    chan struct{}
	jobQueue     chan func(ctx context.Context)

	clock             Clock
	macroTaskDuration time.Duration
	onRequestFrame    func()

	// Profiler integration
	onJobSubmit func(name string) func()
	onJobRun    func(name string, workerID int) func()
}

var _ terminal.Scheduler = (*defaultScheduler)(nil)

func newDefaultScheduler(numWorkers int, clk Clock, macroTaskDuration time.Duration, onRequestFrame func()) *defaultScheduler {
	workerCtx, workerCancel := context.WithCancel(context.Background())
	s := &defaultScheduler{
		workerCtx:         workerCtx,
		workerCancel:      workerCancel,
		workerSem:         make(chan struct{}, numWorkers),
		jobQueue:          make(chan func(ctx context.Context), numWorkers*4),
		clock:             clk,
		macroTaskDuration: macroTaskDuration,
		onRequestFrame:    onRequestFrame,
	}

	for i := range numWorkers {
		s.workerWG.Add(1)
		go func(id int) {
			s.runWorker(workerCtx, id)
		}(i + 1)
	}

	return s
}

func (s *defaultScheduler) RunBackground(task func(ctx context.Context)) {
	var endSubmit func()
	if s.onJobSubmit != nil {
		endSubmit = s.onJobSubmit("anonymous")
	}

	s.jobQueue <- func(ctx context.Context) {
		if endSubmit != nil {
			endSubmit()
		}
		task(ctx)
	}
}

func (s *defaultScheduler) QueueMicrotask(task func()) {
	s.taskMu.Lock()
	s.microQueue = append(s.microQueue, task)
	s.taskMu.Unlock()
	if s.onRequestFrame != nil {
		s.onRequestFrame()
	}
}

func (s *defaultScheduler) QueueMacrotask(task func()) {
	s.taskMu.Lock()
	s.macroQueue = append(s.macroQueue, task)
	s.taskMu.Unlock()
	if s.onRequestFrame != nil {
		s.onRequestFrame()
	}
}

func (s *defaultScheduler) drainMicrotasks() {
	for {
		s.taskMu.Lock()
		if len(s.microQueue) == 0 {
			s.taskMu.Unlock()
			break
		}
		task := s.microQueue[0]
		s.microQueue[0] = nil // Avoid memory leak by clearing reference
		s.microQueue = s.microQueue[1:]
		s.taskMu.Unlock()

		task()
	}
}

func (s *defaultScheduler) drainMacrotasks(budget int) {
	deadline := s.clock.Now().Add(s.macroTaskDuration)
	drained := 0
	for {
		s.taskMu.Lock()
		if len(s.macroQueue) == 0 || drained >= budget || s.clock.Now().After(deadline) {
			s.taskMu.Unlock()
			break
		}
		task := s.macroQueue[0]
		s.macroQueue[0] = nil // Avoid memory leak by clearing reference
		s.macroQueue = s.macroQueue[1:]
		s.taskMu.Unlock()

		task()
		drained++
		s.drainMicrotasks()
	}
}

func (s *defaultScheduler) hasPendingTasks() bool {
	s.taskMu.Lock()
	defer s.taskMu.Unlock()
	return len(s.microQueue) > 0 || len(s.macroQueue) > 0
}

func (s *defaultScheduler) stop() {
	s.workerCancel()
	s.workerWG.Wait()
}

func (s *defaultScheduler) runWorker(ctx context.Context, workerID int) {
	defer s.workerWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.jobQueue:
			s.executeTask(ctx, task, workerID)
			task = nil
		}
	}
}

func (s *defaultScheduler) executeTask(ctx context.Context, task func(ctx context.Context), workerID int) {
	var endRun func()
	if s.onJobRun != nil {
		endRun = s.onJobRun("anonymous", workerID)
	}

	defer func() {
		if v := recover(); v != nil {
			stack := string(debug.Stack())
			kitelog.Error("scheduler: panic in background task",
				slog.Any("panic", v),
				slog.String("stack", stack),
			)
		}
		if endRun != nil {
			endRun()
		}
	}()

	task(ctx)
}
