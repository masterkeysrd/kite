//go:build linux

package clipboard

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

type linuxBackend struct{}

func newPlatformBackend() platformBackend {
	return &linuxBackend{}
}

func isWayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

func (b *linuxBackend) listFormats(ctx context.Context) ([]string, error) {
	if isWayland() {
		if path, err := exec.LookPath("wl-paste"); err == nil {
			out, err := exec.CommandContext(ctx, path, "--list-types").Output()
			if err != nil {
				return nil, err
			}
			return parseLinuxTargets(string(out)), nil
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		out, err := exec.CommandContext(ctx, path, "-selection", "clipboard", "-t", "TARGETS", "-o").Output()
		if err != nil {
			return nil, err
		}
		return parseLinuxTargets(string(out)), nil
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return []string{"text/plain"}, nil
	}
	return nil, ErrUnsupported
}

func parseLinuxTargets(raw string) []string {
	var out []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "", "TARGETS", "MULTIPLE", "TIMESTAMP", "SAVE_TARGETS":
			continue
		case "UTF8_STRING", "STRING", "TEXT", "text/plain;charset=utf-8":
			out = append(out, "text/plain")
		default:
			if strings.Contains(line, "/") {
				out = append(out, line)
			}
		}
	}
	return dedupeStrings(out)
}

func (b *linuxBackend) read(ctx context.Context, mime string) ([]byte, error) {
	if isWayland() {
		if path, err := exec.LookPath("wl-paste"); err == nil {
			return exec.CommandContext(ctx, path, "--no-newline", "--type", mime).Output()
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		target := mime
		if mime == "text/plain" {
			target = "UTF8_STRING"
		}
		return exec.CommandContext(ctx, path, "-selection", "clipboard", "-t", target, "-o").Output()
	}
	if path, err := exec.LookPath("xsel"); err == nil && mime == "text/plain" {
		return exec.CommandContext(ctx, path, "--clipboard", "--output").Output()
	}
	return nil, ErrUnsupported
}

func (b *linuxBackend) write(ctx context.Context, mime string, data []byte) error {
	if isWayland() {
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.CommandContext(ctx, path, "--type", mime)
			cmd.Stdin = bytes.NewReader(data)
			return cmd.Run()
		}
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		target := mime
		if mime == "text/plain" {
			target = "UTF8_STRING"
		}
		cmd := exec.CommandContext(ctx, path, "-selection", "clipboard", "-t", target, "-i")
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	}
	if path, err := exec.LookPath("xsel"); err == nil && mime == "text/plain" {
		cmd := exec.CommandContext(ctx, path, "--clipboard", "--input")
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	}
	return ErrUnsupported
}
