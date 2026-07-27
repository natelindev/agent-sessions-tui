package discovery

import (
	"bufio"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/natelindev/agent-sessions-tui/internal/resume"
	"github.com/natelindev/agent-sessions-tui/internal/session"
)

const maxIndexedTermBytes = 4096

var (
	whitespace     = regexp.MustCompile(`\s+`)
	localFileURL   = regexp.MustCompile(`file://[^\s\)\]>"'\\]+`)
	antigravityLog = regexp.MustCompile(`[/\\]\.system_generated[/\\]logs[/\\]`)
)

type textIndex struct {
	builder strings.Builder
	seen    map[string]struct{}
}

func (t *textIndex) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if t.seen == nil {
		t.seen = make(map[string]struct{})
	}
	for _, term := range strings.Fields(strings.ToLower(value)) {
		if len(term) > maxIndexedTermBytes {
			continue
		}
		if _, exists := t.seen[term]; exists {
			continue
		}
		t.seen[term] = struct{}{}
		t.builder.WriteByte(' ')
		t.builder.WriteString(term)
	}
}

func (t *textIndex) String() string { return t.builder.String() }

func parseJSONSession(candidate fileCandidate) (session.Session, bool) {
	info, err := os.Stat(candidate.path)
	if err != nil || !info.Mode().IsRegular() {
		return session.Session{}, false
	}

	item := session.Session{
		Provider:  candidate.provider,
		ID:        fallbackID(candidate),
		Path:      candidate.path,
		UpdatedAt: info.ModTime(),
	}
	index := &textIndex{}
	index.add(item.ID)
	index.add(string(item.Provider))
	index.add(candidate.path)

	file, err := os.Open(candidate.path)
	if err != nil {
		return session.Session{}, false
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, readErr := reader.ReadBytes('\n')
		var value any
		if len(line) > 0 {
			if err := json.Unmarshal(line, &value); err == nil {
				consumeJSON(value, &item, index)
			}
		}
		if readErr != nil {
			break
		}
	}
	if item.Provider == session.Antigravity && item.CWD == "" {
		if artifactDirectory := antigravityArtifactProjectDirectory(candidate.path); shouldPreferProjectDirectory(artifactDirectory, item.CWD) {
			item.CWD = artifactDirectory
		}
	}

	if item.Project == "" {
		item.Project = projectFromPath(candidate, item.CWD)
	}
	if item.StartedAt.IsZero() {
		item.StartedAt = item.UpdatedAt
	}
	item.Title = cleanTitle(item.Title)
	index.add(item.Title)
	index.add(item.Project)
	index.add(item.CWD)
	index.add(item.Model)
	item.SearchText = index.String()
	item.Resume = resume.For(item.Provider, item.ID, item.Path)
	return item, item.ID != ""
}

func antigravityArtifactProjectDirectory(transcriptPath string) string {
	if !antigravityLog.MatchString(transcriptPath) {
		return ""
	}
	logsDirectory := filepath.Dir(transcriptPath)
	conversationDirectory := filepath.Dir(filepath.Dir(logsDirectory))
	files := []string{
		transcriptPath,
		filepath.Join(logsDirectory, "transcript_full.jsonl"),
	}
	if entries, err := os.ReadDir(conversationDirectory); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
				files = append(files, filepath.Join(conversationDirectory, entry.Name()))
			}
		}
	}

	best := ""
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(file, 2*1024*1024))
		_ = file.Close()
		if readErr != nil {
			continue
		}
		for _, match := range localFileURL.FindAllString(string(data), -1) {
			directory := normalizedProjectDirectory(match)
			if directory == "" || pathWithin(directory, conversationDirectory) {
				continue
			}
			if best == "" || pathWithin(directory, best) {
				best = directory
			}
		}
	}
	return best
}

func shouldPreferProjectDirectory(candidate, current string) bool {
	if candidate == "" {
		return false
	}
	return current == "" || pathWithin(candidate, current)
}

func pathWithin(path, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func consumeJSON(value any, item *session.Session, index *textIndex) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}

	if item.ID == "" {
		item.ID = firstStringDeep(object, "sessionId", "session_id", "conversationId", "conversation_id", "id")
	}
	if item.CWD == "" {
		raw := firstStringDeep(object, "cwd", "directory", "workspacePath", "workspace_path", "projectPath", "project_path")
		item.CWD = normalizedProjectDirectory(raw)
	}
	if item.CWD == "" && item.Provider == session.Antigravity {
		item.CWD = antigravityProjectDirectory(object)
	}
	if item.Model == "" {
		item.Model = firstStringDeep(object, "model", "modelId", "model_id")
	}
	if item.StartedAt.IsZero() {
		if raw := firstStringDeep(object, "timestamp", "createdAt", "created_at", "startTime", "started_at"); raw != "" {
			item.StartedAt = parseTime(raw)
		}
	}

	for _, message := range messageObjects(object) {
		role := strings.ToLower(stringValue(message["role"]))
		content := textValue(message["content"])
		if content == "" {
			content = textValue(message["text"])
		}
		if content == "" {
			continue
		}
		index.add(content)
		if item.Title == "" && (role == "user" || role == "human") {
			if title := titleCandidate(content); title != "" {
				item.Title = title
			}
		}
	}

	for _, key := range []string{"title", "summary", "prompt", "text"} {
		if text := stringValue(object[key]); text != "" {
			index.add(text)
			if item.Title == "" && key == "title" {
				item.Title = text
			}
		}
	}

	if item.Provider == session.Antigravity && item.Title == "" && strings.EqualFold(stringValue(object["type"]), "USER_INPUT") {
		if content := stringValue(object["content"]); content != "" {
			index.add(content)
			item.Title = titleCandidate(content)
		}
	}
}

