package transfer

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	items := make([]taskspec.TransferItem, 0, len(t.elems))
	for i := range t.elems {
		elem := &t.elems[i]
		items = append(items, taskspec.TransferItem{
			SourceStorage:   elem.SourceStorage.Name(),
			SourceStorageID: storage.ID(elem.SourceStorage),
			SourcePath:      elem.SourcePath,
			TargetStorage:   elem.TargetStorage.Name(),
			TargetStorageID: storage.ID(elem.TargetStorage),
			TargetPath:      elem.TargetPath,
			Name:            elem.FileInfo.Name,
			Size:            elem.FileInfo.Size,
			ModTime:         elem.FileInfo.ModTime,
		})
	}
	return json.Marshal(taskspec.Transfer{Items: items})
}
