package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

const webUploadDir = "web-upload"

// StagingRoot is the parent directory for files uploaded through the web API.
func StagingRoot() (string, error) {
	base := strings.TrimSpace(config.C().Temp.BasePath)
	if base == "" {
		base = filepath.Join(os.TempDir(), "saveany-bot")
	}
	absolute, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve staging root: %w", err)
	}
	return filepath.Join(absolute, webUploadDir), nil
}

// StagingDir returns the isolated directory for one web upload task.
func StagingDir(taskID string) (string, error) {
	if err := validateStagingID(taskID); err != nil {
		return "", err
	}
	root, err := StagingRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, taskID), nil
}

// StagingPayloadPath returns the completed payload path for one upload task.
func StagingPayloadPath(taskID string) (string, error) {
	dir, err := StagingDir(taskID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "payload"), nil
}

// RemoveStaging deletes all staged data for one upload task.
func RemoveStaging(taskID string) error {
	dir, err := StagingDir(taskID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

func validateStagingID(taskID string) error {
	if taskID == "" || taskID == "." || taskID == ".." ||
		strings.ContainsAny(taskID, `/\`) ||
		filepath.Base(taskID) != taskID || !filepath.IsLocal(taskID) {
		return fmt.Errorf("invalid upload staging ID %q", taskID)
	}
	return nil
}
