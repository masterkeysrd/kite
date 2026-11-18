//go:build windows

package clipboard

import (
	"context"
	"encoding/base64"
	"os/exec"
	"strings"
)

type windowsBackend struct{}

func newPlatformBackend() platformBackend {
	return &windowsBackend{}
}

func (b *windowsBackend) listFormats(ctx context.Context) ([]string, error) {
	const ps = `
Add-Type -AssemblyName System.Windows.Forms
$out = @()
if ([System.Windows.Forms.Clipboard]::ContainsText()) { $out += "text/plain" }
if ([System.Windows.Forms.Clipboard]::ContainsImage()) { $out += "image/png" }
if ([System.Windows.Forms.Clipboard]::ContainsFileDropList()) { $out += "text/uri-list" }
$out -join "` + "`n" + `"
`
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-STA", "-Command", ps).Output()
	if err != nil {
		return nil, err
	}
	var formats []string
	for _, line := range strings.Split(string(out), "\n") {
		if f := strings.TrimSpace(line); f != "" {
			formats = append(formats, f)
		}
	}
	return dedupeStrings(formats), nil
}

func (b *windowsBackend) read(ctx context.Context, mime string) ([]byte, error) {
	switch mime {
	case "text/plain":
		const ps = `Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::GetText()`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-STA", "-Command", ps).Output()
		if err != nil {
			return nil, err
		}
		return []byte(strings.TrimRight(string(out), "\r\n")), nil
	case "image/png":
		const ps = `
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($null -ne $img) {
	$ms = New-Object System.IO.MemoryStream
	$img.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
	[Convert]::ToBase64String($ms.ToArray())
}
`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-STA", "-Command", ps).Output()
		if err != nil {
			return nil, err
		}
		b64 := strings.TrimSpace(string(out))
		if b64 == "" {
			return nil, nil
		}
		return base64.StdEncoding.DecodeString(b64)
	default:
		return nil, ErrUnsupported
	}
}

func (b *windowsBackend) write(ctx context.Context, mime string, data []byte) error {
	if mime != "text/plain" {
		return ErrUnsupported
	}
	const ps = `Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::SetText($env:KITE_CLIP_TEXT)`
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-STA", "-Command", ps)
	cmd.Env = append(cmd.Environ(), "KITE_CLIP_TEXT="+string(data))
	return cmd.Run()
}
