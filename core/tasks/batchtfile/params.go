package batchtfile

import (
	"encoding/json"
	"errors"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	items := make([]taskspec.TGFileItem, 0, len(t.elems))
	for i := range t.elems {
		elem := &t.elems[i]
		messageFile, ok := elem.File.(tfile.TGFileMessage)
		if !ok || messageFile.Message() == nil {
			return nil, errors.New("task element has no source message")
		}
		message := messageFile.Message()
		items = append(items, taskspec.TGFileItem{
			ChatID:    tgutil.ChatIdFromPeer(message.GetPeerID()),
			MessageID: message.GetID(),
			Name:      elem.File.Name(),
			Storage:   elem.Storage.Name(),
			StorageID: storage.ID(elem.Storage),
			Path:      elem.Path,
		})
	}
	overwrite := false
	if t.ctx != nil {
		overwrite, _ = t.ctx.Value(ctxkey.OverwriteExisting).(bool)
	}
	return json.Marshal(taskspec.TGFile{Items: items, Overwrite: overwrite})
}
