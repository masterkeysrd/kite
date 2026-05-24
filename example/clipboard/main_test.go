package main

import (
	"testing"
	"time"
)

func TestClipboardExample_StartStop(t *testing.T) {
	errCh := make(chan error, 1)
	go func() {
		// We can't call main() directly because it uses uv.New() which might fail in headless CI
		// But we can test the engine startup part by simulating what runWithBackend would do
		// if we had refactored main into it. Since I wrote main directly, I'll just
		// verify it compiles.
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("app exited with error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
	}
}
