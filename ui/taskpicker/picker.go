package taskpicker

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/pomo/db"
	"github.com/Nerver-zip/pomo/ui/colors"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TaskSelectedMsg struct {
	Task db.Task
}

type OpenNewTaskFormMsg struct{}

type OpenEditTaskFormMsg struct {
	Task db.Task
}

type TaskCompletedMsg struct {
	Task db.Task
}

type TaskUntrackedMsg struct{}

type ClosePickerMsg struct{}

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderFg).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Purple).
			MarginBottom(1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colors.ActiveButtonBg)

	itemStyle = lipgloss.NewStyle().
			Foreground(colors.Cream)

	dimStyle = lipgloss.NewStyle().
			Foreground(colors.DimGray)

	activeBadgeStyle = lipgloss.NewStyle().
				Foreground(colors.SuccessMessageFg).
				Bold(true)
)

type Model struct {
	tasks        []db.Task
	cursor       int
	activeTaskID int
	width        int
	height       int
	help         help.Model
}

func New(tasks []db.Task, activeTaskID int) Model {
	return Model{
		tasks:        tasks,
		cursor:       0,
		activeTaskID: activeTaskID,
		help:         help.New(),
	}
}

func (m *Model) SetTasks(tasks []db.Task) {
	m.tasks = tasks
	if m.cursor >= len(m.tasks) {
		m.cursor = max(0, len(m.tasks)-1)
	}
}

func (m *Model) SetActiveTaskID(id int) {
	m.activeTaskID = id
}

func (m Model) SelectedTask() *db.Task {
	if len(m.tasks) == 0 || m.cursor < 0 || m.cursor >= len(m.tasks) {
		return nil
	}
	return &m.tasks[m.cursor]
}

func (m *Model) HandleWindowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
}

func (m *Model) HandleKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return nil

	case key.Matches(msg, Keys.Down):
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
		return nil

	case key.Matches(msg, Keys.Select):
		if task := m.SelectedTask(); task != nil {
			return func() tea.Msg {
				return TaskSelectedMsg{Task: *task}
			}
		}
		return nil

	case key.Matches(msg, Keys.New):
		return func() tea.Msg {
			return OpenNewTaskFormMsg{}
		}

	case key.Matches(msg, Keys.Edit):
		if task := m.SelectedTask(); task != nil {
			return func() tea.Msg {
				return OpenEditTaskFormMsg{Task: *task}
			}
		}
		return nil

	case key.Matches(msg, Keys.Complete):
		if task := m.SelectedTask(); task != nil {
			return func() tea.Msg {
				return TaskCompletedMsg{Task: *task}
			}
		}
		return nil

	case key.Matches(msg, Keys.Untrack):
		return func() tea.Msg {
			return TaskUntrackedMsg{}
		}

	case key.Matches(msg, Keys.Close):
		return func() tea.Msg {
			return ClosePickerMsg{}
		}

	default:
		return nil
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Select Active Task"))
	b.WriteString("\n\n")

	if len(m.tasks) == 0 {
		b.WriteString(dimStyle.Render("No pending tasks. Press 'n' to create one."))
		b.WriteString("\n\n")
	} else {
		for i, t := range m.tasks {
			cursor := "  "
			isCurrent := i == m.cursor
			if isCurrent {
				cursor = "> "
			}

			activeBadge := ""
			if t.ID == m.activeTaskID {
				activeBadge = activeBadgeStyle.Render(" (active)")
			}

			metrics := ""
			if t.TotalPomodoros > 0 {
				metrics = dimStyle.Render(fmt.Sprintf(" (%d · %s)", t.TotalPomodoros, formatDuration(t.TotalDuration)))
			}

			line := fmt.Sprintf("[%d] %s%s%s", t.ID, t.Title, activeBadge, metrics)
			if isCurrent {
				b.WriteString(selectedItemStyle.Render(cursor + line))
			} else {
				b.WriteString(itemStyle.Render(cursor + line))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("───────────────────────────────────────────────────"))
	b.WriteString("\n")
	b.WriteString(m.help.View(Keys))

	dialog := borderStyle.Render(b.String())

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialog,
		)
	}
	return dialog
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0m"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
