package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/db"
	"github.com/Nerver-zip/pomo-tasker/ui/taskpicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database, err := sqlx.Open("sqlite", fmt.Sprintf("file:ui_test_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	require.NoError(t, err)
	database.SetMaxOpenConns(1)
	_, err = database.Exec("PRAGMA foreign_keys = ON;")
	require.NoError(t, err)
	err = db.RunMigrations(database)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = database.Close()
	})
	return database
}

func testConfig() config.Config {
	return config.Config{
		Work: config.Task{
			Duration: 25 * time.Minute,
			Title:    "work",
		},
		Break: config.Task{
			Duration: 5 * time.Minute,
			Title:    "break",
		},
		LongBreak: config.LongBreak{
			Enabled:  true,
			Duration: 15 * time.Minute,
			After:    4,
		},
		OnSessionEnd: "ask",
	}
}

func TestTimer_UntrackedSession(t *testing.T) {
	database := newTestDB(t)
	cfg := testConfig()

	m := NewModelWithDB(config.WorkTask, cfg, database)
	assert.Nil(t, m.ActiveTask(), "initial active task must be nil for untracked sessions")

	// Simulate work elapsed and record session
	m.elapsed = 15 * time.Minute
	m.recordSession()

	// Verify database record has task_id = nil
	var sessions []db.Session
	err := database.Select(&sessions, "SELECT id, task_id, type, duration FROM sessions;")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Nil(t, sessions[0].TaskID, "untracked work session must have nil task_id")
	assert.Equal(t, "work", sessions[0].Type)
	assert.Equal(t, 15*time.Minute, sessions[0].Duration)
}

func TestTimer_AttachedTask_And_FlowRetentionAcrossBreaks(t *testing.T) {
	database := newTestDB(t)
	cfg := testConfig()

	taskRepo := db.NewTaskRepo(database)
	task, err := taskRepo.Create("Write unit tests", "Comprehensive integration coverage")
	require.NoError(t, err)

	m := NewModelWithDB(config.WorkTask, cfg, database)
	m.SetInitialTask(task)
	require.NotNil(t, m.ActiveTask())
	assert.Equal(t, task.ID, m.ActiveTask().ID)

	m.handleWindowResize(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Check view rendering includes task details
	view := m.View()
	assert.Contains(t, view, fmt.Sprintf("#%d · Write unit tests", task.ID))
	assert.Contains(t, view, "Comprehensive integration coverage")

	// 1. Simulate Work Session completion
	m.elapsed = 25 * time.Minute
	m.recordSession()

	// Verify metrics updated on active task in memory
	assert.Equal(t, 1, m.ActiveTask().TotalPomodoros)
	assert.Equal(t, 25*time.Minute, m.ActiveTask().TotalDuration)

	// Verify session recorded in DB with task_id
	var sessions []db.Session
	err = database.Select(&sessions, "SELECT id, task_id, type, duration FROM sessions WHERE task_id = ?;", task.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, task.ID, *sessions[0].TaskID)
	assert.Equal(t, "work", sessions[0].Type)

	// 2. Transition to Break session (nextSession)
	m.nextSession()
	assert.Equal(t, config.BreakTask, m.currentTaskType)
	assert.NotNil(t, m.ActiveTask(), "active task must be retained across break transitions for flow state")
	assert.Equal(t, task.ID, m.ActiveTask().ID)

	// 3. Simulate Break Session completion
	m.elapsed = 5 * time.Minute
	m.recordSession()

	// Verify break session recorded in DB has task_id = nil (break is cognitive rest)
	var breakSessions []db.Session
	err = database.Select(&breakSessions, "SELECT id, task_id, type, duration FROM sessions WHERE type = 'break';")
	require.NoError(t, err)
	require.Len(t, breakSessions, 1)
	assert.Nil(t, breakSessions[0].TaskID, "break session must always have nil task_id")

	// 4. Transition back to Work session
	m.nextSession()
	assert.Equal(t, config.WorkTask, m.currentTaskType)
	assert.NotNil(t, m.ActiveTask(), "active task must still be retained when returning to work session")
	assert.Equal(t, task.ID, m.ActiveTask().ID)

	// 5. Test complete shortcut ('c')
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(Model)
	assert.Nil(t, m.ActiveTask(), "active task must be nil after 'c' complete shortcut")

	completedTask, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, db.TaskCompleted, completedTask.Status)
	assert.NotNil(t, completedTask.CompletedAt)
}

func TestTimer_TaskPicker_OpeningAndSelection(t *testing.T) {
	database := newTestDB(t)
	cfg := testConfig()

	taskRepo := db.NewTaskRepo(database)
	t1, err := taskRepo.Create("Task Alpha", "First pending task")
	require.NoError(t, err)
	_, err = taskRepo.Create("Task Beta", "Second pending task")
	require.NoError(t, err)

	m := NewModelWithDB(config.WorkTask, cfg, database)
	m.handleWindowResize(tea.WindowSizeMsg{Width: 100, Height: 30})
	assert.Nil(t, m.ActiveTask())

	// Press 't' to open task picker
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = newModel.(Model)
	assert.Equal(t, ShowingTaskPicker, m.State())

	// View rendering when ShowingTaskPicker displays picker view
	pickerView := m.View()
	assert.Contains(t, pickerView, "Select Active Task")
	assert.Contains(t, pickerView, "Task Alpha")

	// Select Task Alpha
	newModel, _ = m.Update(taskpicker.TaskSelectedMsg{Task: *t1})
	m = newModel.(Model)
	assert.Equal(t, Running, m.State())
	require.NotNil(t, m.ActiveTask())
	assert.Equal(t, t1.ID, m.ActiveTask().ID)
	assert.Equal(t, "Task Alpha", m.ActiveTask().Title)
}
