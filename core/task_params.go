package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
)

var taskParamsMu sync.Mutex

// MarkTaskElementDone records an uploaded Telegram file element in the task's
// persisted parameters so a restart can skip it.
func MarkTaskElementDone(ctx context.Context, taskID, elementID string) error {
	if elementID == "" {
		return errors.New("empty task element ID")
	}
	taskParamsMu.Lock()
	defer taskParamsMu.Unlock()

	record, err := database.GetTaskRecord(ctx, taskID)
	if err != nil {
		return err
	}
	if record.Type != tasktype.TaskTypeTgfiles.String() {
		return errors.New("task is not a telegram file task")
	}
	if record.Params == "" {
		return errors.New("task has no persisted parameters")
	}
	var spec taskspec.TGFile
	if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
		return err
	}
	for _, id := range spec.Done {
		if id == elementID {
			return nil
		}
	}
	spec.Done = append(spec.Done, elementID)
	raw, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return database.UpdateTaskRecord(ctx, taskID, map[string]any{"params": string(raw)})
}
