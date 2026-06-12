package promise

import (
	"context"
	"testing"
)

func TestWrap(t *testing.T) {
	SetScheduler(nil)

	syncFn := func(ctx context.Context) (string, error) {
		return "wrapped", nil
	}

	asyncFn := Wrap(syncFn)
	p := asyncFn(context.Background())

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val != "wrapped" {
		t.Errorf("expected 'wrapped', got %q", val)
	}
}

func TestWrapWithProps(t *testing.T) {
	SetScheduler(nil)

	type Props struct {
		ID int
	}

	syncFn := func(ctx context.Context, p Props) (int, error) {
		return p.ID * 2, nil
	}

	asyncFn := WrapWithProps(syncFn)
	p := asyncFn(context.Background(), Props{ID: 21})

	val, err := p.Await(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}
