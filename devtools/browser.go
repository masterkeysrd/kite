package devtools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// LocateBrowser finds the path to the first available Chromium-based browser
func LocateBrowser() string {
	var paths []string

	switch runtime.GOOS {
	case "darwin": // macOS
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		// Check both 64-bit and 32-bit Program Files
		paths = []string{
			os.Getenv("ProgramFiles") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Microsoft\\Edge\\Application\\msedge.exe",
			os.Getenv("ProgramFiles") + "\\BraveSoftware\\Brave-Browser\\Application\\brave.exe",
		}
	case "linux":
		// On Linux, browsers are usually in the PATH
		paths = []string{
			"google-chrome",
			"microsoft-edge",
			"brave-browser",
			"chromium-browser",
			"chromium",
		}
	}

	// Iterate through the paths and return the first one that exists
	for _, path := range paths {
		// For Windows/Mac, check if the file exists
		if _, err := os.Stat(path); err == nil {
			return path
		}
		// For Linux, check if the command exists in the system PATH
		if runtime.GOOS == "linux" {
			if p, err := exec.LookPath(path); err == nil {
				return p
			}
		}
	}

	return "" // No compatible browser found
}

// OpenFloatingInspector launches a Chromium-based browser in "app" mode if
// available. When a non-nil context is provided, the launched browser process
// is bound to the context and will be killed when the context is cancelled.
func OpenFloatingInspector(ctx context.Context, url string) error {
	browserPath := LocateBrowser()

	if browserPath != "" {
		// Launch a Chromium-based browser in app mode tied to ctx so it will
		// be terminated automatically when ctx is cancelled.
		cmd := exec.CommandContext(ctx, browserPath, "--app="+url)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start browser %q: %w", browserPath, err)
		}
		return nil
	}

	// Fallback: open with the system default opener (won't auto-close).
	// Caller should be aware that the fallback does not bind the process to ctx.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open URL fallback: %w", err)
	}
	return nil
}
