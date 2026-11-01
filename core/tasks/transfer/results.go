package transfer

import (
	"path"

	"github.com/krau/SaveAny-Bot/core"
)

var _ core.ResultReporter = (*Task)(nil)

// ItemResults reports per-file outcomes so a transfer with ignored failures is
// recorded as partial instead of a full success.
func (t *Task) ItemResults() []core.ItemResult {
	t.processingMu.RLock()
	defer t.processingMu.RUnlock()
	results := make([]core.ItemResult, 0, len(t.elems))
	for i := range t.elems {
		elem := &t.elems[i]
		result := core.ItemResult{
			Index:       i + 1,
			Name:        elem.FileInfo.Name,
			Path:        path.Join(elem.TargetPath, elem.FileInfo.Name),
			StorageName: elem.TargetStorage.Name(),
			Size:        elem.FileInfo.Size,
			Status:      core.ItemStatusCompleted,
		}
		if err := t.failed[elem.ID]; err != nil {
			result.Status = core.ItemStatusFailed
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}
