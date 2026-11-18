//go:build darwin

package clipboard

import (
	"slices"
	"testing"
)

func TestMapDarwinUTIs(t *testing.T) {
	t.Parallel()

	got := mapDarwinUTIs([]string{
		"public.utf8-plain-text",
		"com.compuserve.gif",
		"org.webmproject.webp",
		"public.tiff",
	})

	for _, want := range []string{"text/plain", "image/gif", "image/webp", "image/tiff", "image/png"} {
		if !slices.Contains(got, want) {
			t.Fatalf("mapDarwinUTIs() = %v, missing %q", got, want)
		}
	}
}
