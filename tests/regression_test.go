package tests_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nerver-zip/pomo-tasker/cmd"
	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/db"
	"github.com/Nerver-zip/pomo-tasker/ui"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestEnvironment(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	err := os.MkdirAll(filepath.Join(tempDir, config.AppName), 0o755)
	require.NoError(t, err)
	return tempDir
}

func executeCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	err := cmd.ExecuteWithArgs(args, buf)
	return buf.String(), err
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

func TestRegression_UntrackedFlow(t *testing.T) {
	setupTestEnvironment(t)

	database, err := db.Connect()
	require.NoError(t, err)
	defer database.Close()

	// Initial model without any task (zero friction)
	m := ui.NewModelWithDB(config.WorkTask, testConfig(), database)
	assert.Nil(t, m.ActiveTask(), "untracked session must have nil active task")

	// View rendering has no task header
	view := m.View()
	assert.NotContains(t, view, "task: ")

	// Summary on exit with untracked session
	summary := m.GetSessionSummary()
	summary.AddSession(config.WorkTask, 25*time.Minute)
	summary.AddSession(config.BreakTask, 5*time.Minute)

	rendered := summary.Render()
	assert.Contains(t, rendered, "Work : 25m0s (1 session)")
	assert.Contains(t, rendered, "Break: 5m0s (1 session)")
	assert.NotContains(t, rendered, "Focused Task", "untracked flow must never render Focused Task line")
}

func TestRegression_EndToEndTaskFlow(t *testing.T) {
	setupTestEnvironment(t)

	database, err := db.Connect()
	require.NoError(t, err)
	defer database.Close()

	taskRepo := db.NewTaskRepo(database)

	// 1. Create a task
	task, err := taskRepo.Create("End-to-End Test Task", "Verify full lifecycle flow")
	require.NoError(t, err)
	assert.Equal(t, db.TaskPending, task.Status)

	// 2. Start session with task
	m := ui.NewModelWithDB(config.WorkTask, testConfig(), database)
	m.SetInitialTask(task)
	require.NotNil(t, m.ActiveTask())
	assert.Equal(t, task.ID, m.ActiveTask().ID)

	// 3. Complete work session and verify persistence
	sessionRepo := db.NewSessionRepo(database)
	err = sessionRepo.CreateSession(time.Now(), 25*time.Minute, db.WorkSession, &task.ID)
	require.NoError(t, err)

	// Update stats
	taskStats, err := taskRepo.GetTaskStats(task.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, taskStats.TotalPomodoros)
	assert.Equal(t, 25*time.Minute, taskStats.TotalDuration)

	// 4. Verify exit summary
	summary := m.GetSessionSummary()
	summary.AddSession(config.WorkTask, 25*time.Minute)
	summary.AddTaskSession(task.ID, task.Title, 25*time.Minute)

	rendered := summary.Render()
	assert.Contains(t, rendered, fmt.Sprintf("Focused Task: #%d End-to-End Test Task (1 session · 25m0s)", task.ID))

	// 5. Complete task
	err = taskRepo.Complete(task.ID)
	require.NoError(t, err)

	completedTask, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, db.TaskCompleted, completedTask.Status)
	assert.NotNil(t, completedTask.CompletedAt)
}

func TestRegression_LegacyDatabaseMigration(t *testing.T) {
	tempDir := t.TempDir()
	dbDir := filepath.Join(tempDir, config.AppName)
	err := os.MkdirAll(dbDir, 0o755)
	require.NoError(t, err)

	dbPath := filepath.Join(dbDir, config.AppName+".db")

	// 1. Manually create legacy v1.2.1 pomo database
	legacyDB, err := sqlx.Open("sqlite", dbPath)
	require.NoError(t, err)

	_, err = legacyDB.Exec(`
		CREATE TABLE sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			duration INTEGER NOT NULL,
			started_at DATETIME NOT NULL
		);
	`)
	require.NoError(t, err)

	// Insert legacy sessions
	_, err = legacyDB.Exec(`
		INSERT INTO sessions (type, duration, started_at) VALUES ('work', 1500000000000, '2026-01-01T10:00:00Z');
		INSERT INTO sessions (type, duration, started_at) VALUES ('break', 300000000000, '2026-01-01T10:25:00Z');
	`)
	require.NoError(t, err)
	_ = legacyDB.Close()

	// 2. Open via pomo-tasker and apply migrations
	upgradedDB, err := sqlx.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer upgradedDB.Close()

	err = db.RunMigrations(upgradedDB)
	require.NoError(t, err)

	// 3. Verify legacy sessions are preserved
	var legacyCount int
	err = upgradedDB.Get(&legacyCount, "SELECT COUNT(*) FROM sessions;")
	require.NoError(t, err)
	assert.Equal(t, 2, legacyCount)

	// 4. Verify new task features work on the migrated database
	taskRepo := db.NewTaskRepo(upgradedDB)
	newTask, err := taskRepo.Create("Post-migration Task", "Works seamlessly")
	require.NoError(t, err)
	assert.Greater(t, newTask.ID, 0)

	sessionRepo := db.NewSessionRepo(upgradedDB)
	err = sessionRepo.CreateSession(time.Now(), 25*time.Minute, db.WorkSession, &newTask.ID)
	require.NoError(t, err)

	// Verify foreign key relationship
	var totalSessions int
	err = upgradedDB.Get(&totalSessions, "SELECT COUNT(*) FROM sessions;")
	require.NoError(t, err)
	assert.Equal(t, 3, totalSessions)

	var linkedSessionCount int
	err = upgradedDB.Get(&linkedSessionCount, "SELECT COUNT(*) FROM sessions WHERE task_id IS NOT NULL;")
	require.NoError(t, err)
	assert.Equal(t, 1, linkedSessionCount)
}

func TestRegression_CLIWorkflow(t *testing.T) {
	setupTestEnvironment(t)

	// 1. pomo task add
	out, err := executeCLI(t, "task", "add", "CLI Regression Task", "-d", "Validating CLI suite")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Created task #1: \"CLI Regression Task\"")

	// 2. pomo task list
	out, err = executeCLI(t, "task", "list")
	require.NoError(t, err, out)
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "CLI Regression Task")

	// 3. pomo task edit
	out, err = executeCLI(t, "task", "edit", "1", "-t", "CLI Regression Task Edited")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Updated task #1: \"CLI Regression Task Edited\"")

	// 4. pomo stats -T 1
	out, err = executeCLI(t, "stats", "-T", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Task #1: CLI Regression Task Edited")
	assert.Contains(t, out, "Status:       pending")

	// 5. pomo task done 1
	out, err = executeCLI(t, "task", "done", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Completed task #1")

	// 6. pomo task reopen 1
	out, err = executeCLI(t, "task", "reopen", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Reopened task #1")

	// 7. pomo task delete 1
	out, err = executeCLI(t, "task", "delete", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Deleted task #1")

	// 8. pomo task list (should now be empty)
	out, err = executeCLI(t, "task", "list")
	require.NoError(t, err, out)
	assert.Contains(t, out, "No pending tasks")
}
