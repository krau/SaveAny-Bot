package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/enums/ctxkey"
)

// Execute implements core.Executable.
func (t *Task) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Infof("Starting yt-dlp download task %s", t.ID)

	if t.Progress != nil {
		t.Progress.OnStart(ctx, t)
	}

	// Create temporary directory for downloads
	tempDir, err := os.MkdirTemp(config.C().Temp.BasePath, "ytdlp-*")
	if err != nil {
		logger.Errorf("Failed to create temp directory: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	// Absolute: yt-dlp ignores --paths for absolute output templates.
	if tempDir, err = filepath.Abs(tempDir); err != nil {
		logger.Errorf("Failed to resolve temp directory: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return fmt.Errorf("failed to resolve temp directory: %w", err)
	}

	logger.Debugf("Created temp directory: %s", tempDir)

	// Download files using yt-dlp
	downloadedFiles, err := t.downloadFiles(ctx, tempDir)
	if err != nil {
		logger.Errorf("yt-dlp download failed: %v", err)
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}

	if len(downloadedFiles) == 0 {
		err := errors.New("no files were downloaded")
		logger.Error(err.Error())
		if t.Progress != nil {
			t.Progress.OnDone(ctx, t, err)
		}
		return err
	}

	// Transfer downloaded files to storage
	logger.Infof("Transferring %d file(s) to storage %s", len(downloadedFiles), t.Storage.Name())
	for _, relPath := range downloadedFiles {
		if err := t.transferFile(ctx, tempDir, relPath); err != nil {
			logger.Errorf("File transfer failed: %v", err)
			if t.Progress != nil {
				t.Progress.OnDone(ctx, t, err)
			}
			return err
		}
	}

	logger.Infof("yt-dlp task %s completed successfully", t.ID)
	if t.Progress != nil {
		t.Progress.OnDone(ctx, t, nil)
	}

	return nil
}

// buildDownloadCommand prepares the yt-dlp command and the remaining custom flags
// for a task. The bot owns the output directory: a custom -o/--output only
// contributes its template, relative to tempDir.
func buildDownloadCommand(cfg config.YtdlpConfig, tempDir string, flags []string) (*ytdlp.Command, []string, error) {
	userTemplate, rest, err := splitOutputTemplate(flags)
	if err != nil {
		return nil, nil, err
	}

	output, err := outputTemplatePath(tempDir, resolveFilenameTemplate(cfg, userTemplate))
	if err != nil {
		return nil, nil, err
	}
	cmd := ytdlp.New().Output(output)
	if cfg.RestrictFilenames {
		cmd = cmd.RestrictFilenames()
	}

	// Format/quality defaults only apply when the user passes no custom flags.
	if len(flags) == 0 {
		cmd = applyFormatConfig(cmd, cfg)
	}
	return cmd, rest, nil
}

// downloadFiles downloads files using yt-dlp and returns the list of downloaded file paths
func (t *Task) downloadFiles(ctx context.Context, tempDir string) ([]string, error) {
	logger := log.FromContext(ctx)

	cmd, flags, err := buildDownloadCommand(config.C().Ytdlp, tempDir, t.Flags)
	if err != nil {
		return nil, err
	}

	if t.Progress != nil {
		t.Progress.OnProgress(ctx, t, "Downloading...")
	}

	// Execute download with URLs and custom flags
	logger.Infof("Executing yt-dlp for %d URL(s) with %d custom flag(s)", len(t.URLs), len(flags))

	// Combine flags and URLs as arguments (flags first, then URLs)
	// yt-dlp accepts: yt-dlp [OPTIONS] URL [URL...]
	args := append(flags, t.URLs...)

	// Run with context for cancellation support
	result, err := cmd.Run(ctx, args...)
	if err != nil {
		// Check if context was canceled
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("yt-dlp execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("yt-dlp exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	// List downloaded files
	files, err := collectDownloadedFiles(tempDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		logger.Debugf("Downloaded file: %s", filepath.Base(file))
	}

	return files, nil
}

// collectDownloadedFiles walks dir recursively, since output templates may create
// subdirectories, and returns the files relative to dir.
func collectDownloadedFiles(dir string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to read temp directory: %w", err)
	}
	return files, nil
}

// transferFile transfers the file at relPath below tempDir to storage, keeping
// the directories an output template created.
func (t *Task) transferFile(ctx context.Context, tempDir, relPath string) error {
	logger := log.FromContext(ctx)
	filePath := filepath.Join(tempDir, relPath)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("Downloaded file not found: %s", filePath)
			return nil // Not a fatal error
		}
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Open file
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	// Set content length in context for storage
	ctx = context.WithValue(ctx, ctxkey.ContentLength, fileInfo.Size())

	// Save to storage
	destPath := filepath.Join(t.StorPath, sanitizeFilename(relPath))

	logger.Infof("Transferring file %s to %s:%s", relPath, t.Storage.Name(), destPath)

	if err := t.Storage.Save(ctx, f, destPath); err != nil {
		return fmt.Errorf("failed to save file %s to storage: %w", relPath, err)
	}

	logger.Infof("Successfully transferred file %s", relPath)

	if t.Progress != nil {
		t.Progress.OnProgress(ctx, t, fmt.Sprintf("Transferred: %s", relPath))
	}

	return nil
}

// sanitizeFilename removes or replaces problematic characters in filenames
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "\"", "'")
	return name
}
