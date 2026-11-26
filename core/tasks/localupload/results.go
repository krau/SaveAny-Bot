package localupload

import (
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/storage"
)

func (t *Task) ItemResults() []core.ItemResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	status := t.result
	if status == "" {
		status = core.ItemStatusCompleted
	}
	resultPath := t.StoragePath()
	if status == core.ItemStatusCompleted && t.actualPath != "" {
		resultPath = t.actualPath
	}
	return []core.ItemResult{{
		Index:       1,
		Name:        t.Name,
		Path:        resultPath,
		StorageName: t.Storage.Name(),
		StorageID:   storage.ID(t.Storage),
		Status:      status,
		Error:       t.execErr,
		Size:        t.Size,
	}}
}
