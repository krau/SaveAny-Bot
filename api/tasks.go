package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
)

type mediaMetadataExtractor func(ctx context.Context, url string) (*MediaMetadataResponse, error)

func (s *server) createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "failed to decode request body: "+err.Error())
		return
	}
	if req.Type == "" {
		WriteError(w, http.StatusBadRequest, "invalid_request", "task type is required")
		return
	}
	if req.Storage == "" {
		WriteError(w, http.StatusBadRequest, "invalid_request", "storage is required")
		return
	}
	resp, err := s.factory.CreateTask(&req)
	if err != nil {
		if errors.Is(err, ErrClientUnavailable) {
			WriteError(w, http.StatusServiceUnavailable, "client_unavailable", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "task_creation_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, resp)
}

func (s *server) listTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	records, total, err := database.ListTaskRecords(r.Context(), database.TaskFilter{
		Status: query.Get("status"),
		Type:   query.Get("type"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	tasks := make([]TaskInfoResponse, 0, len(records))
	for i := range records {
		tasks = append(tasks, taskResponse(&records[i]))
	}
	WriteJSON(w, http.StatusOK, TasksListResponse{Tasks: tasks, Total: total})
}

func (s *server) getTask(w http.ResponseWriter, r *http.Request) {
	record, err := database.GetTaskRecord(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	WriteJSON(w, http.StatusOK, taskResponse(record))
}

func (s *server) cancelTask(w http.ResponseWriter, r *http.Request) {
	record, err := database.GetTaskRecord(r.Context(), r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	if isTerminal(record.Status) {
		WriteError(w, http.StatusConflict, "task_not_active", "task is not active")
		return
	}
	if task, ok := GetTask(record.TaskID); ok {
		task.UpdateStatus(TaskStatusCancelled)
	}
	if err := core.CancelTask(r.Context(), record.TaskID); err != nil {
		if task, ok := GetTask(record.TaskID); ok {
			task.UpdateStatus(TaskStatus(record.Status))
		}
		WriteError(w, http.StatusInternalServerError, "cancel_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusAccepted, map[string]any{"task_id": record.TaskID, "status": TaskStatusCancelled})
}

func (s *server) deleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	record, err := database.GetTaskRecord(r.Context(), taskID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	if !isTerminal(record.Status) {
		WriteError(w, http.StatusConflict, "task_active", "cancel the task before deleting it")
		return
	}
	if err := database.DeleteTaskRecord(r.Context(), taskID); err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	DeleteTask(taskID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) taskTypes(w http.ResponseWriter, r *http.Request) {
	types := []tasktype.TaskType{
		tasktype.TaskTypeDirectlinks,
		tasktype.TaskTypeYtdlp,
		tasktype.TaskTypeAria2,
		tasktype.TaskTypeParseditem,
		tasktype.TaskTypeTgfiles,
		tasktype.TaskTypeTphpics,
		tasktype.TaskTypeTransfer,
	}
	out := make([]map[string]string, 0, len(types))
	for _, t := range types {
		out = append(out, map[string]string{"type": t.String(), "label": t.String()})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"types": out})
}

func (s *server) mediaMetadata(w http.ResponseWriter, r *http.Request) {
	mediaURL := r.URL.Query().Get("url")
	if mediaURL == "" {
		WriteError(w, http.StatusBadRequest, "invalid_request", "url query parameter is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	resp, err := s.mediaMetadataExtractor(ctx, mediaURL)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "metadata_extraction_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func isTerminal(status string) bool {
	return status == string(TaskStatusCompleted) || status == string(TaskStatusFailed) || status == string(TaskStatusCancelled)
}

func taskResponse(record *database.TaskRecord) TaskInfoResponse {
	resp := TaskInfoResponse{
		TaskID:     record.TaskID,
		Type:       tasktype.TaskType(record.Type),
		Status:     TaskStatus(record.Status),
		Title:      record.Title,
		Storage:    record.Storage,
		Path:       record.Path,
		Source:     record.Source,
		Error:      record.Error,
		CreatedAt:  record.CreatedAt,
		StartedAt:  record.StartedAt,
		FinishedAt: record.FinishedAt,
		UpdatedAt:  record.UpdatedAt,
	}
	if info, ok := GetTask(record.TaskID); ok {
		status, total, downloaded, totalFiles, downloadedFiles, startedAt, errMsg, updatedAt := info.snapshot()
		resp.Status = status
		resp.Error = errMsg
		resp.UpdatedAt = updatedAt
		if resp.StartedAt == nil && !startedAt.IsZero() {
			resp.StartedAt = &startedAt
		}
		var percent, speedMBPS float64
		if total > 0 {
			percent = float64(downloaded) * 100 / float64(total)
		} else if totalFiles > 0 {
			percent = float64(downloadedFiles) * 100 / float64(totalFiles)
		}
		if !startedAt.IsZero() {
			if elapsed := time.Since(startedAt).Seconds(); elapsed > 0 {
				speedMBPS = float64(downloaded) / elapsed / (1024 * 1024)
			}
		}
		if total > 0 || totalFiles > 0 {
			resp.Progress = &TaskProgress{
				TotalBytes:      total,
				DownloadedBytes: downloaded,
				TotalFiles:      totalFiles,
				DownloadedFiles: downloadedFiles,
				Percent:         percent,
				SpeedMBPS:       speedMBPS,
			}
		}
	}
	if resp.Progress == nil && resp.Status == TaskStatusCompleted {
		resp.Progress = &TaskProgress{Percent: 100}
	}
	return resp
}
