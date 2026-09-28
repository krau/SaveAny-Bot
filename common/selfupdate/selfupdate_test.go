package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/krau/SaveAny-Bot/config"
)

const (
	reexecHelperEnv  = "SAVEANY_TEST_REEXEC"
	reexecStageEnv   = "SAVEANY_TEST_REEXEC_STAGE"
	replaceHelperEnv = "SAVEANY_TEST_REPLACE"
	reexecMarkerEnv  = "SAVEANY_TEST_MARKER"
)

func testBinary(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to find the test binary: %v", err)
	}
	return exe
}

// TestRestartReexecsInPlace runs TestRestartHelper as its own process, which
// restarts itself; what writes the marker afterwards has to be that same
// process, with the pid and arguments it had before the restart.
func TestRestartReexecsInPlace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows cannot replace a running program")
	}
	marker := filepath.Join(t.TempDir(), "marker")
	cmd := exec.Command(testBinary(t), "-test.run=TestRestartHelper", "-test.v")
	cmd.Env = append(os.Environ(), reexecHelperEnv+"=1", reexecMarkerEnv+"="+marker)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper process failed: %v\n%s", err, out)
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("the helper process did not come back from the restart: %v\n%s", err, out)
	}
	pid, args, _ := strings.Cut(string(b), " ")
	if pid != strconv.Itoa(cmd.Process.Pid) {
		t.Fatalf("restart ran as pid %s, want the pid it replaced: %d", pid, cmd.Process.Pid)
	}
	if !strings.Contains(args, "-test.run=TestRestartHelper") {
		t.Fatalf("the new version ran with %q as its arguments", args)
	}
}

// TestRestartHelper is not a test on its own: it is the process
// TestRestartReexecsInPlace starts. It restarts itself, and the version that
// comes back records the pid and arguments it runs with.
func TestRestartHelper(t *testing.T) {
	if os.Getenv(reexecHelperEnv) == "" {
		t.Skip("only runs as a helper process")
	}
	if os.Getenv(reexecStageEnv) == "" {
		t.Setenv(reexecStageEnv, "2")
		RequestRestart()
		if !ShouldRestart() {
			t.Fatal("the restart request was not recorded")
		}
		if err := Restart(); err != nil {
			t.Fatalf("failed to restart: %v", err)
		}
		t.Fatal("restart returned without replacing the process")
	}
	marker := fmt.Sprintf("%d %s", os.Getpid(), strings.Join(os.Args, " "))
	if err := os.WriteFile(os.Getenv(reexecMarkerEnv), []byte(marker), 0o644); err != nil {
		t.Fatalf("failed to write the marker: %v", err)
	}
}

// TestRestartRunsTheFileAnUpdateReplaced runs the helper below from a copy of
// this test binary, which is then moved aside and replaced the way an update
// does it. The restart has to run what took the binary's place, from the same
// pid, rather than the file the running process was moved to.
func TestRestartRunsTheFileAnUpdateReplaced(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows cannot replace a running program")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "saveany-bot")
	if err := copyFile(testBinary(t), exe); err != nil {
		t.Fatalf("failed to copy the test binary: %v", err)
	}
	marker := filepath.Join(dir, "marker")
	cmd := exec.Command(exe, "-test.run=TestRestartAfterReplaceHelper")
	cmd.Env = append(os.Environ(), replaceHelperEnv+"=1", reexecMarkerEnv+"="+marker)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper process failed: %v\n%s", err, out)
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("the replacement never ran: %v\n%s", err, out)
	}
	what, pid, _ := strings.Cut(string(b), " ")
	if what != "script" {
		t.Fatalf("the restart ran the old file, not the one replaced into %s", filepath.Base(exe))
	}
	if pid != strconv.Itoa(cmd.Process.Pid) {
		t.Fatalf("the replacement ran as pid %s, want the pid it replaced: %d", pid, cmd.Process.Pid)
	}
}

// TestRestartAfterReplaceHelper is the process TestRestartRunsTheFileAnUpdateReplaced
// starts from a copy of the test binary: it moves itself aside, writes a shell
// script in its place, and restarts. The script is what says which file ran.
func TestRestartAfterReplaceHelper(t *testing.T) {
	if os.Getenv(replaceHelperEnv) == "" {
		t.Skip("only runs as a helper process")
	}
	exe, err := SelfPath()
	if err != nil {
		t.Fatalf("failed to find the running binary: %v", err)
	}
	dir, name := filepath.Split(exe)
	if err := os.Rename(exe, filepath.Join(dir, "."+name+".old")); err != nil {
		t.Fatalf("failed to move the binary aside: %v", err)
	}
	script := "#!/bin/sh\nprintf 'script %s' \"$$\" > \"$" + reexecMarkerEnv + "\"\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write the replacement: %v", err)
	}
	if err := Restart(); err != nil {
		t.Fatalf("failed to restart: %v", err)
	}
	t.Fatal("restart returned without replacing the process")
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o755)
}

func TestCleanLeftovers(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "saveany-bot")
	if err := os.WriteFile(exe, []byte("this version"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(filepath.Dir(exe), ".saveany-bot.old")
	duplicate := filepath.Join(filepath.Dir(exe), ".saveany-bot.new")
	for _, path := range []string{old, duplicate} {
		if err := os.WriteFile(path, []byte("this version"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// a .new holding a version that is not installed yet is an update in
	// progress, not a leftover
	pending := filepath.Join(t.TempDir(), "saveany-bot")
	if err := os.WriteFile(pending, []byte("this version"), 0o755); err != nil {
		t.Fatal(err)
	}
	download := filepath.Join(filepath.Dir(pending), ".saveany-bot.new")
	if err := os.WriteFile(download, []byte("a newer version"), 0o755); err != nil {
		t.Fatal(err)
	}

	cleanLeftovers(exe)
	cleanLeftovers(pending)

	for _, path := range []string{old, duplicate} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s was not removed", filepath.Base(path))
		}
	}
	if _, err := os.Stat(download); err != nil {
		t.Errorf("a download that is not the binary was removed: %v", err)
	}
}

func TestUpdatableRefusesWhatItCannotReplace(t *testing.T) {
	version, docker := config.Version, config.Docker
	t.Cleanup(func() { config.Version, config.Docker = version, docker })

	config.Docker, config.Version = "false", "dev"
	if err := Updatable(); !errors.Is(err, ErrDevBuild) {
		t.Errorf("Updatable on a build from source: %v", err)
	}

	config.Docker, config.Version = "true", "1.2.3"
	if err := Updatable(); !errors.Is(err, ErrDocker) {
		t.Errorf("Updatable in a Docker image: %v", err)
	}

	config.Docker = "false"
	if err := Updatable(); err != nil {
		t.Errorf("Updatable on a release build: %v", err)
	}
}
