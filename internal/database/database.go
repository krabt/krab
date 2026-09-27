package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/glebarez/go-sqlite"
)

var (
	once    sync.Once
	shared  *sql.DB
	openErr error
)

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".krab", "krab.db"), nil
}

// EnsureDir creates Krab's per-user data directory before any subsystem
// needs it. The same home-relative location is used on macOS, Linux, and
// Windows.
func EnsureDir() error {
	path, err := Path()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func Open() (*sql.DB, error) {
	once.Do(func() {
		path, err := Path()
		if err != nil {
			openErr = err
			return
		}
		if err := EnsureDir(); err != nil {
			openErr = err
			return
		}
		shared, openErr = sql.Open("sqlite", path)
		if openErr != nil {
			return
		}
		shared.SetMaxOpenConns(1)
		_, openErr = shared.Exec(`
			PRAGMA journal_mode=DELETE;
			PRAGMA busy_timeout=5000;
			CREATE TABLE IF NOT EXISTS app_data (
				key TEXT PRIMARY KEY,
				value BLOB NOT NULL,
				updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
			);`)
		if openErr == nil {
			_ = os.Chmod(path, 0o600)
		}
	})
	return shared, openErr
}

func Get(key string) ([]byte, bool, error) {
	db, err := Open()
	if err != nil {
		return nil, false, err
	}
	var value []byte
	err = db.QueryRow(`SELECT value FROM app_data WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return value, err == nil, err
}

func Set(key string, value []byte) error {
	db, err := Open()
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO app_data(key, value, updated_at) VALUES(?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`, key, value)
	return err
}
