package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/natelindev/agent-sessions-tui/internal/session"
)

func TestParseJSONSessionFindsMetadataTitleAndSearchContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout-2026-07-22T10-00-00-019f0000-1111-7222-8333-444444444444.jsonl")
	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"019f0000-1111-7222-8333-444444444444","cwd":"/work/acme","model":"gpt-test"}}`,
		`{"type":"response_item","payload":{"role":"user","content":[{"text":"# AGENTS.md instructions for /work/acme"}]}}`,
		`{"type":"response_item","payload":{"role":"user","content":[{"text":"Fix the payment retry queue\nand add coverage"}]}}`,
		`{"type":"response_item","payload":{"role":"assistant","content":[{"text":"Investigating idempotency keys"}]}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := parseJSONSession(fileCandidate{provider: session.Codex, path: path})
	if !ok {
		t.Fatal("parseJSONSession() rejected a valid session")
	}
	if got.ID != "019f0000-1111-7222-8333-444444444444" || got.Project != "acme" || got.Model != "gpt-test" {
		t.Fatalf("metadata = %#v", got)
	}
	if got.Title != "Fix the payment retry queue" {
		t.Fatalf("Title = %q", got.Title)
	}
	if !strings.Contains(got.SearchText, "idempotency keys") {
		t.Fatalf("SearchText does not include assistant content: %q", got.SearchText)
	}
	if got.Resume == nil || got.Resume.Binary != "codex" {
		t.Fatalf("Resume = %#v", got.Resume)
	}
}

func TestParseJSONSessionIndexesMessagesAfterLargeEarlierContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout-2026-07-27T10-00-00-019f0000-1111-7222-8333-555555555555.jsonl")
	content := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"019f0000-1111-7222-8333-555555555555","cwd":"/work/acme"}}`,
		`{"type":"response_item","payload":{"role":"assistant","content":[{"text":"` + strings.Repeat("repeated-filler ", 80000) + `"}]}}`,
		`{"type":"response_item","payload":{"role":"user","content":[{"text":"很好，现在把一赞文化的7月账单发给我看看"}]}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := parseJSONSession(fileCandidate{provider: session.Codex, path: path})
	if !ok {
		t.Fatal("parseJSONSession() rejected a valid session")
	}
	if !strings.Contains(got.SearchText, "一赞文化") {
		t.Fatalf("SearchText does not include content after a large earlier message: %q", got.SearchText)
	}
}

func TestFallbackIDForAntigravityConversation(t *testing.T) {
	path := "/home/test/.gemini/antigravity/brain/conversation-id/.system_generated/logs/transcript.jsonl"
	got := fallbackID(fileCandidate{provider: session.Antigravity, path: path})
	if got != "conversation-id" {
		t.Fatalf("fallbackID() = %q", got)
	}
}

func TestAntigravityInfersProjectFromToolPath(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "sample-project")
	transcriptDir := filepath.Join(root, "brain", "conversation", ".system_generated", "logs")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(transcriptDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(transcriptDir, "transcript.jsonl")
	content := strings.Join([]string{
		`{"type":"USER_INPUT","content":"<USER_REQUEST>\nImprove the session browser\n</USER_REQUEST>"}`,
		`{"type":"PLANNER_RESPONSE","tool_calls":[{"name":"list_dir","args":{"DirectoryPath":"` + filepath.Join(repo, "internal", "ui") + `"}}]}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := parseJSONSession(fileCandidate{provider: session.Antigravity, path: path})
	if !ok {
		t.Fatal("parseJSONSession() rejected Antigravity fixture")
	}
	if got.CWD != repo || got.Project != "sample-project" {
		t.Fatalf("project metadata = cwd %q, project %q", got.CWD, got.Project)
	}
	if got.Title != "Improve the session browser" {
		t.Fatalf("Title = %q", got.Title)
	}
}

func TestAntigravityFallsBackToFullTranscriptFileURLs(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "legacy-project")
	logs := filepath.Join(root, "brain", "legacy-conversation", ".system_generated", "logs")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o700); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(logs, "transcript.jsonl")
	if err := os.WriteFile(transcript, []byte(`{"type":"USER_INPUT","content":"Review the legacy project"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(logs, "transcript_full.jsonl")
	fullContent := `{"content":"Opened file://` + filepath.Join(repo, "internal", "service.go") + `"}`
	if err := os.WriteFile(full, []byte(fullContent), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := parseJSONSession(fileCandidate{provider: session.Antigravity, path: transcript})
	if !ok {
		t.Fatal("parseJSONSession() rejected legacy Antigravity fixture")
	}
	if got.CWD != repo || got.Project != "legacy-project" {
		t.Fatalf("fallback project metadata = cwd %q, project %q", got.CWD, got.Project)
	}
}
