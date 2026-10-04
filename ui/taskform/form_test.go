package taskform

import (
	"testing"

	"github.com/Nerver-zip/pomo/db"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForm_CreateValidationAndSubmit(t *testing.T) {
	form := NewForCreate()

	// 1. Submit empty title -> should show error
	cmd := form.HandleKeys(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Contains(t, form.errMessage, "Title cannot be empty")

	// 2. Type title
	for _, r := range "Build feature" {
		form.HandleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// 3. Tab to description
	cmd = form.HandleKeys(tea.KeyMsg{Type: tea.KeyTab})
	assert.NotNil(t, cmd)
	assert.Equal(t, 1, form.focused)

	// Type description
	for _, r := range "Important notes" {
		form.HandleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// 4. Submit form
	cmd = form.HandleKeys(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	submitMsg, ok := msg.(TaskFormSubmitMsg)
	require.True(t, ok)
	assert.Equal(t, "Build feature", submitMsg.Title)
	assert.Equal(t, "Important notes", submitMsg.Description)
	assert.False(t, submitMsg.IsEdit)

	// 5. View rendering
	form.HandleWindowResize(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := form.View()
	assert.Contains(t, view, "New Task")
	assert.Contains(t, view, "Build feature")
}

func TestForm_EditAndCancel(t *testing.T) {
	task := db.Task{
		ID:          42,
		Title:       "Existing Task",
		Description: "Existing Desc",
	}

	form := NewForEdit(task)
	assert.True(t, form.isEdit)
	assert.Equal(t, 42, form.taskID)

	// Cancel (Esc)
	cmd := form.HandleKeys(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	_, ok := cmd().(CloseFormMsg)
	assert.True(t, ok)

	// View rendering
	view := form.View()
	assert.Contains(t, view, "Edit Task")
	assert.Contains(t, view, "Existing Task")
}
