package rclone

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
	"github.com/rs/xid"
)

type Rclone struct {
	config config.RcloneStorageConfig
	logger *log.Logger
}

func (r *Rclone) Init(ctx context.Context, cfg config.StorageConfig) error {
	rcloneConfig, ok := cfg.(*config.RcloneStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast rclone config")
	}
	if err := rcloneConfig.Validate(); err != nil {
		return err
	}
	r.config = *rcloneConfig
	r.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("rclone[%s]", r.config.Name))

	// 检查 rclone 是否安装
	if _, err := exec.LookPath("rclone"); err != nil {
		return ErrRcloneNotFound
	}

	args := r.buildBaseArgs()
	args = append(args, "listremotes")
	cmd := exec.CommandContext(ctx, "rclone", args...)
	output, err := cmd.Output()
	if err != nil {
		r.logger.Errorf("Failed to list remotes: %v", err)
		return fmt.Errorf("failed to verify rclone: %w", err)
	}

	remoteName := strings.TrimSuffix(r.config.Remote, ":")
	if !strings.HasSuffix(r.config.Remote, ":") {
		remoteName = r.config.Remote
	}

	found := false
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimSuffix(line, ":")
		if line == remoteName {
			found = true
			break
		}
	}

	if !found {
		r.logger.Errorf("Remote %s not found in rclone config", r.config.Remote)
		return ErrRemoteNotFound
	}

	r.logger.Infof("Initialized rclone storage with remote: %s", r.config.Remote)
	return nil
}

func (r *Rclone) Type() storenum.StorageType {
	return storenum.Rclone
}

func (r *Rclone) Name() string {
	return r.config.Name
}

func (r *Rclone) buildBaseArgs() []string {
	var args []string
	if r.config.ConfigPath != "" {
		args = append(args, "--config", r.config.ConfigPath)
	}
	args = append(args, r.config.Flags...)
	return args
}

func (r *Rclone) getRemotePath(storagePath string) string {
	remote := r.config.Remote
	if !strings.HasSuffix(remote, ":") {
		remote += ":"
	}
	basePath := strings.TrimPrefix(r.config.BasePath, "/")
	fullPath := path.Join(basePath, storagePath)
	return remote + fullPath
}

func (r *Rclone) Save(ctx context.Context, reader io.Reader, storagePath string) error {
	r.logger.Infof("Saving file to %s", storagePath)

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; r.Exists(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
		if i > 100 {
			r.logger.Errorf("Too many attempts to find a unique filename for %s", storagePath)
			candidate = fmt.Sprintf("%s_%s%s", base, xid.New().String(), ext)
			break
		}
	}

	remotePath := r.getRemotePath(candidate)
	r.logger.Debugf("Remote path: %s", remotePath)

	// Use rclone rcat to read from stdin and upload
	args := r.buildBaseArgs()
	args = append(args, "rcat", remotePath)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdin = reader

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		r.logger.Errorf("Failed to save file: %v, stderr: %s", err, stderr.String())
		return fmt.Errorf("%w: %s", ErrFailedToSaveFile, stderr.String())
	}

	r.logger.Infof("Successfully saved file to %s", candidate)
	return nil
}

func (r *Rclone) Exists(ctx context.Context, storagePath string) bool {
	remotePath := r.getRemotePath(storagePath)

	args := r.buildBaseArgs()
	args = append(args, "lsf", remotePath)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	err := cmd.Run()
	return err == nil
}

// lsjsonItem represents a single entry in the output of `rclone lsjson`
type lsjsonItem struct {
	Path     string `json:"Path"`
	Name     string `json:"Name"`
	Size     int64  `json:"Size"`
	MimeType string `json:"MimeType"`
	ModTime  string `json:"ModTime"`
	IsDir    bool   `json:"IsDir"`
}

// ListFiles implements storage.StorageListable
func (r *Rclone) ListFiles(ctx context.Context, dirPath string) ([]storagetypes.FileInfo, error) {
	r.logger.Infof("Listing files in %s", dirPath)

	remotePath := r.getRemotePath(dirPath)

	args := r.buildBaseArgs()
	args = append(args, "lsjson", remotePath)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		r.logger.Errorf("Failed to list files: %v, stderr: %s", err, stderr.String())
		return nil, fmt.Errorf("%w: %s", ErrFailedToListFiles, stderr.String())
	}

	var items []lsjsonItem
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		r.logger.Errorf("Failed to parse lsjson output: %v", err)
		return nil, fmt.Errorf("failed to parse lsjson output: %w", err)
	}

	files := make([]storagetypes.FileInfo, 0, len(items))
	for _, item := range items {
		var modTime time.Time
		if item.ModTime != "" {
			parsedTime, err := time.Parse(time.RFC3339Nano, item.ModTime)
			if err != nil {
				r.logger.Warnf("Failed to parse mod time %q for %s: %v", item.ModTime, item.Name, err)
			} else {
				modTime = parsedTime
			}
		}

		files = append(files, storagetypes.FileInfo{
			Name:    item.Name,
			Path:    path.Join(dirPath, item.Name),
			Size:    item.Size,
			IsDir:   item.IsDir,
			ModTime: modTime,
		})
	}

	r.logger.Debugf("Found %d files/directories in %s", len(files), dirPath)
	return files, nil
}

// OpenFile implements storage.StorageReadable
func (r *Rclone) OpenFile(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	r.logger.Infof("Opening file %s", filePath)

	remotePath := r.getRemotePath(filePath)

	size, err := r.getFileSize(ctx, remotePath)
	if err != nil {
		r.logger.Errorf("Failed to get file size: %v", err)
		return nil, 0, fmt.Errorf("%w: %v", ErrFailedToOpenFile, err)
	}

	args := r.buildBaseArgs()
	args = append(args, "cat", remotePath)

	cmd := exec.CommandContext(ctx, "rclone", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, 0, fmt.Errorf("failed to start rclone cat: %w", err)
	}

	reader := &rcloneCatReader{
		reader: stdout,
		cmd:    cmd,
		logger: r.logger,
	}

	r.logger.Debugf("Opened file %s (size: %d bytes)", filePath, size)
	return reader, size, nil
}

func (r *Rclone) getFileSize(ctx context.Context, remotePath string) (int64, error) {
	args := r.buildBaseArgs()
	args = append(args, "lsjson", remotePath)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return 0, err
	}

	var items []lsjsonItem
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		return 0, err
	}

	if len(items) > 0 {
		return items[0].Size, nil
	}
	return 0, nil
}

type rcloneCatReader struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
	logger *log.Logger
}

func (r *rcloneCatReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *rcloneCatReader) Close() error {
	if err := r.reader.Close(); err != nil {
		r.logger.Warnf("Failed to close reader: %v", err)
	}
	if err := r.cmd.Wait(); err != nil {
		r.logger.Warnf("rclone cat process exited with error: %v", err)
	}
	return nil
}
