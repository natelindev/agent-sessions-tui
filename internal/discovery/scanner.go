package discovery

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/natelindev/agent-sessions-tui/internal/resume"
	"github.com/natelindev/agent-sessions-tui/internal/session"
)

type Result struct {
	Sessions []session.Session
	Warnings []string
}

type Scanner struct {
	Home      string
	CachePath string
}

type fileCandidate struct {
	provider session.Provider
	path     string
}

func New(home string) *Scanner {
	return &Scanner{Home: home, CachePath: defaultCachePath(home)}
}

// Cached returns the last normalized file-backed index plus current database-backed
// sessions. Callers can render this snapshot while Scan validates the file cache.
func (s *Scanner) Cached() (Result, bool) {
	cacheDB, err := openFileSessionCache(s.CachePath)
	if err != nil {
		return Result{Warnings: []string{"Cache: " + err.Error()}}, false
	}
	if cacheDB == nil {
		return Result{}, false
	}
	defer cacheDB.Close()
	entries, err := loadCachedFiles(cacheDB)
	if err != nil {
		return Result{Warnings: []string{"Cache: " + err.Error()}}, false
	}

	result := Result{Sessions: make([]session.Session, 0, len(entries)+128)}
	for _, entry := range entries {
		if !entry.valid {
			continue
		}
		entry.session.Resume = resume.For(entry.session.Provider, entry.session.ID, entry.session.Path)
		result.Sessions = append(result.Sessions, entry.session)
	}
	if len(result.Sessions) == 0 {
		return Result{}, false
	}
	s.scanDatabases(&result)
	result.Sessions = dedupe(result.Sessions)
	session.SortNewest(result.Sessions)
	sort.Strings(result.Warnings)
	return result, true
}

