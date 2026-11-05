// Package taskstatus defines the persisted and public task status vocabulary.
package taskstatus

// Status is the lifecycle state of a task.
type Status string

const (
	Queued    Status = "queued"
	Running   Status = "running"
	Completed Status = "completed"
	Partial   Status = "partial"
	Failed    Status = "failed"
	Cancelled Status = "cancelled"
)

func (s Status) String() string { return string(s) }

// IsTerminal reports whether the status is a final state.
func (s Status) IsTerminal() bool {
	switch s {
	case Completed, Partial, Failed, Cancelled:
		return true
	default:
		return false
	}
}

func Values() []Status {
	return []Status{Queued, Running, Completed, Partial, Failed, Cancelled}
}
