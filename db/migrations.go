package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type Migration struct {
	Version int
	Name    string
	Apply   func(tx *sqlx.Tx) error
}

var migrations = []Migration{
	{
		Version: 1,
		Name:    "0001_legacy_sessions",
		Apply: func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS sessions(
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					type TEXT NOT NULL,
					duration INTEGER NOT NULL,
					started_at TEXT NOT NULL
				);
			`)
			return err
		},
	},
	{
		Version: 2,
		Name:    "0002_create_tasks_and_fk",
		Apply: func(tx *sqlx.Tx) error {
			// 1. Create tasks table
			if _, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS tasks (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					title TEXT NOT NULL,
					description TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL DEFAULT 'pending',
					created_at TEXT NOT NULL,
					completed_at TEXT
				);
			`); err != nil {
				return fmt.Errorf("failed to create tasks table: %w", err)
			}

			// 2. Add task_id column to sessions if not already present
			hasCol, err := txColumnExists(tx, "sessions", "task_id")
			if err != nil {
				return fmt.Errorf("failed to inspect sessions columns: %w", err)
			}
			if !hasCol {
				if _, err := tx.Exec(`
					ALTER TABLE sessions ADD COLUMN task_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL;
				`); err != nil {
					return fmt.Errorf("failed to add task_id column to sessions: %w", err)
				}
			}

			// 3. Create indices
			indexes := []string{
				"CREATE INDEX IF NOT EXISTS idx_sessions_task_id ON sessions(task_id);",
				"CREATE INDEX IF NOT EXISTS idx_sessions_started_at ON sessions(started_at);",
				"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);",
			}
			for _, idx := range indexes {
				if _, err := tx.Exec(idx); err != nil {
					return fmt.Errorf("failed to create index %q: %w", idx, err)
				}
			}

			return nil
		},
	},
}

// RunMigrations runs all registered migrations sequentially within atomic transactions.
func RunMigrations(db *sqlx.DB) error {
	return RunMigrationsList(db, migrations)
}

// RunMigrationsList runs the given list of migrations against db.
func RunMigrationsList(db *sqlx.DB, migrationList []Migration) error {
	if db == nil {
		return fmt.Errorf("database connection cannot be nil")
	}

	// Ensure tracking table exists
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	applied := make(map[int]string)
	rows, err := db.Queryx("SELECT version, name FROM schema_migrations ORDER BY version ASC;")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var maxApplied int
	for rows.Next() {
		var v int
		var n string
		if err := rows.Scan(&v, &n); err != nil {
			return err
		}
		applied[v] = n
		if v > maxApplied {
			maxApplied = v
		}
	}

	latestSupported := 0
	if len(migrationList) > 0 {
		latestSupported = migrationList[len(migrationList)-1].Version
	}

	// Forward compatibility check: reject database version higher than binary supports
	if maxApplied > latestSupported {
		return fmt.Errorf("database schema version (%d) is newer than supported by this binary (%d)", maxApplied, latestSupported)
	}

	for _, m := range migrationList {
		if existingName, ok := applied[m.Version]; ok {
			if existingName != m.Name {
				return fmt.Errorf("migration version %d mismatch: recorded %q, expected %q", m.Version, existingName, m.Name)
			}
			continue
		}

		log.Printf("applying migration %d: %s", m.Version, m.Name)
		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}
	}

	return ValidateSchema(db)
}

func applyMigration(db *sqlx.DB, m Migration) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := m.Apply(tx); err != nil {
		return err
	}

	appliedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?);",
		m.Version, m.Name, appliedAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func txColumnExists(tx *sqlx.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			dfltVal  sql.NullString
			pk       int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltVal, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}

	return false, nil
}

// ValidateSchema verifies that core tables and columns are present in the database.
func ValidateSchema(db *sqlx.DB) error {
	requiredTables := []string{"schema_migrations", "sessions", "tasks"}
	for _, t := range requiredTables {
		var exists int
		err := db.Get(&exists, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?;", t)
		if err != nil || exists == 0 {
			return fmt.Errorf("required table %q is missing", t)
		}
	}

	// Verify task_id in sessions
	rows, err := db.Query("PRAGMA table_info(sessions);")
	if err != nil {
		return fmt.Errorf("failed to query sessions schema: %w", err)
	}
	defer rows.Close()

	hasTaskID := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dfltVal sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltVal, &pk); err != nil {
			return err
		}
		if name == "task_id" {
			hasTaskID = true
			break
		}
	}

	if !hasTaskID {
		return fmt.Errorf("required column 'task_id' is missing from 'sessions' table")
	}

	return nil
}
