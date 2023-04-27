package tui

import (
	"fmt"
	"strings"
	"time"

	"c8backup/pkg/runner"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

// A simple example that shows how to render an animated progress bar. In this
// example we bump the progress by 25% every two seconds, animating our
// progress bar to its new target state.
//
// It's also possible to render a progress bar in a more static fashion without
// transitions. For details on that approach see the progress-static example.

type tickMsg time.Time
type Model struct {
	Progress   progress.Model
	definition runner.BackupDefinition
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.Progress.Width = msg.Width - padding*2 - 4
		if m.Progress.Width > maxWidth {
			m.Progress.Width = maxWidth
		}
		return m, nil
	case runner.BackupDefinition:
		m.definition = msg
		cmd := m.Progress.SetPercent(msg.Percent())
		return m, tea.Batch(tickCmd(), cmd)

	case tickMsg:
		if m.definition.HasFinished() {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		return m, tea.Batch(tickCmd(), nil)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.Progress.Update(msg)
		m.Progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m Model) View() string {
	pad := strings.Repeat(" ", padding)

	var backupMsg string
	if m.definition.HasFinished() {
		backupMsg = fmt.Sprintf("Backup ID: %d", m.definition.BackupID())
	}

	return "\n" +
		pad + m.Progress.View() + "\n\n" +
		pad + m.definition.String() + "\n\n" +
		pad + backupMsg + "\n" +
		pad + helpStyle("Press any key to quit")
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
