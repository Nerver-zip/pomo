package taskform

import (
	"strings"

	"github.com/Bahaaio/pomo/db"
	"github.com/Bahaaio/pomo/ui/colors"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TaskFormSubmitMsg struct {
	Title       string
	Description string
	IsEdit      bool
	TaskID      int
}

type CloseFormMsg struct{}

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderFg).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Purple).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(colors.Cream).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colors.ErrorMessageFg).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(colors.DimGray)
)

type Model struct {
	inputs     []textinput.Model
	focused    int
	errMessage string
	isEdit     bool
	taskID     int
	width      int
	height     int
	help       help.Model
}

func NewForCreate() Model {
	return newForm(false, 0, "", "")
}

func NewForEdit(task db.Task) Model {
	return newForm(true, task.ID, task.Title, task.Description)
}

func newForm(isEdit bool, id int, initialTitle, initialDesc string) Model {
	titleInput := textinput.New()
	titleInput.Placeholder = "What are you focusing on?"
	titleInput.SetValue(initialTitle)
	titleInput.Focus()
	titleInput.CharLimit = 120
	titleInput.Width = 44

	descInput := textinput.New()
	descInput.Placeholder = "Optional notes or details..."
	descInput.SetValue(initialDesc)
	descInput.CharLimit = 250
	descInput.Width = 44

	return Model{
		inputs:  []textinput.Model{titleInput, descInput},
		focused: 0,
		isEdit:  isEdit,
		taskID:  id,
		help:    help.New(),
	}
}

func (m *Model) HandleWindowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
}

func (m *Model) HandleKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Cancel):
		return func() tea.Msg {
			return CloseFormMsg{}
		}

	case key.Matches(msg, Keys.Next):
		m.focused = (m.focused + 1) % len(m.inputs)
		return m.updateFocus()

	case key.Matches(msg, Keys.Prev):
		m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
		return m.updateFocus()

	case key.Matches(msg, Keys.Submit):
		title := strings.TrimSpace(m.inputs[0].Value())
		if title == "" {
			m.errMessage = "Error: Title cannot be empty!"
			return nil
		}

		desc := strings.TrimSpace(m.inputs[1].Value())
		return func() tea.Msg {
			return TaskFormSubmitMsg{
				Title:       title,
				Description: desc,
				IsEdit:      m.isEdit,
				TaskID:      m.taskID,
			}
		}

	default:
		// Forward key to currently focused text input
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return cmd
	}
}

func (m *Model) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focused {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	var b strings.Builder

	header := "New Task"
	if m.isEdit {
		header = "Edit Task"
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Title:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Description:"))
	b.WriteString("\n")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	if m.errMessage != "" {
		b.WriteString(errorStyle.Render(m.errMessage))
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
