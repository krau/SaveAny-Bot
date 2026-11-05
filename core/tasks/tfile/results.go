package tfile

import (
	"context"
	"errors"

	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
)

var _ core.ResultReporter = (*Task)(nil)

func (t *Task) ItemResults() []core.ItemResult {
	result := core.ItemResult{
		Index:       1,
		Name:        t.File.Name(),
		Path:        t.Path,
		StorageName: t.Storage.Name(),
		StorageID:   storage.ID(t.Storage),
		Size:        t.File.Size(),
		Status:      core.ItemStatusCompleted,
	}
	if t.execErr != nil {
		result.Status = core.ItemStatusFailed
		if errors.Is(t.execErr, context.Canceled) {
			result.Status = core.ItemStatusCancelled
		}
		result.Error = t.execErr.Error()
	}
	if messageFile, ok := t.File.(tfilepkg.TGFileMessage); ok && messageFile.Message() != nil {
		msg := messageFile.Message()
		result.SourceChatID = tgutil.ChatIdFromPeer(msg.GetPeerID())
		result.SourceMessageID = msg.GetID()
		result.MetadataJSON = t.metadataJSON
		result.MetadataVersion = t.metadataVersion
	}
	return []core.ItemResult{result}
}
