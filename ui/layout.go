package ui

import (
	"fmt"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/ui/ascii"
	"github.com/Nerver-zip/pomo-tasker/ui/colors"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxWidth           = 80
	margin             = 4
	padding            = 2
	separator          = " — "
	pausedIndicator    = "(paused)"
	completedIndicator = "done!"
)

func (m *Model) buildConfirmDialogView() string {
	idle := time.Since(m.confirmStartTime).Truncate(time.Second)
	title := m.currentTaskType.Opposite().GetTask().Title

	// if we're prompting to start a long break
	if m.cyclePosition == m.longBreak.After {
		title = "long " + title
	}

	return m.confirmDialog.View("start "+title+"?", time.Duration(idle))
}

func (m *Model) buildMainContent() string {
	timeLeft := m.buildTimeLeft()

	title := m.currentTask.Title
	if m.activeTask != nil && m.currentTaskType == config.WorkTask {
		title = fmt.Sprintf("#%d · %s", m.activeTask.ID, m.activeTask.Title)
	}

	var content string
	if m.useTimerArt {
		content = timeLeft + "\n\n" + title
	} else {
		content = title
		if !m.timer.Timedout() {
			content += separator + timeLeft
		}
	}

	if m.activeTask != nil && m.currentTaskType == config.WorkTask && m.activeTask.Description != "" && m.width >= 60 {
		desc := m.activeTask.Description
		maxDescLen := m.width - 10
		if len(desc) > maxDescLen && maxDescLen > 3 {
			desc = desc[:maxDescLen-3] + "..."
		}
		content += "\n" + lipgloss.NewStyle().Faint(true).Render(desc)
	}

	return content
}

func (m *Model) buildStatusIndicators() string {
	if m.timer.Timedout() {
		return separator + completedIndicator
	}

	indicators := ""

	if m.longBreak.Enabled {
		indicators += fmt.Sprintf(" · %d/%d", m.cyclePosition, m.longBreak.After)
	}

	if m.activeTask != nil {
		indicators += fmt.Sprintf(" · task: %d 🍅", m.activeTask.TotalPomodoros)
	}

	if m.sessionState == Paused {
		indicators += " " + pausedIndicator
	}

	return indicators
}

func (m *Model) buildProgressBar() string {
	return "\n\n" + m.progressBar.View() + "\n"
}

// returns time left as a string in HH:MM:SS format
func (m *Model) buildTimeLeft() string {
	left := m.timer.Timeout
	hours := int(left.Hours())
	minutes := int(left.Minutes()) % 60
	seconds := int(left.Seconds()) % 60

	time := ""

	// only show hours if they are non-zero
	if hours > 0 {
		time += fmt.Sprintf("%02d:", hours)
	}
	time += fmt.Sprintf("%02d:%02d", minutes, seconds)

	if m.useTimerArt {
		time = ascii.RenderNumber(time, m.timerFont)

		// remove color on pause
		if m.sessionState == Paused {
			noColor := m.asciiTimerStyle.Foreground(colors.PauseFg)
			return noColor.Render(time)
		}

		return m.asciiTimerStyle.Render(time)
	}

	return time
}

func (m *Model) buildHelpView() string {
	return m.help.View(keyMap)
}

func (m Model) buildWaitingForCommandsView() string {
	help := m.help.View(KeyMap{Quit: keyMap.Quit})

	message := lipgloss.JoinVertical(
		lipgloss.Center,
		"Waiting for post commands to complete...",
		"\n",
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		message,
	)
}
