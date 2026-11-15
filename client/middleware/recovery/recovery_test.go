package recovery

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gotd/td/tgerr"
)

func TestShouldRecoverErrors(t *testing.T) {
	r := &recovery{ctx: context.Background()}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "wrapped context canceled",
			err:  fmt.Errorf("engine forcibly closed: %w", context.Canceled),
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "wrapped context deadline exceeded",
			err:  fmt.Errorf("request timed out: %w", context.DeadlineExceeded),
			want: false,
		},
		{
			name: "telegram error",
			err:  tgerr.New(400, "BAD_REQUEST"),
			want: false,
		},
		{
			name: "network error",
			err:  errors.New("connection reset"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.shouldRecover(context.Background(), tt.err); got != tt.want {
				t.Fatalf("shouldRecover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRecoverStoppedContexts(t *testing.T) {
	t.Run("recovery context stopped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		r := &recovery{ctx: ctx}
		if r.shouldRecover(context.Background(), errors.New("connection reset")) {
			t.Fatal("shouldRecover() = true, want false")
		}
	})

	t.Run("request context stopped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		r := &recovery{ctx: context.Background()}
		if r.shouldRecover(ctx, errors.New("connection reset")) {
			t.Fatal("shouldRecover() = true, want false")
		}
	})
}
