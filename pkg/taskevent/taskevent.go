// Package taskevent provides a decoupled, context-scoped event bus for task
// lifecycle progress. Producers (task implementations) emit events via Emit;
// consumers (e.g. the API progress store, the Telegram message editor) register
// as Sinks and are injected through context. This keeps the task layer free of
// any concrete progress-display dependency, so new task types gain progress
// reporting for free and new observers can be added without touching tasks.
package taskevent

import "context"

// Phase marks a stage in a task's lifecycle.
type Phase int

const (
	PhaseStart Phase = iota
	PhaseProgress
	PhaseDone
)

func (p Phase) String() string {
	switch p {
	case PhaseStart:
		return "start"
	case PhaseProgress:
		return "progress"
	case PhaseDone:
		return "done"
	default:
		return "unknown"
	}
}

// Event describes a single progress observation for a task. Byte fields are
// populated by byte-stream tasks; file-count fields by count-based tasks. A
// task may fill whichever subset it has; observers ignore zero values.
type Event struct {
	TaskID          string
	Phase           Phase
	TotalBytes      int64
	DownloadedBytes int64
	TotalFiles      int
	DownloadedFiles int
	Err             error
}

// Sink receives task events. Implementations must be safe for concurrent use.
type Sink interface {
	Emit(Event)
}

// SinkFunc is a function adapter for Sink.
type SinkFunc func(Event)

func (f SinkFunc) Emit(e Event) { f(e) }

type sinkKey struct{}

// WithSink returns a ctx carrying the given sinks. Multiple sinks can be passed
// and all will receive every emitted event. Sinks already present in ctx are
// preserved.
func WithSink(ctx context.Context, sinks ...Sink) context.Context {
	if len(sinks) == 0 {
		return ctx
	}
	var existing []Sink
	if v, ok := ctx.Value(sinkKey{}).([]Sink); ok {
		existing = v
	}
	merged := make([]Sink, 0, len(existing)+len(sinks))
	merged = append(merged, existing...)
	merged = append(merged, sinks...)
	return context.WithValue(ctx, sinkKey{}, merged)
}

// Emit broadcasts an event to all sinks carried by ctx. It is a no-op when no
// sink is attached, so producers can call it unconditionally.
func Emit(ctx context.Context, e Event) {
	if ctx == nil {
		return
	}
	sinks, ok := ctx.Value(sinkKey{}).([]Sink)
	if !ok {
		return
	}
	for _, s := range sinks {
		s.Emit(e)
	}
}
