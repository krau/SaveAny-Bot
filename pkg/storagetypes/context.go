package storagetypes

import (
	"context"
	"errors"
)

var ErrUploadSkipped = errors.New("storage skipped upload")

type requiredUploadContextKey struct{}

// WithRequiredUpload requests an explicit error if the backend skips the file.
func WithRequiredUpload(ctx context.Context) context.Context {
	return context.WithValue(ctx, requiredUploadContextKey{}, true)
}

func UploadRequired(ctx context.Context) bool {
	required, _ := ctx.Value(requiredUploadContextKey{}).(bool)
	return required
}

type publishGuardContextKey struct{}

// WithPublishGuard lets the queue serialize publication with explicit cancellation.
func WithPublishGuard(ctx context.Context, guard func(func() error) error) context.Context {
	return context.WithValue(ctx, publishGuardContextKey{}, guard)
}

func Publish(ctx context.Context, publish func() error) error {
	checked := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return publish()
	}
	if guard, ok := ctx.Value(publishGuardContextKey{}).(func(func() error) error); ok {
		return guard(checked)
	}
	return checked()
}

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

type messageMetadataContextKey struct{}

// WithMessageMetadata records normalized Telegram message metadata for storage
// backends that can use it without uploading a sidecar file.
func WithMessageMetadata(ctx context.Context, raw string) context.Context {
	if raw == "" {
		return ctx
	}
	return context.WithValue(ctx, messageMetadataContextKey{}, raw)
}

// MessageMetadataFromContext returns normalized message metadata supplied by
// the task layer.
func MessageMetadataFromContext(ctx context.Context) string {
	raw, _ := ctx.Value(messageMetadataContextKey{}).(string)
	return raw
}
