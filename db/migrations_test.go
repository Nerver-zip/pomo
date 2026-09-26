package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	// Unique in-memory database per test
	db, err := sqlx.Open("sqlite", fmt.Sprintf("file:test_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestMigrations_FreshDatabase(t *testing.T) {
	db := newTestDB(t)

	err := RunMigrations(db)
	require.NoError(t, err)

	// Validate required tables exist
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM schema_migrations;")
	require.NoError(t, err)
	assert.Equal(t, len(migrations), count)

	var latestVersion int
	err = db.Get(&latestVersion, "SELECT MAX(version) FROM schema_migrations;")
	require.NoError(t, err)
	assert.Equal(t, 2, latestVersion)

	// Verify tasks and sessions tables exist
	var tables []string
	err = db.Select(&tables, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;")
	require.NoError(t, err)
	assert.Contains(t, tables, "schema_migrations")
	assert.Contains(t, tables, "sessions")
	assert.Contains(t, tables, "tasks")

	// Validate schema
	err = ValidateSchema(db)
	assert.NoError(t, err)
}

func TestMigrations_LegacyPomoDatabaseConvergence(t *testing.T) {
	db := newTestDB(t)

	// Setup exact legacy schema from upstream pomo
	_, err := db.Exec(`
		CREATE TABLE sessions(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			duration INTEGER NOT NULL,
			started_at TEXT NOT NULL
		);
		INSERT INTO sessions (type, duration, started_at)
		VALUES ('work', 1500000000000, '2026-09-01T10:00:00Z');
		INSERT INTO sessions (type, duration, started_at)
		VALUES ('break', 300000000000, '2026-09-01T10:25:00Z');
	`)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrations(db)
	require.NoError(t, err)

	// Verify legacy sessions preserved
	type LegacySession struct {
		ID        int     `db:"id"`
		Type      string  `db:"type"`
		Duration  int64   `db:"duration"`
		StartedAt string  `db:"started_at"`
		TaskID    *int    `db:"task_id"`
	}

	var sessions []LegacySession
	err = db.Select(&sessions, "SELECT id, type, duration, started_at, task_id FROM sessions ORDER BY id ASC;")
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.Equal(t, "work", sessions[0].Type)
	assert.Equal(t, int64(1500000000000), sessions[0].Duration)
	assert.Nil(t, sessions[0].TaskID, "legacy session task_id should be NULL")

	assert.Equal(t, "break", sessions[1].Type)
	assert.Equal(t, int64(300000000000), sessions[1].Duration)
	assert.Nil(t, sessions[1].TaskID, "legacy session task_id should be NULL")

	// Verify tasks table is available
	_, err = db.Exec("INSERT INTO tasks (title, created_at) VALUES ('New Task', '2026-09-01T11:00:00Z');")
	assert.NoError(t, err)

	assert.NoError(t, ValidateSchema(db))
}

func TestMigrations_Idempotency(t *testing.T) {
	db := newTestDB(t)

	// Run once
	err := RunMigrations(db)
	require.NoError(t, err)

	// Run twice
	err = RunMigrations(db)
	require.NoError(t, err)

	// Run third time
	err = RunMigrations(db)
	require.NoError(t, err)

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM schema_migrations;")
	require.NoError(t, err)
	assert.Equal(t, len(migrations), count)
}

func TestMigrations_FutureSchemaRejected(t *testing.T) {
	db := newTestDB(t)

	// Setup schema_migrations with a future version
	_, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);
		INSERT INTO schema_migrations (version, name, applied_at)
		VALUES (999, 'future_migration_beyond_supported', '2030-01-01T00:00:00Z');
	`)
	require.NoError(t, err)

	err = RunMigrations(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "newer than supported by this binary")
}
