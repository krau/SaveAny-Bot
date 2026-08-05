package storagetypes

import (
	"context"
	"testing"
)

func TestSourceCaptionContext(t *testing.T) {
	if _, ok := SourceCaptionFromContext(context.Background()); ok {
		t.Fatal("caption unexpectedly present on empty context")
	}

	tests := []string{"original caption", ""}
	for _, want := range tests {
		ctx := WithSourceCaption(context.Background(), want)
		got, ok := SourceCaptionFromContext(ctx)
		if !ok {
			t.Fatalf("caption %q was not marked as present", want)
		}
		if got != want {
			t.Fatalf("caption = %q, want %q", got, want)
		}
	}
}
