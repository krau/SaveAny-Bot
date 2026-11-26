package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

const webUploadDir = "web-upload"

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

func StagingDir(root, taskID string) (string, error) {
	if err := validateStagingID(taskID); err != nil {
		return "", err
	}
	// Records created before the staging root was persisted use the current root.
	if root == "" {
		var err error
		root, err = StagingRoot()
		if err != nil {
			return "", err
		}
	}
	if !filepath.IsAbs(root) || filepath.Base(filepath.Clean(root)) != webUploadDir {
		return "", fmt.Errorf("invalid upload staging root %q", root)
	}
	return filepath.Join(root, taskID), nil
}

func StagingPayloadPath(root, taskID string) (string, error) {
	dir, err := StagingDir(root, taskID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "payload"), nil
}

func RemoveStaging(root, taskID string) error {
	dir, err := StagingDir(root, taskID)
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
