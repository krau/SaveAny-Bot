package upload

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/krau/SaveAny-Bot/common/utils/dlutil"
)

const (
	uploadProgressBarWidth   = 30
	uploadProgressMinRefresh = 100 * time.Millisecond
	uploadProgressNameLimit  = 32
)

type UploadProgress struct {
	mu          sync.Mutex
	fileName    string
	fileSize    int64
	startedAt   time.Time
	lastRefresh time.Time
	lastPercent float64
	finished    bool
	interactive bool
	lineWidth   int
}

func NewUploadProgress(fileName string, fileSize int64) *UploadProgress {
	return &UploadProgress{
		fileName:    truncateProgressName(fileName, uploadProgressNameLimit),
		fileSize:    fileSize,
		interactive: isTerminal(os.Stderr),
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (up *UploadProgress) Start() {
	if !up.interactive {
		return
	}
	up.mu.Lock()
	defer up.mu.Unlock()
	up.startedAt = time.Now()
	up.renderLocked(0)
}

func (up *UploadProgress) UpdateProgress(percent float64) {
	if !up.interactive {
		return
	}
	up.mu.Lock()
	defer up.mu.Unlock()
	if up.finished {
		return
	}
	percent = min(max(percent, 0), 1)
	if percent < 1 && time.Since(up.lastRefresh) < uploadProgressMinRefresh {
		return
	}
	up.renderLocked(percent)
}

func (up *UploadProgress) SetError(error) {
	if !up.interactive {
		return
	}
	up.mu.Lock()
	defer up.mu.Unlock()
	if up.finished {
		return
	}
	up.finished = true
	up.clearLocked()
}

// Done completes the progress line.
func (up *UploadProgress) Done() {
	if !up.interactive {
		return
	}
	up.mu.Lock()
	defer up.mu.Unlock()
	if up.finished {
		return
	}
	up.finished = true
	if up.lastPercent < 1 {
		up.renderLocked(1)
	}
	fmt.Fprint(os.Stderr, "\n")
}

func (up *UploadProgress) renderLocked(percent float64) {
	up.lastRefresh = time.Now()
	up.lastPercent = percent
	line := up.lineLocked(percent)
	padding := up.lineWidth - utf8.RuneCountInString(line)
	if padding < 0 {
		up.lineWidth = utf8.RuneCountInString(line)
		padding = 0
	}
	fmt.Fprint(os.Stderr, "\r", line, strings.Repeat(" ", padding))
}

func (up *UploadProgress) clearLocked() {
	if up.lineWidth == 0 {
		return
	}
	fmt.Fprint(os.Stderr, "\r", strings.Repeat(" ", up.lineWidth), "\r")
}

func (up *UploadProgress) lineLocked(percent float64) string {
	uploaded := int64(percent * float64(up.fileSize))
	speed := "--"
	if uploaded > 0 && time.Since(up.startedAt) >= time.Second {
		speed = dlutil.FormatSize(int64(dlutil.GetSpeed(uploaded, up.startedAt))) + "/s"
	}
	return fmt.Sprintf(
		"%s %s %3.0f%% %s/%s %s",
		up.fileName,
		uploadProgressBar(percent),
		percent*100,
		dlutil.FormatSize(uploaded),
		dlutil.FormatSize(up.fileSize),
		speed,
	)
}

func uploadProgressBar(percent float64) string {
	filled := int(percent * uploadProgressBarWidth)
	filled = min(max(filled, 0), uploadProgressBarWidth)
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", uploadProgressBarWidth-filled) + "]"
}

func truncateProgressName(name string, limit int) string {
	runes := []rune(name)
	if len(runes) <= limit {
		return name
	}
	ext := path.Ext(name)
	if utf8.RuneCountInString(ext) >= limit-2 {
		return string(runes[:limit-1]) + "…"
	}
	baseLimit := limit - utf8.RuneCountInString(ext) - 1
	base := []rune(strings.TrimSuffix(name, ext))
	return string(base[:baseLimit]) + "…" + ext
}
