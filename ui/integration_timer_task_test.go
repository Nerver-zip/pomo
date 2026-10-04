package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	"github.com/Bahaaio/pomo/db"
	"github.com/Bahaaio/pomo/ui/taskpicker"
	"github.com/charmbracelet/bubbles/timer"
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

func TestTimer_SeparateTimersAndSwitchingWithoutSharingTime(t *testing.T) {
	database := newTestDB(t)
	cfg := testConfig()

	taskRepo := db.NewTaskRepo(database)
	t1, err := taskRepo.Create("Task One", "First task with dedicated timer")
	require.NoError(t, err)
	t2, err := taskRepo.Create("Task Two", "Second task with dedicated timer")
	require.NoError(t, err)

	m := NewModelWithDB(config.WorkTask, cfg, database)
	m.handleWindowResize(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.SetInitialTask(t1)

	// 1. Work 10 minutes on Task One
	m.elapsed = 10 * time.Minute
	m.timer.Timeout = 15 * time.Minute

	// 2. Switch to Task Two
	newModel, _ := m.Update(taskpicker.TaskSelectedMsg{Task: *t2})
	m = newModel.(Model)

	// Verify Task One's 10m was committed to the DB
	var t1Sessions []db.Session
	err = database.Select(&t1Sessions, "SELECT id, task_id, duration FROM sessions WHERE task_id = ?;", t1.ID)
	require.NoError(t, err)
	require.Len(t, t1Sessions, 1)
	assert.Equal(t, 10*time.Minute, t1Sessions[0].Duration)

	// Verify Task Two starts with its OWN separate timer (25:00, 0 elapsed)
	require.NotNil(t, m.ActiveTask())
	assert.Equal(t, t2.ID, m.ActiveTask().ID)
	assert.Equal(t, 0*time.Minute, m.elapsed, "Task Two must start with 0 elapsed time")
	assert.Equal(t, 25*time.Minute, m.duration)
	assert.Equal(t, 25*time.Minute, m.timer.Timeout, "Task Two must have its own fresh 25m countdown")

	// 3. Work 5 minutes on Task Two
	m.elapsed = 5 * time.Minute
	m.timer.Timeout = 20 * time.Minute

	// 4. Switch back to Task One
	newModel, _ = m.Update(taskpicker.TaskSelectedMsg{Task: *t1})
	m = newModel.(Model)

	// Verify Task Two's 5m was committed to the DB
	var t2Sessions []db.Session
	err = database.Select(&t2Sessions, "SELECT id, task_id, duration FROM sessions WHERE task_id = ?;", t2.ID)
	require.NoError(t, err)
	require.Len(t, t2Sessions, 1)
	assert.Equal(t, 5*time.Minute, t2Sessions[0].Duration)

	// Verify Task One's timer was restored to 15m remaining!
	require.NotNil(t, m.ActiveTask())
	assert.Equal(t, t1.ID, m.ActiveTask().ID)
	assert.Equal(t, 10*time.Minute, m.elapsed, "Task One's elapsed progress must be restored")
	assert.Equal(t, 15*time.Minute, m.timer.Timeout, "Task One must resume with 15m remaining")

	// 5. Work remaining 15 minutes on Task One and complete
	m.elapsed = 25 * time.Minute
	m.recordSession()

	// 6. Verify full DB history: Task One = 25m total, Task Two = 5m total. No shared time!
	stats1, err := taskRepo.GetTaskStats(t1.ID)
	require.NoError(t, err)
	assert.Equal(t, 25*time.Minute, stats1.TotalDuration)

	stats2, err := taskRepo.GetTaskStats(t2.ID)
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, stats2.TotalDuration)
}

func TestTimer_NoAcceleratedTicksOnPauseResumeOrPicker(t *testing.T) {
	database := newTestDB(t)
	cfg := testConfig()

	m := NewModelWithDB(config.WorkTask, cfg, database)
	m.handleWindowResize(tea.WindowSizeMsg{Width: 100, Height: 30})

	// 1. Initial tick decrements by 1s
	newModel, _ := m.Update(timer.TickMsg{ID: m.timer.ID()})
	m = newModel.(Model)
	assert.Equal(t, 1*time.Second, m.elapsed)
	assert.Equal(t, 25*time.Minute-1*time.Second, m.timer.Timeout)

	// 2. Pause with Space
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)
	assert.Equal(t, Paused, m.State())

	// 3. Ticks arriving while paused must be rejected (not advanced)
	newModel, _ = m.Update(timer.TickMsg{ID: m.timer.ID()})
	m = newModel.(Model)
	assert.Equal(t, 1*time.Second, m.elapsed, "elapsed must not advance while paused")
	assert.Equal(t, 25*time.Minute-1*time.Second, m.timer.Timeout)

	// 4. Resume with Space
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)
	assert.Equal(t, Running, m.State())

	// 5. Open Task Picker with 't'
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = newModel.(Model)
	assert.Equal(t, ShowingTaskPicker, m.State())

	// 6. Ticks arriving while in picker must be rejected (timer is stopped)
	newModel, _ = m.Update(timer.TickMsg{ID: m.timer.ID()})
	m = newModel.(Model)
	assert.Equal(t, 1*time.Second, m.elapsed, "elapsed must not advance while in picker")

	// 7. Close picker with ClosePickerMsg
	newModel, _ = m.Update(taskpicker.ClosePickerMsg{})
	m = newModel.(Model)
	assert.Equal(t, Running, m.State())

	// 8. Subsequent valid tick advances exactly 1s, never 2x
	newModel, _ = m.Update(timer.TickMsg{ID: m.timer.ID()})
	m = newModel.(Model)
	assert.Equal(t, 2*time.Second, m.elapsed)
	assert.Equal(t, 25*time.Minute-2*time.Second, m.timer.Timeout)

	// 9. Stale tick from mismatched timer ID is rejected
	newModel, _ = m.Update(timer.TickMsg{ID: 999999})
	m = newModel.(Model)
	assert.Equal(t, 2*time.Second, m.elapsed, "stale tick from different timer ID must be ignored")
}
