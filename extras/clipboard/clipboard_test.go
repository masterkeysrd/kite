package clipboard

import (
	"context"
	"slices"
	"testing"
)

type fakePlatformBackend struct {
	items map[string][]byte
	err   error
}

func (f *fakePlatformBackend) listFormats(_ context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []string
	for k := range f.items {
		out = append(out, k)
	}
	return out, nil
}

func (f *fakePlatformBackend) read(_ context.Context, mime string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.items[mime], nil
}

func (f *fakePlatformBackend) write(_ context.Context, mime string, data []byte) error {
	if f.err != nil {
		return f.err
	}
	if f.items == nil {
		f.items = make(map[string][]byte)
	}
	f.items[mime] = append([]byte(nil), data...)
	return nil
}

func TestSystemClipboard_MultiMIME(t *testing.T) {
	t.Parallel()

	cb := &SystemClipboard{
		backend: &fakePlatformBackend{
			items: map[string][]byte{
				"text/plain": []byte("hello"),
				"image/png":  {0x89, 'P', 'N', 'G'},
			},
		},
	}

	formats, err := cb.ListFormats().Await(t.Context())
	if err != nil {
		t.Fatalf("ListFormats error: %v", err)
	}
	if !slices.Contains(formats, "text/plain") || !slices.Contains(formats, "image/png") {
		t.Fatalf("ListFormats = %v, want text/plain and image/png", formats)
	}

	txt, err := cb.ReadText().Await(t.Context())
	if err != nil || txt != "hello" {
		t.Fatalf("ReadText = (%q, %v), want (%q, nil)", txt, err, "hello")
	}

	img, err := cb.Read("image/png").Await(t.Context())
	if err != nil || string(img) != "\x89PNG" {
		t.Fatalf("Read(image/png) = (%q, %v), want PNG header", img, err)
	}

	if _, err := cb.WriteText("updated").Await(t.Context()); err != nil {
		t.Fatalf("WriteText error: %v", err)
	}
	txt, err = cb.ReadText().Await(t.Context())
	if err != nil || txt != "updated" {
		t.Fatalf("ReadText after WriteText = (%q, %v), want (%q, nil)", txt, err, "updated")
	}
}

func TestDedupeStrings(t *testing.T) {
	t.Parallel()

	got := dedupeStrings([]string{"", "text/plain", "image/png", "text/plain", ""})
	want := []string{"text/plain", "image/png"}
	if !slices.Equal(got, want) {
		t.Fatalf("dedupeStrings = %v, want %v", got, want)
	}
}
