package parsed

import (
	"encoding/json"
	"errors"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	if t.item == nil {
		return nil, errors.New("task has no parsed item")
	}
	return json.Marshal(taskspec.Parsed{Item: t.item})
}
