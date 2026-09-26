// Package ui provides the terminal user interface for pomodoro sessions.
package ui

import (
	"github.com/Nerver-zip/pomo-tasker/ui/confirm"
	"github.com/Nerver-zip/pomo-tasker/ui/taskform"
	"github.com/Nerver-zip/pomo-tasker/ui/taskpicker"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Init() tea.Cmd {
	return m.timer.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, m.handleKeys(msg)

	case tea.WindowSizeMsg:
		return m, m.handleWindowResize(msg)

	case timer.TickMsg:
		return m, m.handleTimerTick(msg)

	case confirmTickMsg:
		return m, m.handleConfirmTick()

	case timer.StartStopMsg:
		return m, m.handleTimerStartStop(msg)

	case progress.FrameMsg:
		return m, m.handleProgressBarFrame(msg)

	case confirm.ChoiceMsg:
		return m, m.handleConfirmChoice(msg)

	case commandsDoneMsg:
		return m, m.handleCommandsDone()

	case taskpicker.TaskSelectedMsg:
		m.activeTask = &msg.Task
		m.sessionSummary.SetFocusedTask(msg.Task.ID, msg.Task.Title)
		m.sessionState = m.prevState
		return m, nil

	case taskpicker.OpenNewTaskFormMsg:
		m.form = taskform.NewForCreate()
		m.form.HandleWindowResize(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.sessionState = ShowingTaskForm
		return m, nil

	case taskpicker.OpenEditTaskFormMsg:
		m.form = taskform.NewForEdit(msg.Task)
		m.form.HandleWindowResize(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.sessionState = ShowingTaskForm
		return m, nil

	case taskpicker.TaskCompletedMsg:
		if m.taskRepo != nil {
			_ = m.taskRepo.Complete(msg.Task.ID)
			if m.activeTask != nil && m.activeTask.ID == msg.Task.ID {
				m.activeTask = nil
			}
			tasks, _ := m.taskRepo.ListPending()
			m.picker.SetTasks(tasks)
		}
		return m, nil

	case taskpicker.ClosePickerMsg:
		m.sessionState = m.prevState
		return m, nil

	case taskform.TaskFormSubmitMsg:
		if m.taskRepo != nil {
			if msg.IsEdit {
				_ = m.taskRepo.Update(msg.TaskID, msg.Title, msg.Description)
				if m.activeTask != nil && m.activeTask.ID == msg.TaskID {
					m.activeTask.Title = msg.Title
					m.activeTask.Description = msg.Description
					m.sessionSummary.SetFocusedTask(msg.TaskID, msg.Title)
				}
				tasks, _ := m.taskRepo.ListPending()
				m.picker.SetTasks(tasks)
				m.sessionState = ShowingTaskPicker
			} else {
				task, err := m.taskRepo.Create(msg.Title, msg.Description)
				if err == nil {
					m.activeTask = task
					m.sessionSummary.SetFocusedTask(task.ID, task.Title)
				}
				m.sessionState = m.prevState
			}
		} else {
			m.sessionState = m.prevState
		}
		return m, nil

	case taskform.CloseFormMsg:
		m.sessionState = ShowingTaskPicker
		return m, nil

	default:
		return m, nil
	}
}

func (m Model) View() string {
	if m.sessionState == Quitting {
		return ""
	}

	if m.sessionState == WaitingForCommands {
		return m.buildWaitingForCommandsView()
	}

	// show confirmation dialog
	if m.sessionState == ShowingConfirm {
		return m.buildConfirmDialogView()
	}

	// show task picker overlay
	if m.sessionState == ShowingTaskPicker {
		return m.picker.View()
	}

	// show task form overlay
	if m.sessionState == ShowingTaskForm {
		return m.form.View()
	}

	content := m.buildMainContent()
	content += m.buildStatusIndicators()
	content += m.buildProgressBar()

	help := m.buildHelpView()

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, content, help),
	)
}
