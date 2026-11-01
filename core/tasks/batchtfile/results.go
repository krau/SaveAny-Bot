package batchtfile

import (
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/pkg/tfile"
)

var _ core.ResultReporter = (*Task)(nil)

// ItemResults reports the final state of every element. The worker persists
// these rows so failed files stay visible and retryable after the task ends.
func (t *Task) ItemResults() []core.ItemResult {
	states := t.Items()
	byID := make(map[string]TaskItemProgress, len(states))
	for _, state := range states {
		byID[state.ID] = state
	}

	results := make([]core.ItemResult, 0, len(t.elems))
	for i := range t.elems {
		elem := &t.elems[i]
		state, ok := byID[elem.ID]
		result := core.ItemResult{
			Index:       i + 1,
			Name:        elem.File.Name(),
			Path:        elem.Path,
			StorageName: elem.Storage.Name(),
			Size:        state.Size,
		}
		if !ok {
			result.Status = core.ItemStatusSkipped
		} else {
			result.Index = state.Index
			switch state.Phase {
			case ItemPhaseCompleted:
				result.Status = core.ItemStatusCompleted
			case ItemPhaseFailed:
				result.Status = core.ItemStatusFailed
				result.Error = failureSummary(state)
			case ItemPhaseStopped:
				result.Status = core.ItemStatusCancelled
			default:
				result.Status = core.ItemStatusSkipped
			}
		}
		if messageFile, ok := elem.File.(tfile.TGFileMessage); ok && messageFile.Message() != nil {
			msg := messageFile.Message()
			result.SourceChatID = tgutil.ChatIdFromPeer(msg.GetPeerID())
			result.SourceMessageID = msg.GetID()
		}
		results = append(results, result)
	}
	return results
}

func failureSummary(state TaskItemProgress) string {
	stage := failureStageToken(state.FailureStage)
	if state.Error == "" {
		return stage
	}
	return stage + ": " + state.Error
}

// failureStageToken returns a stable, non-localized stage name for storage.
func failureStageToken(stage FailureStage) string {
	switch stage {
	case FailureStageDownload:
		return "download"
	case FailureStageCache:
		return "local cache"
	case FailureStageUpload:
		return "upload"
	case FailureStageConfirm:
		return "remote confirmation"
	case FailureStageBatchUpload:
		return "batch upload"
	default:
		return "internal task"
	}
}