func (s *Scanner) Scan(ctx context.Context) Result {
	result := Result{}
	cacheDB, cacheErr := openFileSessionCache(s.CachePath)
	if cacheDB != nil {
		defer cacheDB.Close()
	}
	cached := make(map[string]cachedFile)
	if cacheErr == nil {
		cached, cacheErr = loadCachedFiles(cacheDB)
	}
	if cacheErr != nil {
		result.Warnings = append(result.Warnings, "Cache: "+cacheErr.Error())
	}

	candidates := s.discoverFiles()
	workers := min(max(runtime.NumCPU(), 4), 16)
	jobs := make(chan fileCandidate)
	items := make(chan cachedFile)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for candidate := range jobs {
				if ctx.Err() != nil {
					return
				}
				info, err := os.Stat(candidate.path)
				if err != nil || !info.Mode().IsRegular() {
					continue
				}
				key := cacheKey(candidate.provider, candidate.path)
				entry, hit := cached[key]
				if hit && entry.valid && entry.size == info.Size() && entry.modTimeNano == info.ModTime().UnixNano() {
					entry.session.Resume = resume.For(entry.session.Provider, entry.session.ID, entry.session.Path)
				} else {
					item, ok := parseJSONSession(candidate)
					if !ok {
						continue
					}
					entry = cachedFile{
						provider:    candidate.provider,
						path:        candidate.path,
						size:        info.Size(),
						modTimeNano: info.ModTime().UnixNano(),
						session:     item,
						valid:       true,
					}
				}
				select {
				case items <- entry:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, candidate := range candidates {
			select {
			case jobs <- candidate:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(items)
	}()

	result.Sessions = make([]session.Session, 0, len(candidates)+128)
	current := make(map[string]cachedFile, len(candidates))
	changed := make([]cachedFile, 0)
	for entry := range items {
		key := cacheKey(entry.provider, entry.path)
		current[key] = entry
		result.Sessions = append(result.Sessions, entry.session)
		previous, exists := cached[key]
		if !exists || !previous.valid || previous.size != entry.size || previous.modTimeNano != entry.modTimeNano {
			changed = append(changed, entry)
		}
	}
	if ctx.Err() == nil && cacheErr == nil {
		if err := updateCachedFiles(cacheDB, cached, current, changed); err != nil {
			result.Warnings = append(result.Warnings, "Cache: "+err.Error())
		}
	}

	if ctx.Err() == nil {
		s.scanDatabases(&result)
	}

	result.Sessions = dedupe(result.Sessions)
	session.SortNewest(result.Sessions)
	sort.Strings(result.Warnings)
	return result
}

func (s *Scanner) scanDatabases(result *Result) {
	if rows, err := scanOpenCodeDB(filepath.Join(s.Home, ".local/share/opencode/opencode.db")); err == nil {
		result.Sessions = append(result.Sessions, rows...)
	} else if !errors.Is(err, os.ErrNotExist) {
		result.Warnings = append(result.Warnings, "OpenCode: "+err.Error())
	}
	if rows, err := scanHermesDB(filepath.Join(s.Home, ".hermes/state.db")); err == nil {
		result.Sessions = append(result.Sessions, rows...)
	} else if !errors.Is(err, os.ErrNotExist) {
		result.Warnings = append(result.Warnings, "Hermes: "+err.Error())
	}
}

func (s *Scanner) discoverFiles() []fileCandidate {
	var candidates []fileCandidate
	seen := make(map[string]struct{})
	addTree := func(provider session.Provider, root string, accept func(string, fs.DirEntry) bool) {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrPermission) {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if accept(path, entry) {
				if _, exists := seen[path]; !exists {
					seen[path] = struct{}{}
					candidates = append(candidates, fileCandidate{provider: provider, path: path})
				}
			}
			return nil
		})
	}
	jsonLines := func(path string, _ fs.DirEntry) bool {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".jsonl" || ext == ".ndjson"
	}

	codexRoot := filepath.Join(s.Home, ".codex")
	for _, root := range []string{filepath.Join(codexRoot, "sessions"), filepath.Join(codexRoot, "archived_sessions")} {
		addTree(session.Codex, root, func(path string, _ fs.DirEntry) bool {
			return strings.HasPrefix(filepath.Base(path), "rollout-") && strings.EqualFold(filepath.Ext(path), ".jsonl")
		})
	}

	claudeRoots, _ := filepath.Glob(filepath.Join(s.Home, ".claude*"))
	for _, root := range claudeRoots {
		addTree(session.Claude, root, func(path string, entry fs.DirEntry) bool {
			return filepath.Base(path) != "journal.jsonl" && jsonLines(path, entry)
		})
	}
	addTree(session.Claude, filepath.Join(s.Home, "Library/Application Support/Claude/local-agent-mode-sessions"), jsonLines)

	for _, root := range []string{
		filepath.Join(s.Home, ".gemini/antigravity/brain"),
		filepath.Join(s.Home, ".gemini/antigravity-cli/brain"),
	} {
		addTree(session.Antigravity, root, func(path string, entry fs.DirEntry) bool {
			return filepath.Base(path) == "transcript.jsonl" && jsonLines(path, entry)
		})
	}
	addTree(session.Copilot, filepath.Join(s.Home, ".copilot/session-state"), jsonLines)
	addTree(session.Droid, filepath.Join(s.Home, ".factory/sessions"), jsonLines)
	addTree(session.Droid, filepath.Join(s.Home, ".factory/projects"), jsonLines)

	openClawRoot := os.Getenv("OPENCLAW_STATE_DIR")
	if openClawRoot == "" {
		openClawRoot = filepath.Join(s.Home, ".openclaw")
	}
	for _, root := range []string{openClawRoot, filepath.Join(s.Home, ".clawdbot")} {
		addTree(session.OpenClaw, filepath.Join(root, "agents"), func(path string, _ fs.DirEntry) bool {
			name := filepath.Base(path)
			return (strings.HasSuffix(name, ".jsonl") || strings.Contains(name, ".jsonl.deleted.")) &&
				!strings.HasSuffix(name, ".trajectory.jsonl") && !strings.HasSuffix(name, ".jsonl.lock")
		})
	}
	addTree(session.Cursor, filepath.Join(s.Home, ".cursor/projects"), func(path string, entry fs.DirEntry) bool {
		return strings.Contains(filepath.ToSlash(path), "/agent-transcripts/") && jsonLines(path, entry)
	})
	addTree(session.Pi, filepath.Join(s.Home, ".pi/agent/sessions"), jsonLines)

	addTree(session.OpenCode, filepath.Join(s.Home, ".local/share/opencode/storage/session"), func(path string, _ fs.DirEntry) bool {
		return strings.HasPrefix(filepath.Base(path), "ses_") && strings.EqualFold(filepath.Ext(path), ".json")
	})
	addTree(session.Hermes, filepath.Join(s.Home, ".hermes/sessions"), func(path string, _ fs.DirEntry) bool {
		return strings.HasPrefix(filepath.Base(path), "session_") && strings.EqualFold(filepath.Ext(path), ".json")
	})
	return candidates
}

func dedupe(items []session.Session) []session.Session {
	byKey := make(map[string]session.Session, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%s\x00%s", item.Provider, item.ID)
		if existing, found := byKey[key]; !found || richer(item, existing) {
			byKey[key] = item
		}
	}
	out := make([]session.Session, 0, len(byKey))
	for _, item := range byKey {
		out = append(out, item)
	}
	return out
}

func richer(a, b session.Session) bool {
	aScore := len(a.SearchText) + len(a.Title)*20 + len(a.CWD)*10
	bScore := len(b.SearchText) + len(b.Title)*20 + len(b.CWD)*10
	if aScore == bScore {
		return a.UpdatedAt.After(b.UpdatedAt)
	}
	return aScore > bScore
}
