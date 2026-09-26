package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsCmd_TaskFlag(t *testing.T) {
	setupTestEnvironment(t)

	// Add a task
	out, err := executeCommand("task", "add", "Stats Test Task", "-d", "Metrics testing")
	require.NoError(t, err, out)

	// Execute stats --task 1
	out, err = executeCommand("stats", "--task", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Task #1: Stats Test Task")
	assert.Contains(t, out, "Status:       pending")
	assert.Contains(t, out, "Pomodoros:    0 completed")
	assert.Contains(t, out, "Total Focus:  0m")

	// Execute stats with shorthand -T 1
	out, err = executeCommand("stats", "-T", "1")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Task #1: Stats Test Task")

	// Non-existent task returns error
	_, err = executeCommand("stats", "-T", "999")
	assert.Error(t, err)
}
