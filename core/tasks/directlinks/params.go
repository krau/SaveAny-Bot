package directlinks

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	urls := make([]string, 0, len(t.files))
	for _, file := range t.files {
		urls = append(urls, file.URL)
	}
	path := t.StorPath
	return json.Marshal(taskspec.DirectLinks{URLs: urls, Path: &path})
}
