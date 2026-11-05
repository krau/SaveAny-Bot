package core

import "fmt"

// ItemStatus is the final state of one file inside a task.
type ItemStatus string

const (
	ItemStatusCompleted ItemStatus = "completed"
	ItemStatusFailed    ItemStatus = "failed"
	ItemStatusSkipped   ItemStatus = "skipped"
	ItemStatusCancelled ItemStatus = "cancelled"
)

// ItemResult describes the outcome of one file in a task. Tasks that handle
// multiple files implement ResultReporter so the outcome survives after the
// task leaves memory.
type ItemResult struct {
	Index           int
	Name            string
	Path            string
	StorageName     string
	StorageID       uint
	Status          ItemStatus
	Error           string
	Size            int64
	SourceChatID    int64
	SourceMessageID int
	MetadataJSON    string
	MetadataVersion int
}

// ResultReporter exposes per-file outcomes after execution.
type ResultReporter interface {
	ItemResults() []ItemResult
}

func collectItemResults(task Executable) []ItemResult {
	reporter, ok := task.(ResultReporter)
	if !ok {
		return nil
	}
	return reporter.ItemResults()
}

func countItems(items []ItemResult, status ItemStatus) int {
	count := 0
	for _, item := range items {
		if item.Status == status {
			count++
		}
	}
	return count
}

func partialError(items []ItemResult) string {
	failed := countItems(items, ItemStatusFailed)
	return fmt.Sprintf("%d of %d files failed", failed, len(items))
}
