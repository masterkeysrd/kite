//go:build !darwin && !linux && !windows

package clipboard

import "context"

type otherBackend struct{}

func newPlatformBackend() platformBackend {
	return &otherBackend{}
}

func (b *otherBackend) listFormats(_ context.Context) ([]string, error) {
	return nil, ErrUnsupported
}

func (b *otherBackend) read(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrUnsupported
}

func (b *otherBackend) write(_ context.Context, _ string, _ []byte) error {
	return ErrUnsupported
}
