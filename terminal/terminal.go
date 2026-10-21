package terminal

import (
	"context"

	"github.com/masterkeysrd/kite/promise"
)

// Terminal provides access to terminal-specific features like the
// clipboard and layout engine.
//
// This interface is implement by the engine and is accessible for the
// dom via el.Document().Terminal().
type Terminal interface {
	Clipboard() Clipboard
	Scheduler() Scheduler
	SetTitle(title string)
	Bell()
	SetProgressBar(state ProgressBarState, percentage int)
	SetProgramStatus(status *ProgramStatus)
}

type ProgressBarState uint8

const (
	ProgressBarHide ProgressBarState = iota
	ProgressBarNormal
	ProgressBarError
	ProgressBarIndeterminate
	ProgressBarPaused
)

// ProgramState is the state of a program reported with the Program Status
// Protocol (OSC 7501).
type ProgramState string

const (
	ProgramStateIdle    ProgramState = "idle"
	ProgramStateWorking ProgramState = "working"
	ProgramStateDone    ProgramState = "done"
	ProgramStateBlocked ProgramState = "blocked"
	ProgramStateError   ProgramState = "error"
	ProgramStateClear   ProgramState = "clear"
)

// ProgramStatusKind says what a blocked program is waiting for.
type ProgramStatusKind string

const (
	ProgramStatusKindPermission ProgramStatusKind = "permission"
	ProgramStatusKindQuestion   ProgramStatusKind = "question"
	ProgramStatusKindAuth       ProgramStatusKind = "auth"
)

// ProgramStatus is a report for the Program Status Protocol (OSC 7501).
type ProgramStatus struct {
	// State is the program state. Required.
	State ProgramState
	// ID addresses a record. Empty means the root record.
	ID string
	// App is a stable machine-readable program name.
	App string
	// Kind says what a blocked program waits for. Only used with ProgramStateBlocked.
	Kind ProgramStatusKind
	// Progress is a percentage between 0 and 100.
	Progress int
	// HasProgress reports whether Progress is set.
	HasProgress bool
	// Title is a short human-readable label for the record.
	Title string
	// Message is one human-readable line describing the record.
	Message string
}

type Scheduler interface {
	// RunBackground executes a task on a background worker pool.
	// The provided context is managed by the scheduler.
	RunBackground(task func(ctx context.Context))
	// QueueMicrotask schedules a task to run as a microtask on the main thread.
	QueueMicrotask(task func())
	// QueueMacrotask schedules a task to run as a macrotask on the main thread.
	QueueMacrotask(task func())
}

type Clipboard interface {
	// ReadText returns the current text content of the clipboard. If the clipboard is
	// empty or unavailable, it returns an empty string and an error.
	//
	// This call blocks until the clipboard content is available and fails
	// after 100ms if the clipboard is unavailable or access is denied.
	ReadText() *promise.Promise[string]
	WriteText(text string) *promise.Promise[struct{}]

	// ListFormats returns a list of MIME types representing the formats currently available in
	// the clipboard. If the clipboard is unavailable or access is denied, it returns an error.
	// ListFormats() ([]string, error)

	// Read returns the current content of the clipboard as a byte slice. If the clipboard is
	// empty or unavailable, it returns an empty byte slice and an error.
	//
	// This call blocks until the clipboard content is available and fails
	// after 100ms if the clipboard is unavailable or access is denied.
	//
	// E.g. Read("text/plain") returns the same content as ReadText() but as bytes.
	// Read("image/png") would return PNG-encoded image data if the clipboard contains an image.
	// Read("text/html") would return HTML content if the clipboard contains HTML.
	// Read(mime string) ([]byte, error)
	Read(mime string) *promise.Promise[[]byte]

	// Write writes the given data to the clipboard with the specified MIME type. If the clipboard is unavailable or access is denied, it returns an error.
	Write(mime string, data []byte) *promise.Promise[struct{}]
}
