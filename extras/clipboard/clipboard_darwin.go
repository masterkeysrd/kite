//go:build darwin

package clipboard

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os/exec"
	"strings"
)

type darwinBackend struct{}

func newPlatformBackend() platformBackend {
	return &darwinBackend{}
}

var utiToMIME = map[string]string{
	"public.utf8-plain-text": "text/plain",
	"NSStringPboardType":     "text/plain",
	"public.html":            "text/html",
	"Apple HTML pasteboard":  "text/html",
	"public.png":             "image/png",
	"public.jpeg":            "image/jpeg",
	"com.compuserve.gif":     "image/gif",
	"org.webmproject.webp":   "image/webp",
	"public.tiff":            "image/tiff",
	"public.file-url":        "text/uri-list",
}

var mimeToUTI = map[string]string{
	"text/plain":    "public.utf8-plain-text",
	"text/html":     "public.html",
	"image/png":     "public.png",
	"image/jpeg":    "public.jpeg",
	"image/gif":     "com.compuserve.gif",
	"image/webp":    "org.webmproject.webp",
	"image/tiff":    "public.tiff",
	"text/uri-list": "public.file-url",
}

func (b *darwinBackend) listFormats(ctx context.Context) ([]string, error) {
	const script = `
ObjC.import("AppKit");
const pb = $.NSPasteboard.generalPasteboard;
const types = pb.types;
const out = [];
if (types) {
	const n = types.count;
	for (let i = 0; i < n; i++) {
		out.push(ObjC.unwrap(types.objectAtIndex(i)));
	}
}
JSON.stringify(out);
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var utis []string
	if err := json.Unmarshal(bytes.TrimSpace(out), &utis); err != nil {
		return nil, err
	}
	return mapDarwinUTIs(utis), nil
}

func mapDarwinUTIs(utis []string) []string {
	var mimes []string
	hasTIFF := false
	hasPNG := false
	for _, u := range utis {
		if m, ok := utiToMIME[u]; ok {
			mimes = append(mimes, m)
			if m == "image/png" {
				hasPNG = true
			}
			if m == "image/tiff" {
				hasTIFF = true
			}
		} else if strings.Contains(u, "/") {
			mimes = append(mimes, u)
		}
	}
	// macOS often places only public.tiff for screenshots/Finder image copies;
	// expose image/png as well since read() can transcode TIFF -> PNG.
	if hasTIFF && !hasPNG {
		mimes = append(mimes, "image/png")
	}
	return dedupeStrings(mimes)
}

func (b *darwinBackend) read(ctx context.Context, mime string) ([]byte, error) {
	if mime == "text/plain" {
		cmd := exec.CommandContext(ctx, "pbpaste")
		return cmd.Output()
	}

	uti, ok := mimeToUTI[mime]
	if !ok {
		uti = mime
	}

	const script = `
ObjC.import("AppKit");
ObjC.import("Foundation");
const uti = $.NSProcessInfo.processInfo.environment.objectForKey("KITE_CLIP_UTI");
const mime = ObjC.unwrap($.NSProcessInfo.processInfo.environment.objectForKey("KITE_CLIP_MIME"));
const pb = $.NSPasteboard.generalPasteboard;
let data = pb.dataForType(uti);
if ((!data || data.isNil()) && mime === "image/png") {
	const tiff = pb.dataForType($("public.tiff"));
	if (tiff && !tiff.isNil()) {
		const rep = $.NSBitmapImageRep.imageRepWithData(tiff);
		if (rep && !rep.isNil()) {
			data = rep.representationUsingTypeProperties($.NSBitmapImageFileTypePNG, $({}));
		}
	}
}
if (!data || data.isNil()) {
	"";
} else {
	data.base64EncodedStringWithOptions(0).js;
}
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	cmd.Env = append(cmd.Environ(), "KITE_CLIP_UTI="+uti, "KITE_CLIP_MIME="+mime)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	b64 := strings.TrimSpace(string(out))
	if b64 == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(b64)
}

func (b *darwinBackend) write(ctx context.Context, mime string, data []byte) error {
	if mime == "text/plain" {
		cmd := exec.CommandContext(ctx, "pbcopy")
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	}

	uti, ok := mimeToUTI[mime]
	if !ok {
		uti = mime
	}

	const script = `
ObjC.import("AppKit");
ObjC.import("Foundation");
const uti = $.NSProcessInfo.processInfo.environment.objectForKey("KITE_CLIP_UTI");
const b64 = $.NSProcessInfo.processInfo.environment.objectForKey("KITE_CLIP_B64");
const data = $.NSData.alloc.initWithBase64EncodedStringOptions(b64, 0);
const pb = $.NSPasteboard.generalPasteboard;
pb.clearContents;
pb.setDataForType(data, uti);
`
	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-e", script)
	cmd.Env = append(cmd.Environ(),
		"KITE_CLIP_UTI="+uti,
		"KITE_CLIP_B64="+base64.StdEncoding.EncodeToString(data),
	)
	return cmd.Run()
}
