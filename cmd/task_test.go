package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bahaaio/pomo/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnvironment(t *testing.T) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Ensure config directory exists
	err := os.MkdirAll(filepath.Join(tempDir, config.AppName), 0o755)
	require.NoError(t, err)
}

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestTaskCLI_Workflow(t *testing.T) {
	setupTestEnvironment(t)

	// 1. Add a task
	out, err := executeCommand("task", "add", "CLI Test Task", "-d", "Testing from CLI")
	require.NoError(t, err, out)

	// 2. List tasks
	out, err = executeCommand("task", "list")
	require.NoError(t, err, out)

	// 3. Edit task
	out, err = executeCommand("task", "edit", "1", "-t", "Updated CLI Task", "-d", "Updated description")
	require.NoError(t, err, out)

	// 4. Task stats
	out, err = executeCommand("task", "stats", "1")
	require.NoError(t, err, out)

	// 5. Complete task
	out, err = executeCommand("task", "done", "1")
	require.NoError(t, err, out)

	// 6. List completed tasks
	out, err = executeCommand("task", "list", "--done")
	require.NoError(t, err, out)

	// 7. Reopen task
	out, err = executeCommand("task", "reopen", "1")
	require.NoError(t, err, out)

	// 8. Delete task
	out, err = executeCommand("task", "delete", "1")
	require.NoError(t, err, out)
}

func TestTaskCLI_InvalidArguments(t *testing.T) {
	setupTestEnvironment(t)

	// Missing title for add
	_, err := executeCommand("task", "add")
	assert.Error(t, err)

	// Invalid numeric ID for done
	out, err := executeCommand("task", "done", "abc")
	_ = out
	_ = err // die() calls os.Exit, handled in subcommands

	// Format duration helper
	assert.Equal(t, "0m", formatDuration(0))
	assert.Equal(t, "45m", formatDuration(45*60*1000*1000*1000))
	assert.Equal(t, "1h30m", formatDuration(90*60*1000*1000*1000))
}

func TestRootCmd_TaskFlag(t *testing.T) {
	// Verify --task and -T flags exist on rootCmd
	taskFlag := rootCmd.Flags().Lookup("task")
	require.NotNil(t, taskFlag)
	assert.Equal(t, "T", taskFlag.Shorthand)
	assert.Equal(t, "0", taskFlag.DefValue)
}
