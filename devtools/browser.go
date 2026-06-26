package devtools

import (
	"context"
	"fmt"
	kitelog "github.com/masterkeysrd/kite/log"
	"os"
	"os/exec"
	"runtime"
	"syscall"
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
		paths = []string{
			os.Getenv("ProgramFiles") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Microsoft\\Edge\\Application\\msedge.exe",
			os.Getenv("ProgramFiles") + "\\BraveSoftware\\Brave-Browser\\Application\\brave.exe",
		}
	case "linux":
		paths = []string{
			"google-chrome",
			"microsoft-edge",
			"brave-browser",
			"chromium-browser",
			"chromium",
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if runtime.GOOS == "linux" {
			if p, err := exec.LookPath(path); err == nil {
				return p
			}
		}
	}

	return ""
}

// OpenFloatingInspector launches a Chromium-based browser in "app" mode if
// available.
func OpenFloatingInspector(ctx context.Context, url string) error {
	kitelog.Info("devtools: OpenFloatingInspector called", "url", url)
	browserPath := LocateBrowser()

	if browserPath != "" {
		kitelog.Info("devtools: launching Chromium in app mode", "path", browserPath, "url", url)
		cmd := exec.Command(browserPath, "--app="+url)

		if runtime.GOOS != "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start browser %q: %w", browserPath, err)
		}

		go func() {
			<-ctx.Done()
			kitelog.Info("devtools: context cancelled, killing browser process")
			if runtime.GOOS == "windows" {
				_ = cmd.Process.Kill()
			} else {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		}()

		return nil
	}

	kitelog.Info("devtools: falling back to system default opener", "url", url)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
