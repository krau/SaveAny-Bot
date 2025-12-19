//go:build !no_bubbletea

package upload

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

// progressMsg is sent to update the progress bar
type progressMsg float64

// progressErrMsg is sent when an error occurs
type progressErrMsg struct{ err error }

// progressDoneMsg is sent when the upload is complete
type progressDoneMsg struct{}

// uploadModel is the bubbletea model for the upload progress UI
type uploadModel struct {
	progress  progress.Model
	fileName  string
	fileSize  int64
	bytesRead int64
	err       error
	done      bool
	quitting  bool
	width     int
}

func newUploadModel(fileName string, fileSize int64) uploadModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)
	return uploadModel{
		progress: p,
		fileName: fileName,
		fileSize: fileSize,
	}
}

func (m uploadModel) Init() tea.Cmd {
	return nil
}

func (m uploadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = min(msg.Width-10, 80)
		return m, nil

	case progressMsg:
		var cmds []tea.Cmd
		percent := float64(msg)
		m.bytesRead = int64(percent * float64(m.fileSize))

		cmds = append(cmds, m.progress.SetPercent(percent))
		return m, tea.Batch(cmds...)

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressDoneMsg:
		m.done = true
		m.progress.SetPercent(1.0)
		return m, tea.Quit

	case progress.FrameMsg:
		// Don't process frame messages if we're done or quitting
		if m.done || m.quitting {
			return m, nil
		}
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m uploadModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  ❌ Error: %s\n\n", m.err.Error())
	}

	var sb strings.Builder
	sb.WriteString("\n")

	// File info
	sb.WriteString(fmt.Sprintf("  📁 %s\n", m.fileName))
	sb.WriteString(fmt.Sprintf("  📊 %s / %s\n\n",
		humanize.Bytes(uint64(m.bytesRead)),
		humanize.Bytes(uint64(m.fileSize)),
	))

	// Progress bar
	sb.WriteString("  ")
	sb.WriteString(m.progress.View())
	sb.WriteString("\n\n")

	if m.done {
		sb.WriteString("  √ Upload complete!\n\n")
	} else {
		sb.WriteString(helpStyle.Render("  Press Ctrl+C to cancel"))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// UploadProgress manages the progress UI for uploads
type UploadProgress struct {
	program *tea.Program
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewUploadProgress creates a new upload progress tracker
func NewUploadProgress(ctx context.Context, fileName string, fileSize int64) *UploadProgress {
	model := newUploadModel(fileName, fileSize)
	ctx, cancel := context.WithCancel(ctx)
	p := tea.NewProgram(
		model,
		tea.WithoutSignalHandler(),
		tea.WithContext(ctx),
		tea.WithInput(nil), // Disable keyboard input, rely on context cancellation
	)
	return &UploadProgress{
		program: p,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the progress UI in a goroutine and returns immediately
func (up *UploadProgress) Start() {
	go func() {
		up.program.Run()
	}()
}

// UpdateProgress updates the progress bar with a new percentage (0.0 - 1.0)
func (up *UploadProgress) UpdateProgress(percent float64) {
	up.program.Send(progressMsg(percent))
}

// SetError sets an error and quits the progress UI
func (up *UploadProgress) SetError(err error) {
	up.program.Send(progressErrMsg{err: err})
}

// Done signals that the upload is complete
func (up *UploadProgress) Done() {
	up.program.Send(progressDoneMsg{})
}

// Wait waits for the progress UI to finish
func (up *UploadProgress) Wait() {
	up.program.Wait()
}

// Quit quits the progress UI
func (up *UploadProgress) Quit() {
	up.program.Quit()
}
