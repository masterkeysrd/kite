# Task: Refactor Engine and Implement Scheduler

## Objective
Extract the worker pool and task queues out of the `engine.Engine` struct into an independent `terminal.Scheduler` implementation, making the Engine a pure rendering pipeline coordinator.

## Requirements
1. **Define `terminal.Scheduler`:**
   - In `terminal/terminal.go`, add `Scheduler()` to the `Terminal` interface.
   - Define the `Scheduler` interface:
     ```go
     type Scheduler interface {
         RunBackground(task func())
         QueueMicrotask(task func())
         QueueMacrotask(task func())
         DrainMicrotasks()
         DrainMacrotasks(budget int)
     }
     ```

2. **Implement `engine.Scheduler`:**
   - Create `engine/scheduler.go` (or similar).
   - Move the worker pool logic (`jobQueue`, `workers`, `macroQueue`, `microQueue`, mutexes) out of `engine.Engine` and into a new `engine.defaultScheduler` struct that implements `terminal.Scheduler`.
   - Update the worker goroutines to accept plain `func()` closures instead of `engine.Job`.

3. **Purge `engine.Job`:**
   - Delete `engine/job.go`.
   - Remove the `workerResults` channel and `Job Sync` phase from the engine frame loop.

4. **Update `engine.Engine`:**
   - Remove all queue and worker-related fields from `Engine`.
   - In the frame loop (`pipeline.go`), replace the old task draining logic with calls to `e.terminal.Scheduler().DrainMacrotasks(...)` and `e.terminal.Scheduler().DrainMicrotasks()`.

## Tests to Verify
- Run `go test ./engine/...` to ensure the new Scheduler logic correctly drains tasks and doesn't deadlock the frame loop.
- Fix any tests that previously relied on `engine.Job` or `engine.PostMacro()`.

## Documentation Updates
- None required.