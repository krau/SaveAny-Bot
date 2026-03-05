package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
	"github.com/krau/SaveAny-Bot/core/tasks/aria2dl"
	"github.com/krau/SaveAny-Bot/core/tasks/batchtfile"
	"github.com/krau/SaveAny-Bot/core/tasks/directlinks"
	"github.com/krau/SaveAny-Bot/core/tasks/parsed"
	tphtask "github.com/krau/SaveAny-Bot/core/tasks/telegraph"
	"github.com/krau/SaveAny-Bot/core/tasks/tfile"
	"github.com/krau/SaveAny-Bot/core/tasks/transfer"
	"github.com/krau/SaveAny-Bot/core/tasks/ytdlp"
	"github.com/krau/SaveAny-Bot/parsers/parsers"
	"github.com/krau/SaveAny-Bot/pkg/aria2"
	"github.com/krau/SaveAny-Bot/pkg/enums/tasktype"
	"github.com/krau/SaveAny-Bot/pkg/parser"
	"github.com/krau/SaveAny-Bot/pkg/telegraph"
	"github.com/krau/SaveAny-Bot/storage"
	"github.com/rs/xid"
)

// TaskFactory 任务工厂
type TaskFactory struct {
	ctx context.Context
}

// NewTaskFactory 创建任务工厂
func NewTaskFactory(ctx context.Context) *TaskFactory {
	return &TaskFactory{ctx: ctx}
}

// CreateTask 创建任务
func (f *TaskFactory) CreateTask(req *CreateTaskRequest) (*CreateTaskResponse, error) {
	// 验证存储
	stor, ok := storage.Storages[req.Storage]
	if !ok {
		return nil, fmt.Errorf("storage not found: %s", req.Storage)
	}

	taskID := xid.New().String()
	createdAt := time.Now()

	switch req.Type {
	case tasktype.TaskTypeDirectlinks:
		return f.createDirectLinksTask(taskID, createdAt, req, stor)
	case tasktype.TaskTypeYtdlp:
		return f.createYTDLPTask(taskID, createdAt, req, stor)
	case tasktype.TaskTypeAria2:
		return f.createAria2Task(taskID, createdAt, req, stor)
	case tasktype.TaskTypeParseditem:
		return f.createParsedTask(taskID, createdAt, req, stor)
	case tasktype.TaskTypeTgfiles:
		return f.createTGFilesTask(taskID, createdAt, req, stor)
	case tasktype.TaskTypeTphpics:
		return f.createTPHPicsTask(taskID, createdAt, req, stor)
	case tasktype.TaskTypeTransfer:
		return f.createTransferTask(taskID, createdAt, req)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", req.Type)
	}
}

