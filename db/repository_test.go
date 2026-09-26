package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRepo_CreateAndStats(t *testing.T) {
	db := newTestDB(t)
	err := RunMigrations(db)
	require.NoError(t, err)

	sessionRepo := NewSessionRepo(db)
	taskRepo := NewTaskRepo(db)

	task, err := taskRepo.Create("Repository Task", "Testing session repo")
	require.NoError(t, err)

	now := time.Now().UTC()

	// 1. Create sessions (tracked and untracked)
	err = sessionRepo.CreateSession(now.Add(-2*time.Hour), 25*time.Minute, WorkSession, &task.ID)
	require.NoError(t, err)

	err = sessionRepo.CreateSession(now.Add(-90*time.Minute), 5*time.Minute, BreakSession, nil)
	require.NoError(t, err)

	err = sessionRepo.CreateSession(now.Add(-1*time.Hour), 25*time.Minute, WorkSession, nil)
	require.NoError(t, err)

	// 2. Test GetAllTimeStats
	allTime, err := sessionRepo.GetAllTimeStats()
	require.NoError(t, err)
	assert.Equal(t, 3, allTime.TotalSessions)
	assert.Equal(t, 50*time.Minute, allTime.TotalWorkDuration)
	assert.Equal(t, 5*time.Minute, allTime.TotalBreakDuration)

	// 3. Test GetSessionsByTaskID
	taskSessions, err := sessionRepo.GetSessionsByTaskID(task.ID)
	require.NoError(t, err)
	require.Len(t, taskSessions, 1)
	assert.Equal(t, &task.ID, taskSessions[0].TaskID)
	assert.Equal(t, string(WorkSession), taskSessions[0].Type)

	// 4. Test GetWeeklyStats
	weekly, err := sessionRepo.GetWeeklyStats()
	require.NoError(t, err)
	assert.Len(t, weekly, 7) // exactly 7 normalized days

	// 5. Test GetStreakStats
	streak, err := sessionRepo.GetStreakStats()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, streak.Current, 1)
	assert.GreaterOrEqual(t, streak.Best, 1)
}
