// Package resume rebuilds tasks that were queued when the process stopped.
// Task creation parameters are persisted with the task record, so a queued
// task can be re-enqueued without resubmitting the original request.
package resume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/client/bot"
	userclient "github.com/krau/SaveAny-Bot/client/user"
	"github.com/krau/SaveAny-Bot/common/utils/tgutil"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/aria2dl"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	"github.com/krau/SaveAny-Bot/core/tasks/directlinks"
	"github.com/krau/SaveAny-Bot/core/tasks/parsed"
	"github.com/krau/SaveAny-Bot/core/tasks/telegraph"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/core/tasks/transfer"
	"github.com/krau/SaveAny-Bot/core/tasks/ytdlp"
	"github.com/krau/SaveAny-Bot/database"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/krau/SaveAny-Bot/pkg/taskspec"
	tphclient "github.com/krau/SaveAny-Bot/pkg/telegraph"
	tfilepkg "github.com/krau/SaveAny-Bot/pkg/tfile"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/krau/mygotg/ext"
)

const retryInterval = 15 * time.Second

// ErrNeedsClient means the task needs a Telegram client that is not running
// yet. The record stays queued and is retried later.
var ErrNeedsClient = errors.New("telegram client not available")

var scanMu sync.Mutex

// Run resumes queued tasks and keeps retrying while some of them are waiting
// for a Telegram client.
func Run(ctx context.Context) {
	for {
		pending := runOnce(ctx)
		if pending == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryInterval):
		}
	}
}

func runOnce(ctx context.Context) (pending int) {
	scanMu.Lock()
	defer scanMu.Unlock()

	logger := log.FromContext(ctx).WithPrefix("resume")
	records, err := database.ListQueuedTaskRecords(ctx)
	if err != nil {
		logger.Errorf("Failed to list queued tasks: %v", err)
		return 1
	}
	for i := range records {
		taskID := records[i].TaskID
		if core.HasTask(taskID) {
			continue
		}
		record, err := database.GetTaskRecord(ctx, taskID)
		if err != nil || record.Status != "queued" {
			continue
		}
		task, taskCtx, err := rebuild(ctx, record)
		switch {
		case errors.Is(err, ErrNeedsClient):
			pending++
			logger.Debugf("Task %s waits for a telegram client", taskID)
			continue
		case err != nil:
			logger.Warnf("Cannot resume task %s: %v", taskID, err)
			if dbErr := database.UpdateTaskRecord(ctx, taskID, map[string]any{
				"status":      "failed",
				"error":       "cannot resume: " + err.Error(),
				"finished_at": time.Now(),
			}); dbErr != nil {
				logger.Errorf("Failed to mark task %s as failed: %v", taskID, dbErr)
			}
			continue
		}
		if err := core.EnqueueExisting(taskCtx, task); err != nil {
			logger.Errorf("Failed to enqueue resumed task %s: %v", taskID, err)
			pending++
			continue
		}
		// The task may have been cancelled while it was being rebuilt.
		if current, err := database.GetTaskRecord(ctx, taskID); err == nil && current.Status == "cancelled" {
			if err := core.CancelTask(ctx, taskID); err != nil {
				logger.Errorf("Failed to cancel resumed task %s: %v", taskID, err)
			}
			continue
		}
		logger.Infof("Resumed queued task %s", taskID)
	}
	return pending
}

