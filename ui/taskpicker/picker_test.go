package taskpicker

import (
	"testing"
	"time"

	"github.com/Nerver-zip/pomo-tasker/db"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPicker_NavigationAndSelection(t *testing.T) {
	tasks := []db.Task{
		{ID: 1, Title: "Task 1", Status: db.TaskPending, TotalPomodoros: 2, TotalDuration: 50 * time.Minute},
		{ID: 2, Title: "Task 2", Status: db.TaskPending},
		{ID: 3, Title: "Task 3", Status: db.TaskPending},
	}

	picker := New(tasks, 1)
	assert.Equal(t, 1, picker.SelectedTask().ID)

	// Move down
	picker.HandleKeys(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, picker.SelectedTask().ID)

	// Move down again
	picker.HandleKeys(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 3, picker.SelectedTask().ID)

	// Bound check: down again stays at 3
	picker.HandleKeys(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 3, picker.SelectedTask().ID)

	// Move up
	picker.HandleKeys(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 2, picker.SelectedTask().ID)

	// Select task 2 (Enter)
	cmd := picker.HandleKeys(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	selectedMsg, ok := msg.(TaskSelectedMsg)
	require.True(t, ok)
	assert.Equal(t, 2, selectedMsg.Task.ID)

	// Press 'n' to open new task modal
	cmd = picker.HandleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd)
	_, ok = cmd().(OpenNewTaskFormMsg)
	assert.True(t, ok)

	// Press 'c' to complete task
	cmd = picker.HandleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	require.NotNil(t, cmd)
	doneMsg, ok := cmd().(TaskCompletedMsg)
	require.True(t, ok)
	assert.Equal(t, 2, doneMsg.Task.ID)

	// Press 'esc' to close
	cmd = picker.HandleKeys(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	_, ok = cmd().(ClosePickerMsg)
	assert.True(t, ok)

	// View rendering
	picker.HandleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := picker.View()
	assert.Contains(t, view, "Select Active Task")
	assert.Contains(t, view, "Task 1")
	assert.Contains(t, view, "(active)")
}

func TestPicker_EmptyList(t *testing.T) {
	picker := New([]db.Task{}, 0)
	assert.Nil(t, picker.SelectedTask())

	view := picker.View()
	assert.Contains(t, view, "No pending tasks")
}
