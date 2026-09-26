package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*TaskRepo, *SessionRepo) {
	t.Helper()
	db := newTestDB(t)
	err := RunMigrations(db)
	require.NoError(t, err)

	return NewTaskRepo(db), NewSessionRepo(db)
}

func TestTaskRepo_CRUD(t *testing.T) {
	taskRepo, _ := setupTestDB(t)

	// 1. Create task
	task, err := taskRepo.Create("Implement Parser", "Lexer and tokens")
	require.NoError(t, err)
	assert.Greater(t, task.ID, 0)
	assert.Equal(t, "Implement Parser", task.Title)
	assert.Equal(t, "Lexer and tokens", task.Description)
	assert.Equal(t, TaskPending, task.Status)
	assert.Nil(t, task.CompletedAt)

	// 2. GetByID
	fetched, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, fetched.ID)
	assert.Equal(t, task.Title, fetched.Title)
	assert.Equal(t, task.Description, fetched.Description)
	assert.Equal(t, TaskPending, fetched.Status)

	// 3. Update task
	err = taskRepo.Update(task.ID, "Implement Parser & AST", "Updated description")
	require.NoError(t, err)

	updated, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Implement Parser & AST", updated.Title)
	assert.Equal(t, "Updated description", updated.Description)

	// 4. Complete task
	err = taskRepo.Complete(task.ID)
	require.NoError(t, err)

	completed, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskCompleted, completed.Status)
	assert.NotNil(t, completed.CompletedAt)

	// 5. Reopen task
	err = taskRepo.Reopen(task.ID)
	require.NoError(t, err)

	reopened, err := taskRepo.GetByID(task.ID)
	require.NoError(t, err)
	assert.Equal(t, TaskPending, reopened.Status)
	assert.Nil(t, reopened.CompletedAt)

	// 6. Delete task
	err = taskRepo.Delete(task.ID)
	require.NoError(t, err)

	_, err = taskRepo.GetByID(task.ID)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepo_Validation(t *testing.T) {
	taskRepo, _ := setupTestDB(t)

	// Empty title rejected
	_, err := taskRepo.Create("", "Some description")
	assert.ErrorIs(t, err, ErrEmptyTaskTitle)

	// Whitespace-only title rejected
	_, err = taskRepo.Create("   \t  \n ", "Some description")
	assert.ErrorIs(t, err, ErrEmptyTaskTitle)

	// Non-existent ID returns ErrTaskNotFound
	_, err = taskRepo.GetByID(99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)

	err = taskRepo.Update(99999, "Title", "Desc")
	assert.ErrorIs(t, err, ErrTaskNotFound)

	err = taskRepo.Complete(99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)

	err = taskRepo.Reopen(99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)

	err = taskRepo.Delete(99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepo_ListPendingAndAll(t *testing.T) {
	taskRepo, _ := setupTestDB(t)

	t1, err := taskRepo.Create("Task 1", "Desc 1")
	require.NoError(t, err)
	t2, err := taskRepo.Create("Task 2", "Desc 2")
	require.NoError(t, err)
	t3, err := taskRepo.Create("Task 3", "Desc 3")
	require.NoError(t, err)

	// Complete t2
	err = taskRepo.Complete(t2.ID)
	require.NoError(t, err)

	// ListPending should return t1 and t3
	pending, err := taskRepo.ListPending()
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, t1.ID, pending[0].ID)
	assert.Equal(t, t3.ID, pending[1].ID)

	// ListAll(true) should return all 3
	all, err := taskRepo.ListAll(true)
	require.NoError(t, err)
	require.Len(t, all, 3)

	// ListAll(false) should return 2 pending
	pendingOnly, err := taskRepo.ListAll(false)
	require.NoError(t, err)
	require.Len(t, pendingOnly, 2)
}

func TestTaskRepo_StatsAndAggregations(t *testing.T) {
	taskRepo, sessionRepo := setupTestDB(t)

	task, err := taskRepo.Create("Task with Sessions", "Testing metrics")
	require.NoError(t, err)

	now := time.Now().UTC()

	// Add 2 work sessions (25m each) and 1 break session (5m)
	err = sessionRepo.CreateSession(now.Add(-60*time.Minute), 25*time.Minute, WorkSession, &task.ID)
	require.NoError(t, err)
	err = sessionRepo.CreateSession(now.Add(-30*time.Minute), 5*time.Minute, BreakSession, &task.ID)
	require.NoError(t, err)
	err = sessionRepo.CreateSession(now.Add(-25*time.Minute), 25*time.Minute, WorkSession, &task.ID)
	require.NoError(t, err)

	// Also add an untracked work session
	err = sessionRepo.CreateSession(now, 25*time.Minute, WorkSession, nil)
	require.NoError(t, err)

	// Fetch task metrics
	stats, err := taskRepo.GetTaskStats(task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, stats.TaskID)
	assert.Equal(t, 2, stats.TotalPomodoros, "only work sessions should count toward task pomodoros")
	assert.Equal(t, 50*time.Minute, stats.TotalDuration, "only work session durations should be summed")

	// Verify ListPending also populates aggregates
	pending, err := taskRepo.ListPending()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, 2, pending[0].TotalPomodoros)
	assert.Equal(t, 50*time.Minute, pending[0].TotalDuration)
}

func TestTaskRepo_ForeignKeyDeleteSetNull(t *testing.T) {
	taskRepo, sessionRepo := setupTestDB(t)

	task, err := taskRepo.Create("Ephemeral Task", "To be deleted")
	require.NoError(t, err)

	// Log a session for this task
	err = sessionRepo.CreateSession(time.Now().UTC(), 25*time.Minute, WorkSession, &task.ID)
	require.NoError(t, err)

	// Verify session is linked
	sessions, err := sessionRepo.GetSessionsByTaskID(task.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	// Delete task
	err = taskRepo.Delete(task.ID)
	require.NoError(t, err)

	// Verify task is gone
	_, err = taskRepo.GetByID(task.ID)
	assert.ErrorIs(t, err, ErrTaskNotFound)

	// Verify session still exists in database, but its task_id became NULL
	var allSessions []Session
	err = taskRepo.db.Select(&allSessions, "SELECT id, task_id, type, duration, started_at FROM sessions;")
	require.NoError(t, err)
	require.Len(t, allSessions, 1)
	assert.Nil(t, allSessions[0].TaskID, "session task_id should be NULL after task deletion")
	assert.Equal(t, 25*time.Minute, allSessions[0].Duration, "session duration must be preserved")
}
