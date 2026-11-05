package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/blang/semver"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/unvgo/ghselfupdate"
)

var (
	ErrDevBuild      = errors.New("not a release build")
	ErrDocker        = errors.New("built into a Docker image")
	ErrNotWritable   = errors.New("the binary's folder is not writable")
	ErrRestart       = errors.New(RestartSignal)
	ErrNoRelease     = errors.New("no release found")
	ErrMajorUpgrade  = errors.New("major upgrade requires a manual download and migration review")
	ErrAlreadyLatest = errors.New("already on the latest version")
)

// RestartSignal marks a handler error that asks the client to restart the
// process after an update.
const RestartSignal = "saveanybot-restart"

var (
	restarting atomic.Bool

	self struct {
		once sync.Once
		path string
		err  error
	}
)

func RequestRestart() {
	restarting.Store(true)
}

func ShouldRestart() bool {
	return restarting.Load()
}

func SelfPath() (string, error) {
	self.once.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			self.err = fmt.Errorf("failed to find the running binary: %w", err)
			return
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			self.err = fmt.Errorf("failed to resolve %s: %w", exe, err)
			return
		}
		self.path = exe
	})
	return self.path, self.err
}

func Updatable() error {
	if config.Docker == "true" {
		return ErrDocker
	}
	if _, err := semver.Parse(config.Version); err != nil {
		return fmt.Errorf("%w: %s", ErrDevBuild, config.Version)
	}
	exe, err := SelfPath()
	if err != nil {
		return err
	}
	if dir := filepath.Dir(exe); !writable(dir) {
		return fmt.Errorf("%w: %s", ErrNotWritable, dir)
	}
	return nil
}

// CheckTarget validates that latest can be installed by this build.
func CheckTarget(current, latest semver.Version) error {
	if latest.Major > current.Major {
		return fmt.Errorf("%w: %s -> %s", ErrMajorUpgrade, current, latest)
	}
	if !latest.GT(current) {
		return fmt.Errorf("%w: %s", ErrAlreadyLatest, current)
	}
	return nil
}

func Update(current semver.Version) (*ghselfupdate.Release, error) {
	if err := Updatable(); err != nil {
		return nil, err
	}
	latest, found, err := ghselfupdate.DetectLatest(config.GitRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to detect latest release: %w", err)
	}
	if !found {
		return nil, ErrNoRelease
	}
	if err := CheckTarget(current, latest.Version); err != nil {
		return nil, err
	}
	exe, err := SelfPath()
	if err != nil {
		return nil, err
	}
	if err := ghselfupdate.DefaultUpdater().UpdateTo(latest, exe); err != nil {
		return nil, fmt.Errorf("failed to update: %w", err)
	}
	return latest, nil
}

func Restart() error {
	exe, err := SelfPath()
	if err != nil {
		return err
	}
	return reexec(exe, os.Args, os.Environ())
}

func CleanLeftovers() {
	exe, err := SelfPath()
	if err != nil {
		log.Warnf("Failed to resolve executable path for cleanup: %v", err)
		return
	}
	cleanLeftovers(exe)
}

func cleanLeftovers(exe string) {
	dir, name := filepath.Split(exe)
	if err := os.Remove(filepath.Join(dir, "."+name+".old")); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warnf("Failed to remove stale update file: %v", err)
	}
	staged := filepath.Join(dir, "."+name+".new")
	if same, err := sameContent(exe, staged); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("Failed to compare staged update: %v", err)
		}
	} else if same {
		if err := os.Remove(staged); err != nil {
			log.Warnf("Failed to remove staged update: %v", err)
		}
	}
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".saveany-update-*")
	if err != nil {
		return false
	}
	if err := f.Close(); err != nil {
		log.Warnf("Failed to close update probe: %v", err)
		return false
	}
	if err := os.Remove(f.Name()); err != nil {
		log.Warnf("Failed to remove update probe: %v", err)
		return false
	}
	return true
}

func sameContent(a, b string) (bool, error) {
	fa, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	fb, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if fa.Size() != fb.Size() {
		return false, nil
	}
	ha, err := fileHash(a)
	if err != nil {
		return false, err
	}
	hb, err := fileHash(b)
	if err != nil {
		return false, err
	}
	return ha == hb, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
