package aria2dl

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	return json.Marshal(taskspec.Aria2{GID: t.GID, URIs: t.URIs})
}