func rebuild(ctx context.Context, record *database.TaskRecord) (core.Executable, context.Context, error) {
	if record.Params == "" {
		return nil, ctx, errors.New("missing task parameters")
	}
	switch tasktype.TaskType(record.Type) {
	case tasktype.TaskTypeDirectlinks:
		var spec taskspec.DirectLinks
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		stor, err := resolveStorage(ctx, record.StorageName, record.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		taskCtx, err := attachTelegramClient(ctx, stor)
		if err != nil {
			return nil, ctx, err
		}
		target := strings.TrimSuffix(record.Path, "/")
		if spec.Path != nil {
			target = *spec.Path
		}
		return directlinks.NewTask(record.TaskID, taskCtx, spec.URLs, stor, target, nil), taskCtx, nil
	case tasktype.TaskTypeYtdlp:
		var spec taskspec.YTDLP
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		stor, err := resolveStorage(ctx, record.StorageName, record.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		taskCtx, err := attachTelegramClient(ctx, stor)
		if err != nil {
			return nil, ctx, err
		}
		return ytdlp.NewTask(record.TaskID, taskCtx, spec.URLs, spec.Flags, stor, record.Path, nil), taskCtx, nil
	case tasktype.TaskTypeAria2:
		var spec taskspec.Aria2
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		stor, err := resolveStorage(ctx, record.StorageName, record.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		taskCtx, err := attachTelegramClient(ctx, stor)
		if err != nil {
			return nil, ctx, err
		}
		cfg := config.C().Aria2
		if !cfg.Enable {
			return nil, ctx, errors.New("aria2 is not enabled")
		}
		aria2Client, err := aria2.NewClient(cfg.Url, cfg.Secret)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to create aria2 client: %w", err)
		}
		gid := spec.GID
		if gid == "" {
			gid, err = aria2Client.AddURI(ctx, spec.URIs, nil)
			if err != nil {
				return nil, ctx, fmt.Errorf("failed to add aria2 task: %w", err)
			}
		} else if _, err := aria2Client.TellStatus(ctx, gid); err != nil {
			gid, err = aria2Client.AddURI(ctx, spec.URIs, nil)
			if err != nil {
				return nil, ctx, fmt.Errorf("failed to add aria2 task: %w", err)
			}
		}
		return aria2dl.NewTask(record.TaskID, taskCtx, gid, spec.URIs, aria2Client, stor, record.Path, nil), taskCtx, nil
	case tasktype.TaskTypeParseditem:
		var spec taskspec.Parsed
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		if spec.Item == nil {
			return nil, ctx, errors.New("missing parsed item")
		}
		stor, err := resolveStorage(ctx, record.StorageName, record.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		taskCtx, err := attachTelegramClient(ctx, stor)
		if err != nil {
			return nil, ctx, err
		}
		return parsed.NewTask(record.TaskID, taskCtx, stor, record.Path, spec.Item, nil), taskCtx, nil
	case tasktype.TaskTypeTphpics:
		var spec taskspec.Telegraph
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		if len(spec.Pics) == 0 {
			return nil, ctx, errors.New("missing telegraph images")
		}
		stor, err := resolveStorage(ctx, record.StorageName, record.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		taskCtx, err := attachTelegramClient(ctx, stor)
		if err != nil {
			return nil, ctx, err
		}
		return telegraph.NewTask(record.TaskID, taskCtx, spec.PhPath, spec.Pics, stor, record.Path, tphclient.NewClient(), nil), taskCtx, nil
	case tasktype.TaskTypeTransfer:
		var spec taskspec.Transfer
		if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
			return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
		}
		elems := make([]transfer.TaskElement, 0, len(spec.Items))
		storages := make([]storage.Storage, 0, len(spec.Items)*2)
		for _, item := range spec.Items {
			sourceStor, err := resolveStorage(ctx, item.SourceStorage, item.SourceStorageID)
			if err != nil {
				return nil, ctx, err
			}
			targetStor, err := resolveStorage(ctx, item.TargetStorage, item.TargetStorageID)
			if err != nil {
				return nil, ctx, err
			}
			elem := transfer.NewTaskElement(sourceStor, storagetypes.FileInfo{
				Name:    item.Name,
				Path:    item.SourcePath,
				Size:    item.Size,
				ModTime: item.ModTime,
			}, targetStor, item.TargetPath)
			elems = append(elems, *elem)
			storages = append(storages, sourceStor, targetStor)
		}
		if len(elems) == 0 {
			return nil, ctx, errors.New("missing transfer files")
		}
		taskCtx, err := attachTelegramClient(ctx, storages...)
		if err != nil {
			return nil, ctx, err
		}
		return transfer.NewTransferTask(record.TaskID, taskCtx, elems, nil, true), taskCtx, nil
	case tasktype.TaskTypeTgfiles:
		return rebuildTG(ctx, record)
	default:
		return nil, ctx, fmt.Errorf("unsupported task type %q", record.Type)
	}
}

func rebuildTG(ctx context.Context, record *database.TaskRecord) (core.Executable, context.Context, error) {
	var spec taskspec.TGFile
	if err := json.Unmarshal([]byte(record.Params), &spec); err != nil {
		return nil, ctx, fmt.Errorf("invalid parameters: %w", err)
	}
	if len(spec.Items) == 0 {
		return nil, ctx, errors.New("missing telegram file items")
	}
	candidates := make([]*ext.Context, 0, 2)
	for _, candidate := range []*ext.Context{bot.ExtContext(), userclient.GetCtx()} {
		if candidate != nil {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, ctx, ErrNeedsClient
	}
	elems := make([]batchtfile.TaskElement, 0, len(spec.Items))
	var taskExt *ext.Context
	for _, item := range spec.Items {
		stor, err := resolveStorage(ctx, item.Storage, item.StorageID)
		if err != nil {
			return nil, ctx, err
		}
		message, clientCtx, err := tgutil.GetMessageByIDFreshAny(candidates, item.ChatID, item.MessageID)
		if err != nil {
			if clientPending() {
				return nil, ctx, ErrNeedsClient
			}
			return nil, ctx, fmt.Errorf("failed to fetch source message %d: %w", item.MessageID, err)
		}
		if message.Media == nil {
			return nil, ctx, fmt.Errorf("source message %d has no media", item.MessageID)
		}
		file, err := tfilepkg.FromMediaMessage(message.Media, clientCtx.Raw, message)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to rebuild file %s: %w", item.Name, err)
		}
		file.SetName(item.Name)
		elem, err := batchtfile.NewTaskElement(stor, item.Path, file)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to create task element for %s: %w", item.Name, err)
		}
		elems = append(elems, *elem)
		if taskExt == nil {
			taskExt = clientCtx
		}
	}
	taskCtx := tgutil.ExtWithContext(ctx, taskExt)
	if spec.Overwrite {
		taskCtx = storage.WithOverwrite(taskCtx)
	}
	if len(elems) == 1 {
		task, err := tfile.NewTGFileTask(record.TaskID, taskCtx, elems[0].File, elems[0].Storage, elems[0].Path, nil)
		if err != nil {
			return nil, ctx, err
		}
		return task, taskCtx, nil
	}
	return batchtfile.NewBatchTGFileTask(record.TaskID, taskCtx, elems, nil, true), taskCtx, nil
}

// activeClient returns the first running Telegram client.
func activeClient() *ext.Context {
	if botCtx := bot.ExtContext(); botCtx != nil {
		return botCtx
	}
	return userclient.GetCtx()
}

// attachTelegramClient adds a client context when the task writes to Telegram.
// A Telegram destination without a running client must wait for one.
func attachTelegramClient(ctx context.Context, storages ...storage.Storage) (context.Context, error) {
	needsTelegram := false
	for _, stor := range storages {
		if stor != nil && stor.Type() == storenum.Telegram {
			needsTelegram = true
			break
		}
	}
	client := activeClient()
	if client == nil {
		if needsTelegram {
			return ctx, ErrNeedsClient
		}
		return ctx, nil
	}
	return tgutil.ExtWithContext(ctx, client), nil
}

// clientPending reports whether a configured Telegram client is still starting
// or not logged in, so a fetch failure should be retried instead of failing
// the task permanently.
func clientPending() bool {
	cfg := config.C().Telegram
	if cfg.Token != "" && bot.CurrentStatus().State != "running" {
		return true
	}
	if cfg.Userbot.Enable && userclient.CurrentStatus().State != "running" {
		return true
	}
	return false
}

func resolveStorage(ctx context.Context, name string, id uint) (storage.Storage, error) {
	if id != 0 {
		stor, err := storage.GetStorageByID(ctx, id)
		if err == nil {
			return stor, nil
		}
		if name == "" {
			return nil, err
		}
	}
	if name == "" {
		return nil, errors.New("storage is empty")
	}
	stor, ok := storage.GetStorage(name)
	if !ok {
		return nil, fmt.Errorf("storage %q is not available", name)
	}
	return stor, nil
}
