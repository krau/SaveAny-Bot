package api

import (
	"net/http"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/fnamest"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/rule"
	"github.com/krau/SaveAny-Bot/pkg/tcbdata"
)

type metaResponse struct {
	APIVersion         string   `json:"api_version"`
	Version            string   `json:"version"`
	SchemaVersion      int      `json:"schema_version"`
	ChatIDFormat       string   `json:"chat_id_format"`
	TaskTypes          []string `json:"task_types"`
	TaskStatuses       []string `json:"task_statuses"`
	TaskItemStatuses   []string `json:"task_item_statuses"`
	StorageTypes       []string `json:"storage_types"`
	RuleTypes          []string `json:"rule_types"`
	FilenameStrategies []string `json:"filename_strategies"`
	ConflictStrategies []string `json:"conflict_strategies"`
}

func (s *server) meta(w http.ResponseWriter, _ *http.Request) {
	taskTypes := tasktype.TaskTypeValues()
	taskTypeNames := make([]string, 0, len(taskTypes))
	for _, value := range taskTypes {
		taskTypeNames = append(taskTypeNames, value.String())
	}
	ruleTypes := rule.Values()
	ruleTypeNames := make([]string, 0, len(ruleTypes))
	for _, value := range ruleTypes {
		ruleTypeNames = append(ruleTypeNames, value.String())
	}
	WriteJSON(w, http.StatusOK, metaResponse{
		APIVersion:    "v1",
		Version:       config.Version,
		SchemaVersion: database.CurrentSchemaVersion,
		ChatIDFormat:  "bot_api",
		TaskTypes:     taskTypeNames,
		TaskStatuses: []string{
			string(TaskStatusQueued),
			string(TaskStatusRunning),
			string(TaskStatusCompleted),
			string(TaskStatusPartial),
			string(TaskStatusFailed),
			string(TaskStatusCancelled),
		},
		TaskItemStatuses: []string{
			string(core.ItemStatusCompleted),
			string(core.ItemStatusFailed),
			string(core.ItemStatusSkipped),
			string(core.ItemStatusCancelled),
		},
		StorageTypes:       storenum.StorageTypeNames(),
		RuleTypes:          ruleTypeNames,
		FilenameStrategies: fnamest.FnameSTNames(),
		ConflictStrategies: tcbdata.ConflictStrategyValues(),
	})
}
