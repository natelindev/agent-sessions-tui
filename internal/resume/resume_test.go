package resume

import (
	"reflect"
	"testing"

	"github.com/natelindev/agent-sessions-tui/internal/session"
)

func TestForBuildsProviderCommands(t *testing.T) {
	tests := []struct {
		provider session.Provider
		binary   string
		args     []string
	}{
		{session.Codex, "codex", []string{"resume", "abc"}},
		{session.Claude, "claude", []string{"--resume", "abc"}},
		{session.Antigravity, "agy", []string{"--conversation", "abc"}},
		{session.OpenCode, "opencode", []string{"--session", "abc"}},
		{session.Cursor, "cursor", []string{"agent", "--resume", "abc"}},
		{session.Copilot, "copilot", []string{"--resume=abc"}},
		{session.Pi, "pi", []string{"--session", "abc"}},
	}
	for _, test := range tests {
		t.Run(string(test.provider), func(t *testing.T) {
			got := For(test.provider, "abc", "")
			if got == nil || got.Binary != test.binary || !reflect.DeepEqual(got.Args, test.args) {
				t.Fatalf("For() = %#v, want %s %#v", got, test.binary, test.args)
			}
		})
	}
}

func TestPiPrefersExactSessionPath(t *testing.T) {
	got := For(session.Pi, "abc", "/sessions/project/session.jsonl")
	want := []string{"--session", "/sessions/project/session.jsonl"}
	if got == nil || !reflect.DeepEqual(got.Args, want) {
		t.Fatalf("Pi resume = %#v, want args %#v", got, want)
	}
}

func TestForRejectsUnsupportedAndEmptySessions(t *testing.T) {
	if got := For(session.Droid, "abc", ""); got != nil {
		t.Fatalf("Droid resume = %#v, want nil", got)
	}
	if got := For(session.Codex, " ", ""); got != nil {
		t.Fatalf("empty Codex resume = %#v, want nil", got)
	}
}