// createDirectLinksTask 创建直链下载任务
func (f *TaskFactory) createDirectLinksTask(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params DirectLinksParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if len(params.URLs) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	task := directlinks.NewTask(taskID, f.ctx, params.URLs, stor, req.Path, nil)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeDirectlinks,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createYTDLPTask 创建 yt-dlp 任务
func (f *TaskFactory) createYTDLPTask(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params YTDLPParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if len(params.URLs) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	task := ytdlp.NewTask(taskID, f.ctx, params.URLs, params.Flags, stor, req.Path, nil)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeYtdlp,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createAria2Task 创建 Aria2 任务
func (f *TaskFactory) createAria2Task(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params Aria2Params
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if len(params.URLs) == 0 {
		return nil, fmt.Errorf("no URLs provided")
	}

	// 检查 Aria2 是否启用
	cfg := config.C().Aria2
	if !cfg.Enable {
		return nil, fmt.Errorf("aria2 is not enabled")
	}

	aria2Client, err := aria2.NewClient(cfg.Url, cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aria2 client: %w", err)
	}

	// 添加下载任务到 Aria2
	gid, err := aria2Client.AddURI(f.ctx, params.URLs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to add aria2 task: %w", err)
	}

	task := aria2dl.NewTask(taskID, f.ctx, gid, params.URLs, aria2Client, stor, req.Path, nil)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeAria2,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createParsedTask 创建解析任务
func (f *TaskFactory) createParsedTask(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params ParsedParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.URL == "" {
		return nil, fmt.Errorf("no URL provided")
	}

	// 查找合适的解析器
	var p parser.Parser
	for _, parserItem := range parsers.Get() {
		if parserItem.CanHandle(params.URL) {
			p = parserItem
			break
		}
	}

	if p == nil {
		return nil, fmt.Errorf("no parser found for URL: %s", params.URL)
	}

	// 解析 URL
	item, err := p.Parse(f.ctx, params.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	task := parsed.NewTask(taskID, f.ctx, stor, req.Path, item, nil)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeParseditem,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createTGFilesTask 创建 Telegram 文件下载任务
func (f *TaskFactory) createTGFilesTask(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params TGFilesParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if len(params.MessageLinks) == 0 {
		return nil, fmt.Errorf("no message links provided")
	}

	// 提取文件
	files, err := ExtractFilesFromLinks(f.ctx, params.MessageLinks)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in provided links")
	}

	if len(files) == 1 {
		// 单个文件任务
		tfileTask, err := tfile.NewTGFileTask(taskID, f.ctx, files[0], stor, req.Path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create tfile task: %w", err)
		}
		if err := core.AddTask(f.ctx, tfileTask); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
	} else {
		// 批量文件任务
		elems := make([]batchtfile.TaskElement, 0, len(files))
		for _, file := range files {
			elem, err := batchtfile.NewTaskElement(stor, req.Path, file)
			if err != nil {
				return nil, fmt.Errorf("failed to create task element: %w", err)
			}
			elems = append(elems, *elem)
		}

		task := batchtfile.NewBatchTGFileTask(taskID, f.ctx, elems, nil, true)
		if err := core.AddTask(f.ctx, task); err != nil {
			return nil, fmt.Errorf("failed to add task: %w", err)
		}
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeTgfiles,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createTPHPicsTask 创建 Telegraph 图片下载任务
func (f *TaskFactory) createTPHPicsTask(taskID string, createdAt time.Time, req *CreateTaskRequest, stor storage.Storage) (*CreateTaskResponse, error) {
	var params TPHPicsParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.TelegraphURL == "" {
		return nil, fmt.Errorf("no telegraph URL provided")
	}

	// 提取图片
	pics, phPath, err := ExtractTelegraphImages(f.ctx, params.TelegraphURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract telegraph images: %w", err)
	}

	if len(pics) == 0 {
		return nil, fmt.Errorf("no images found in telegraph page")
	}

	client := telegraph.NewClient()
	task := tphtask.NewTask(taskID, f.ctx, phPath, pics, stor, req.Path, client, nil)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeTphpics,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}

// createTransferTask 创建存储间传输任务
func (f *TaskFactory) createTransferTask(taskID string, createdAt time.Time, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	var params TransferParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 验证源存储和目标存储
	sourceStor, ok := storage.Storages[params.SourceStorage]
	if !ok {
		return nil, fmt.Errorf("source storage not found: %s", params.SourceStorage)
	}

	targetStor, ok := storage.Storages[params.TargetStorage]
	if !ok {
		return nil, fmt.Errorf("target storage not found: %s", params.TargetStorage)
	}

	// 检查源存储是否可读
	sourceReadable, ok := sourceStor.(storage.StorageReadable)
	if !ok {
		return nil, fmt.Errorf("source storage does not support reading: %s", params.SourceStorage)
	}

	// 检查源存储是否可列
	sourceListable, ok := sourceStor.(storage.StorageListable)
	if !ok {
		return nil, fmt.Errorf("source storage does not support listing: %s", params.SourceStorage)
	}

	// 列出源文件
	files, err := sourceListable.ListFiles(f.ctx, params.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list source files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found at source path: %s", params.SourcePath)
	}

	// 创建传输元素
	elems := make([]transfer.TaskElement, 0, len(files))
	for _, file := range files {
		elem := transfer.NewTaskElement(sourceReadable, file, targetStor, params.TargetPath)
		elems = append(elems, *elem)
	}

	task := transfer.NewTransferTask(taskID, f.ctx, elems, nil, true)

	if err := core.AddTask(f.ctx, task); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return &CreateTaskResponse{
		TaskID:    taskID,
		Type:      tasktype.TaskTypeTransfer,
		Status:    TaskStatusQueued,
		CreatedAt: createdAt,
	}, nil
}
