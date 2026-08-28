package discovery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/natelindev/agent-sessions-tui/internal/session"
)

const cacheFileName = "sessions-v1.sqlite3"

type cachedFile struct {
	provider    session.Provider
	path        string
	size        int64
	modTimeNano int64
	session     session.Session
	valid       bool
}

func defaultCachePath(home string) string {
	currentHome, homeErr := os.UserHomeDir()
	cacheRoot, cacheErr := os.UserCacheDir()
	if homeErr == nil && cacheErr == nil && filepath.Clean(home) == filepath.Clean(currentHome) {
		return filepath.Join(cacheRoot, "agent-sessions-tui", cacheFileName)
	}
	return filepath.Join(home, ".cache", "agent-sessions-tui", cacheFileName)
}

func openFileSessionCache(path string) (*sql.DB, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create cache: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close cache: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("secure cache: %w", err)
	}

	query := url.Values{}
	query.Set("mode", "rwc")
	query.Add("_pragma", "busy_timeout(2000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: query.Encode()}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open cache: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS file_sessions (
			provider TEXT NOT NULL,
			path TEXT NOT NULL,
			size INTEGER NOT NULL,
			mtime_ns INTEGER NOT NULL,
			session_json BLOB NOT NULL,
			PRIMARY KEY (provider, path)
		)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize cache: %w", err)
	}
	return db, nil
}

func loadCachedFiles(db *sql.DB) (map[string]cachedFile, error) {
	entries := make(map[string]cachedFile)
	if db == nil {
		return entries, nil
	}
	rows, err := db.Query(`SELECT provider, path, size, mtime_ns, session_json FROM file_sessions`)
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var provider, path string
		var size, modTimeNano int64
		var encoded []byte
		if err := rows.Scan(&provider, &path, &size, &modTimeNano, &encoded); err != nil {
			continue
		}
		entry := cachedFile{
			provider:    session.Provider(provider),
			path:        path,
			size:        size,
			modTimeNano: modTimeNano,
		}
		entry.valid = json.Unmarshal(encoded, &entry.session) == nil
		entries[cacheKey(entry.provider, entry.path)] = entry
	}
	return entries, rows.Err()
}

func updateCachedFiles(db *sql.DB, previous map[string]cachedFile, current map[string]cachedFile, changed []cachedFile) error {
	if db == nil {
		return nil
	}
	removed := make([]cachedFile, 0)
	for key, entry := range previous {
		if _, exists := current[key]; !exists {
			removed = append(removed, entry)
		}
	}
	if len(changed) == 0 && len(removed) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin cache update: %w", err)
	}
	defer tx.Rollback()
	for _, entry := range changed {
		encoded, err := json.Marshal(entry.session)
		if err != nil {
			return fmt.Errorf("encode cached session: %w", err)
		}
		if _, err := tx.Exec(`
			INSERT INTO file_sessions (provider, path, size, mtime_ns, session_json)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(provider, path) DO UPDATE SET
				size = excluded.size,
				mtime_ns = excluded.mtime_ns,
				session_json = excluded.session_json`,
			entry.provider, entry.path, entry.size, entry.modTimeNano, encoded,
		); err != nil {
			return fmt.Errorf("write cached session: %w", err)
		}
	}
	for _, entry := range removed {
		if _, err := tx.Exec(`DELETE FROM file_sessions WHERE provider = ? AND path = ?`, entry.provider, entry.path); err != nil {
			return fmt.Errorf("remove cached session: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cache update: %w", err)
	}
	return nil
}

func cacheKey(provider session.Provider, path string) string {
	return string(provider) + "\x00" + path
}
