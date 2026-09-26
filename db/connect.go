// Package db handles the database connection, migrations, and initialization.
package db

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Nerver-zip/pomo-tasker/config"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const DBFile = config.AppName + ".db"

// Connect connects to the SQLite database,
// creates the necessary directories, imports legacy pomo data if needed,
// configures SQLite pragmas (WAL, foreign_keys, busy_timeout),
// and performs migrations.
func Connect() (*sqlx.DB, error) {
	dbDir, err := getDBDir()
	if err != nil {
		log.Println("failed to get db path:", err)
		return nil, err
	}

	// create the db directory if it doesn't exist
	if err = os.MkdirAll(dbDir, 0o755); err != nil {
		log.Println("failed to create db directory:", err)
		return nil, err
	}

	dbPath := filepath.Join(dbDir, DBFile)

	// One-time automatic import of legacy pomo.db if pomo-tasker.db does not exist
	importLegacyDBIfNeeded(dbPath)

	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		log.Println("failed to connect to the db:", err)
		return nil, err
	}
	log.Println("connected to the db")

	if err = db.Ping(); err != nil {
		log.Println("failed to ping the db:", err)
		return nil, err
	}
	log.Println("pinged the db")

	// Limit the number of open connections to 1 for SQLite
	db.SetMaxOpenConns(1)

	// Configure pragmas for concurrency, safety, and referential integrity
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("failed to set pragma %q: %v", pragma, err)
		}
	}

	// Run sequential migrations
	if err = RunMigrations(db); err != nil {
		log.Println("failed to migrate the db:", err)
		return nil, err
	}

	return db, nil
}

// importLegacyDBIfNeeded checks if pomo-tasker.db is missing and copies legacy pomo.db if found.
func importLegacyDBIfNeeded(destPath string) {
	if _, err := os.Stat(destPath); err == nil {
		return // dest already exists
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	var candidates []string
	if xdgState := os.Getenv("XDG_STATE_HOME"); xdgState != "" {
		candidates = append(candidates, filepath.Join(xdgState, "pomo", "pomo.db"))
	}
	candidates = append(candidates, filepath.Join(home, ".local", "state", "pomo", "pomo.db"))

	for _, legacyPath := range candidates {
		if info, err := os.Stat(legacyPath); err == nil && !info.IsDir() {
			log.Printf("importing legacy pomo database from %s to %s", legacyPath, destPath)
			if err := copyFile(legacyPath, destPath); err != nil {
				log.Printf("failed to copy legacy pomo database: %v", err)
			}
			return
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// returns the path to the db directory respecting XDG specification
func getDBDir() (string, error) {
	var baseDir string

	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		baseDir = xdg
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		home := os.Getenv("HOME")
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return "", errors.New("$HOME is not defined")
			}
		}

		baseDir = filepath.Join(home, ".local", "state")
	} else {
		// on other OSes, use the standard user config directory
		var err error
		baseDir, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(baseDir, config.AppName), nil
}
