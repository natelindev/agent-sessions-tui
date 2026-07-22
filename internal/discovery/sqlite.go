package discovery

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/natelindev/agent-sessions-tui/internal/resume"
	"github.com/natelindev/agent-sessions-tui/internal/session"
	_ "modernc.org/sqlite"
)

func openReadOnly(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=busy_timeout(2000)"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func scanOpenCodeDB(path string) ([]session.Session, error) {
	db, err := openReadOnly(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, title, directory, COALESCE(model, ''), time_created, time_updated FROM session`)
	withModel := true
	if err != nil {
		rows, err = db.Query(`SELECT id, title, directory, time_created, time_updated FROM session`)
		withModel = false
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read session index: %w", err)
	}
	defer rows.Close()

	var out []session.Session
	for rows.Next() {
		var id, title, cwd, model string
		var created, updated int64
		var scanErr error
		if withModel {
			scanErr = rows.Scan(&id, &title, &cwd, &model, &created, &updated)
		} else {
			scanErr = rows.Scan(&id, &title, &cwd, &created, &updated)
		}
		if scanErr != nil {
			continue
		}
		item := session.Session{
			Provider:  session.OpenCode,
			ID:        id,
			Title:     cleanTitle(title),
			CWD:       cwd,
			Project:   filepath.Base(filepath.Clean(cwd)),
			Model:     model,
			Path:      path,
			StartedAt: epochTime(created),
			UpdatedAt: epochTime(updated),
		}
		item.Resume = resume.For(item.Provider, item.ID, item.Path)
		item.SearchText = strings.ToLower(strings.Join([]string{id, title, cwd, item.Project, model, "opencode"}, " "))
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanHermesDB(path string) ([]session.Session, error) {
	db, err := openReadOnly(path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, COALESCE(title, ''), COALESCE(cwd, ''), COALESCE(model, ''), started_at, COALESCE(ended_at, started_at) FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("cannot read session index: %w", err)
	}
	defer rows.Close()

	var out []session.Session
	for rows.Next() {
		var id, title, cwd, model string
		var started, updated float64
		if err := rows.Scan(&id, &title, &cwd, &model, &started, &updated); err != nil {
			continue
		}
		item := session.Session{
			Provider:  session.Hermes,
			ID:        id,
			Title:     cleanTitle(title),
			CWD:       cwd,
			Project:   filepath.Base(filepath.Clean(cwd)),
			Model:     model,
			Path:      path,
			StartedAt: epochFloat(started),
			UpdatedAt: epochFloat(updated),
		}
		item.Resume = resume.For(item.Provider, item.ID, item.Path)
		item.SearchText = strings.ToLower(strings.Join([]string{id, title, cwd, item.Project, model, "hermes"}, " "))
		out = append(out, item)
	}
	return out, rows.Err()
}

func epochTime(value int64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

func epochFloat(value float64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	seconds := int64(value)
	nanos := int64((value - float64(seconds)) * float64(time.Second))
	return time.Unix(seconds, nanos)
}

func isMissingDB(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
