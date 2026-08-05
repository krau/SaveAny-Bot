package storagetypes

import "context"

type sourceCaptionContextKey struct{}

// WithSourceCaption records the source Telegram message caption for storage
// backends that can preserve it. Calling this function with an empty caption
// intentionally suppresses a backend-generated fallback caption.
func WithSourceCaption(ctx context.Context, caption string) context.Context {
	return context.WithValue(ctx, sourceCaptionContextKey{}, caption)
}

// SourceCaptionFromContext returns the source caption and whether the caller
// explicitly supplied one.
func SourceCaptionFromContext(ctx context.Context) (string, bool) {
	caption, ok := ctx.Value(sourceCaptionContextKey{}).(string)
	return caption, ok
}
