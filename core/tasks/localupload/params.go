package localupload

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	"github.com/krau/SaveAny-Bot/storage"
)

func (t *Task) TaskParams() (json.RawMessage, error) {
	return json.Marshal(taskspec.Upload{
		Name:        t.Name,
		Size:        t.Size,
		Storage:     t.Storage.Name(),
		StorageID:   storage.ID(t.Storage),
		Path:        t.Path,
		Overwrite:   t.Overwrite,
		StagingRoot: t.stagingRoot,
	})
}
