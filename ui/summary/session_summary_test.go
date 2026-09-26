package summary_test

import (
	"testing"
	"time"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/Nerver-zip/pomo-tasker/ui/summary"
	"github.com/stretchr/testify/assert"
)

func TestSessionSummary_UntrackedSession(t *testing.T) {
	s := summary.SessionSummary{}
	assert.Empty(t, s.Render(), "empty summary should render nothing")

	s.AddSession(config.WorkTask, 25*time.Minute)
	s.AddSession(config.BreakTask, 5*time.Minute)

	rendered := s.Render()
	assert.Contains(t, rendered, "Session Summary:")
	assert.Contains(t, rendered, "Work : 25m0s (1 session)")
	assert.Contains(t, rendered, "Break: 5m0s (1 session)")
	assert.Contains(t, rendered, "Total: 30m0s")
	assert.NotContains(t, rendered, "Focused Task", "untracked session must not display Focused Task line")
	assert.Contains(t, rendered, "work")
}

func TestSessionSummary_TrackedTaskSession(t *testing.T) {
	s := summary.SessionSummary{}
	s.SetFocusedTask(42, "Refactor database engine")
	s.AddSession(config.WorkTask, 25*time.Minute)
	s.AddTaskSession(42, "Refactor database engine", 25*time.Minute)

	rendered := s.Render()
	assert.Contains(t, rendered, "Session Summary:")
	assert.Contains(t, rendered, "Focused Task: #42 Refactor database engine (1 session · 25m0s)")
	assert.Contains(t, rendered, "Work : 25m0s (1 session)")

	// Second session on the same task
	s.AddSession(config.WorkTask, 25*time.Minute)
	s.AddTaskSession(42, "Refactor database engine", 25*time.Minute)

	rendered2 := s.Render()
	assert.Contains(t, rendered2, "Focused Task: #42 Refactor database engine (2 sessions · 50m0s)")
	assert.Contains(t, rendered2, "Work : 50m0s (2 sessions)")
}

func TestSessionSummary_DatabaseUnavailable(t *testing.T) {
	s := summary.SessionSummary{}
	s.AddSession(config.WorkTask, 10*time.Minute)
	s.SetDatabaseUnavailable()

	rendered := s.Render()
	assert.Contains(t, rendered, "Not saved (database unavailable)")
}