func antigravityProjectDirectory(object map[string]any) string {
	for _, keys := range [][]string{
		{"DirectoryPath", "directoryPath", "workspaceRoot", "workspace_root", "repoRoot", "repo_root"},
		{"AbsolutePath", "absolutePath", "FilePath", "filePath"},
	} {
		if raw := firstStringDeep(object, keys...); raw != "" {
			if directory := normalizedProjectDirectory(raw); directory != "" {
				return directory
			}
		}
	}
	return ""
}

func normalizedProjectDirectory(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "\"'"))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "file://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		raw = parsed.Path
	}
	decoded, err := url.PathUnescape(raw)
	if err == nil {
		raw = decoded
	}
	if !filepath.IsAbs(raw) {
		return ""
	}

	start := filepath.Clean(raw)
	if info, statErr := os.Stat(start); statErr == nil {
		if !info.IsDir() {
			start = filepath.Dir(start)
		}
	} else if filepath.Ext(start) != "" {
		start = filepath.Dir(start)
	}

	directory := start
	for range 16 {
		if _, statErr := os.Stat(filepath.Join(directory, ".git")); statErr == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return start
}

func messageObjects(object map[string]any) []map[string]any {
	out := make([]map[string]any, 0, 3)
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if role := stringValue(typed["role"]); role != "" {
				out = append(out, typed)
			}
			for _, key := range []string{"payload", "message", "response", "data"} {
				if child, exists := typed[key]; exists {
					walk(child)
				}
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(object)
	return out
}

func textValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, child := range typed {
			if text := textValue(child); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		for _, key := range []string{"text", "content", "input_text", "output_text", "value"} {
			if text := textValue(typed[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func firstStringDeep(value any, keys ...string) string {
	wanted := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	var find func(any, int) string
	find = func(current any, depth int) string {
		if depth > 4 {
			return ""
		}
		switch typed := current.(type) {
		case map[string]any:
			for _, key := range keys {
				if raw, exists := typed[key]; exists {
					if text := stringValue(raw); text != "" {
						return text
					}
				}
			}
			for key, child := range typed {
				if _, skip := wanted[key]; skip {
					continue
				}
				if found := find(child, depth+1); found != "" {
					return found
				}
			}
		case []any:
			for _, child := range typed {
				if found := find(child, depth+1); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return find(value, 0)
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func titleCandidate(content string) string {
	content = strings.TrimSpace(content)
	blocked := []string{
		"# agents.md instructions", "<environment_context>", "<instructions>",
		"<permissions instructions>", "<collaboration_mode>", "<skills_instructions>",
		"suggest a conventional commit subject line summarizing",
	}
	lower := strings.ToLower(content)
	for _, prefix := range blocked {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#-* "))
		if len(line) < 3 || strings.HasPrefix(line, "<") {
			continue
		}
		return truncateRunes(whitespace.ReplaceAllString(line, " "), 140)
	}
	return ""
}

func cleanTitle(value string) string {
	return truncateRunes(whitespace.ReplaceAllString(strings.TrimSpace(value), " "), 140)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

func parseTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func fallbackID(candidate fileCandidate) string {
	base := filepath.Base(candidate.path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.TrimSuffix(base, ".trajectory")
	if candidate.provider == session.Codex {
		parts := strings.Split(base, "-")
		if len(parts) >= 8 {
			return strings.Join(parts[len(parts)-5:], "-")
		}
	}
	if candidate.provider == session.Antigravity {
		parts := strings.Split(filepath.ToSlash(candidate.path), "/brain/")
		if len(parts) == 2 {
			return strings.Split(parts[1], "/")[0]
		}
	}
	if candidate.provider == session.Copilot {
		parts := strings.Split(filepath.ToSlash(candidate.path), "/session-state/")
		if len(parts) == 2 {
			return strings.Split(parts[1], "/")[0]
		}
	}
	if candidate.provider == session.Cursor && filepath.Base(filepath.Dir(candidate.path)) != "subagents" {
		return filepath.Base(filepath.Dir(candidate.path))
	}
	return base
}

func projectFromPath(candidate fileCandidate, cwd string) string {
	if cwd != "" {
		return filepath.Base(filepath.Clean(cwd))
	}
	path := filepath.ToSlash(candidate.path)
	if candidate.provider == session.Cursor {
		if parts := strings.Split(path, "/.cursor/projects/"); len(parts) == 2 {
			return strings.Split(parts[1], "/")[0]
		}
	}
	if candidate.provider == session.Claude {
		if parts := strings.Split(path, "/projects/"); len(parts) > 1 {
			return strings.Split(parts[len(parts)-1], "/")[0]
		}
	}
	return ""
}
