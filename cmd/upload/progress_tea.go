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

type progressMsg float64

type progressErrMsg struct{ err error }

type progressDoneMsg struct{}

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

	sb.WriteString(fmt.Sprintf("  📁 %s\n", m.fileName))
	sb.WriteString(fmt.Sprintf("  📊 %s / %s\n\n",
		humanize.Bytes(uint64(m.bytesRead)),
		humanize.Bytes(uint64(m.fileSize)),
	))

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

type UploadProgress struct {
	program *tea.Program
	ctx     context.Context
	cancel  context.CancelFunc
}

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

func (up *UploadProgress) Start() {
	go func() {
		up.program.Run()
	}()
}

func (up *UploadProgress) UpdateProgress(percent float64) {
	up.program.Send(progressMsg(percent))
}

func (up *UploadProgress) SetError(err error) {
	up.program.Send(progressErrMsg{err: err})
}

func (up *UploadProgress) Done() {
	up.program.Send(progressDoneMsg{})
}

func (up *UploadProgress) Wait() {
	up.program.Wait()
}

func (up *UploadProgress) Quit() {
	up.program.Quit()
}
