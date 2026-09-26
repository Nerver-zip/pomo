// Package summary tracks pomodoro sessions and renders visual summary with progress bar.
package summary

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/ui/colors"
	"github.com/charmbracelet/lipgloss"
)

var (
	messageStyle = lipgloss.NewStyle().Foreground(colors.SuccessMessageFg)
	errorStyle   = lipgloss.NewStyle().Foreground(colors.ErrorMessageFg)
)

type SessionSummary struct {
	totalWorkSessions int
	totalWorkDuration time.Duration

	totalBreakSessions int
	totalBreakDuration time.Duration

	isDatabaseUnavailable bool

	focusedTaskID       int
	focusedTaskTitle    string
	focusedTaskSessions int
	focusedTaskDuration time.Duration
}

// SetFocusedTask records the focused task information for the session exit summary.
func (t *SessionSummary) SetFocusedTask(id int, title string) {
	t.focusedTaskID = id
	t.focusedTaskTitle = title
}

// AddTaskSession records a completed work session for the focused task.
func (t *SessionSummary) AddTaskSession(id int, title string, elapsed time.Duration) {
	t.focusedTaskID = id
	t.focusedTaskTitle = title
	t.focusedTaskSessions++
	t.focusedTaskDuration += elapsed
}

// AddTaskDuration adds duration to the focused task for short sessions.
func (t *SessionSummary) AddTaskDuration(duration time.Duration) {
	t.focusedTaskDuration += duration
}

func (t SessionSummary) FocusedTaskID() int {
	return t.focusedTaskID
}

func (t SessionSummary) FocusedTaskTitle() string {
	return t.focusedTaskTitle
}

func (t SessionSummary) FocusedTaskSessions() int {
	return t.focusedTaskSessions
}

func (t SessionSummary) FocusedTaskDuration() time.Duration {
	return t.focusedTaskDuration
}

// AddSession adds a session to the summary based on the task type and elapsed time.
func (t *SessionSummary) AddSession(taskType config.TaskType, elapsed time.Duration) {
	if taskType == config.WorkTask {
		t.totalWorkSessions++
	} else {
		t.totalBreakSessions++
	}

	t.AddDuration(taskType, elapsed)
}

// AddDuration adds a duration to the summary based on the task type.
func (t *SessionSummary) AddDuration(taskType config.TaskType, duration time.Duration) {
	if taskType == config.WorkTask {
		t.totalWorkDuration += duration
	} else {
		t.totalBreakDuration += duration
	}
}

// SetDatabaseUnavailable marks the database as unavailable.
// prints a warning in the summary.
func (t *SessionSummary) SetDatabaseUnavailable() {
	t.isDatabaseUnavailable = true
}

// Render builds the formatted session summary as a string.
func (t SessionSummary) Render() string {
	if t.totalWorkDuration == 0 && t.totalBreakDuration == 0 {
		return ""
	}

	var b strings.Builder

	workIndicator := "sessions"
	if t.totalWorkSessions == 1 {
		workIndicator = "session"
	}

	breakIndicator := "sessions"
	if t.totalBreakSessions == 1 {
		breakIndicator = "session"
	}

	b.WriteString(messageStyle.Render("Session Summary:") + "\n")

	if t.focusedTaskID > 0 && t.focusedTaskTitle != "" {
		taskIndicator := "sessions"
		if t.focusedTaskSessions == 1 {
			taskIndicator = "session"
		}
		b.WriteString(fmt.Sprintf(" Focused Task: #%d %s (%d %s · %v)\n",
			t.focusedTaskID,
			t.focusedTaskTitle,
			t.focusedTaskSessions,
			taskIndicator,
			t.focusedTaskDuration,
		))
	}

	if t.totalWorkDuration > 0 {
		b.WriteString(fmt.Sprintf(" Work : %v (%d %s)\n", t.totalWorkDuration, t.totalWorkSessions, workIndicator))
	}

	if t.totalBreakDuration > 0 {
		b.WriteString(fmt.Sprintf(" Break: %v (%d %s)\n", t.totalBreakDuration, t.totalBreakSessions, breakIndicator))
	}

	if t.totalBreakDuration > 0 && t.totalWorkDuration > 0 {
		b.WriteString(fmt.Sprintf(" Total: %v\n", t.totalWorkDuration+t.totalBreakDuration))
	}

	if t.totalWorkDuration > 0 {
		b.WriteString(t.renderProgressBar() + "\n")
	}

	if t.isDatabaseUnavailable {
		b.WriteString(errorStyle.Render("\n Not saved (database unavailable)") + "\n")
	}

	return b.String()
}

// Print prints the session summary to the console.
func (t SessionSummary) Print() {
	rendered := t.Render()
	if rendered != "" {
		fmt.Print(rendered)
	}
}

// renderProgressBar returns a progress bar showing the ratio of work to total time.
func (t SessionSummary) renderProgressBar() string {
	const barWidth = 30

	totalDuration := t.totalWorkDuration + t.totalBreakDuration
	workRatio := float64(t.totalWorkDuration.Milliseconds()) / float64(totalDuration.Milliseconds())

	filledWidth := int(workRatio * barWidth)
	emptyWidth := barWidth - filledWidth

	bar := lipgloss.NewStyle().Foreground(colors.TimerFg).
		Render(strings.Repeat("█", filledWidth)) +
		strings.Repeat("░", emptyWidth)

	return fmt.Sprintf("\n [%s] %.0f%% work", bar, workRatio*100)
}
