package tfile

import (
	"encoding/json"
	"errors"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.ParamSource = (*Task)(nil)

func (t *Task) TaskParams() (json.RawMessage, error) {
	messageFile, ok := t.File.(tfilepkg.TGFileMessage)
	if !ok || messageFile.Message() == nil {
		return nil, errors.New("task has no source message")
	}
	message := messageFile.Message()
	overwrite := false
	if t.Ctx != nil {
		overwrite, _ = t.Ctx.Value(ctxkey.OverwriteExisting).(bool)
	}
	return json.Marshal(taskspec.TGFile{
		Overwrite: overwrite,
		Items: []taskspec.TGFileItem{{
			ChatID:    tgutil.ChatIdFromPeer(message.GetPeerID()),
			MessageID: message.GetID(),
			Name:      t.File.Name(),
			Storage:   t.Storage.Name(),
			StorageID: storage.ID(t.Storage),
			Path:      t.Path,
		}},
	})
}
