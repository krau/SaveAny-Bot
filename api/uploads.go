package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/upload"
	"github.com/krau/SaveAny-Bot/common/utils/fsutil"
	"github.com/rs/xid"
)

const bytesPerMB = int64(1 << 20)

var errIncompleteUpload = errors.New("upload body ended before Content-Length")

func (s *server) createUpload(w http.ResponseWriter, r *http.Request) {
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Time{}); err != nil {
		log.Warnf("Failed to clear upload read deadline: %v", err)
	}
	if err := controller.SetWriteDeadline(time.Time{}); err != nil {
		log.Warnf("Failed to clear upload write deadline: %v", err)
	}

	storageID, err := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("storage_id")), 10, 64)
	if err != nil || storageID == 0 {
		WriteError(w, http.StatusBadRequest, "invalid_request", "storage_id is required")
		return
	}
	rawName := strings.TrimSpace(r.URL.Query().Get("filename"))
	if rawName == "" {
		WriteError(w, http.StatusBadRequest, "invalid_request", "filename is required")
		return
	}
	filename := fsutil.SafeBaseName(rawName)
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath != "" {
		targetPath = path.Clean(strings.ReplaceAll(targetPath, "\\", "/"))
		if targetPath == "." {
			targetPath = ""
		}
		if path.IsAbs(targetPath) || targetPath == ".." || strings.HasPrefix(targetPath, "../") {
			WriteError(w, http.StatusBadRequest, "invalid_request", "path must stay inside the target storage")
			return
		}
	}
	overwrite := false
	if raw := strings.TrimSpace(r.URL.Query().Get("overwrite")); raw != "" {
		overwrite, err = strconv.ParseBool(raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid_request", "overwrite must be a boolean")
			return
		}
	}

	maxSizeMB := s.deps.Settings.Config().Web.MaxUploadSizeMB
	if maxSizeMB < 1 || maxSizeMB > math.MaxInt64/bytesPerMB {
		WriteError(w, http.StatusInternalServerError, "invalid_upload_limit", "invalid upload size limit")
		return
	}
	maxBytes := maxSizeMB * bytesPerMB
	if r.ContentLength > maxBytes {
		WriteError(w, http.StatusRequestEntityTooLarge, "upload_too_large", fmt.Sprintf("upload exceeds %d MB", maxSizeMB))
		return
	}

	target, err := s.factory.ResolveUploadTarget(uint(storageID))
	if err != nil {
		if errors.Is(err, ErrClientUnavailable) {
			WriteError(w, http.StatusServiceUnavailable, "client_unavailable", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "task_creation_failed", err.Error())
		return
	}

	taskID := xid.New().String()
	createdAt := time.Now()
	stagingDir, err := upload.StagingDir(taskID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	payloadPath, err := upload.StagingPayloadPath(taskID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := os.MkdirAll(stagingDir, 0o700); err != nil {
		writeUploadStorageError(w, err)
		return
	}
	keepStaging := false
	defer func() {
		if !keepStaging {
			if err := upload.RemoveStaging(taskID); err != nil {
				log.Warnf("Failed to clean staged upload %s: %v", taskID, err)
			}
		}
	}()

	size, err := receiveUpload(w, r, payloadPath, maxBytes)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			WriteError(w, http.StatusRequestEntityTooLarge, "upload_too_large", fmt.Sprintf("upload exceeds %d MB", maxSizeMB))
		case errors.Is(err, errIncompleteUpload):
			WriteError(w, http.StatusBadRequest, "incomplete_upload", err.Error())
		case errors.Is(err, syscall.ENOSPC):
			WriteError(w, http.StatusInsufficientStorage, "insufficient_storage", "not enough disk space for upload")
		default:
			log.Errorf("Failed to receive web upload: %v", err)
			WriteError(w, http.StatusInternalServerError, "upload_failed", "failed to store upload")
		}
		return
	}

	resp, err := s.factory.CreateUploadTask(
		taskID,
		createdAt,
		target,
		filename,
		targetPath,
		size,
		overwrite,
		strings.TrimSpace(r.Header.Get("X-Webhook-URL")),
	)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "task_creation_failed", err.Error())
		return
	}
	keepStaging = true
	WriteJSON(w, http.StatusCreated, resp)
}

func receiveUpload(w http.ResponseWriter, r *http.Request, payloadPath string, maxBytes int64) (int64, error) {
	partPath := payloadPath + ".part"
	file, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, err
	}
	closeWithError := func() error {
		if err := file.Sync(); err != nil {
			return errors.Join(err, file.Close())
		}
		return file.Close()
	}

	size, copyErr := io.Copy(file, http.MaxBytesReader(w, r.Body, maxBytes))
	closeErr := closeWithError()
	if copyErr != nil {
		if errors.Is(copyErr, io.ErrUnexpectedEOF) {
			copyErr = errIncompleteUpload
		}
		return 0, errors.Join(copyErr, closeErr)
	}
	if closeErr != nil {
		return 0, closeErr
	}
	if (size == 0 && r.ContentLength > 0) || (r.ContentLength >= 0 && size != r.ContentLength) {
		return 0, fmt.Errorf("%w: received %d of %d bytes", errIncompleteUpload, size, r.ContentLength)
	}
	if err := os.Rename(partPath, payloadPath); err != nil {
		return 0, err
	}
	return size, nil
}

func writeUploadStorageError(w http.ResponseWriter, err error) {
	if errors.Is(err, syscall.ENOSPC) {
		WriteError(w, http.StatusInsufficientStorage, "insufficient_storage", "not enough disk space for upload")
		return
	}
	log.Errorf("Failed to prepare web upload staging: %v", err)
	WriteError(w, http.StatusInternalServerError, "upload_failed", "failed to prepare upload storage")
}
