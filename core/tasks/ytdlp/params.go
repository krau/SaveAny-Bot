package ytdlp

import (
	"encoding/json"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	return json.Marshal(taskspec.YTDLP{URLs: t.URLs, Flags: t.Flags})
}
