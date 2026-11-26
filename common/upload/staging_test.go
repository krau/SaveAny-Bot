package upload

import (
	"path/filepath"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

func TestStagingPaths(t *testing.T) {
	base := t.TempDir()
	previous := config.C()
	cfg := previous
	cfg.Temp.BasePath = base
	config.Set(cfg)
	t.Cleanup(func() { config.Set(previous) })

	dir, err := StagingDir("task123")
	if err != nil {
		t.Fatalf("StagingDir: %v", err)
	}
	wantDir := filepath.Join(base, webUploadDir, "task123")
	if dir != wantDir {
		t.Fatalf("StagingDir = %q, want %q", dir, wantDir)
	}
	payload, err := StagingPayloadPath("task123")
	if err != nil {
		t.Fatalf("StagingPayloadPath: %v", err)
	}
	if payload != filepath.Join(wantDir, "payload") {
		t.Fatalf("StagingPayloadPath = %q", payload)
	}
}

func TestStagingRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"", ".", "..", "../escape", "sub/task", `sub\task`} {
		if _, err := StagingDir(id); err == nil {
			t.Fatalf("StagingDir(%q) succeeded", id)
		}
	}
}
