package localupload

import (
	"path"

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
	return []core.ItemResult{{
		Index:       1,
		Name:        t.Name,
		Path:        path.Join(t.Path, t.Name),
		StorageName: t.Storage.Name(),
		StorageID:   storage.ID(t.Storage),
		Status:      status,
		Error:       t.execErr,
		Size:        t.Size,
	}}
}
