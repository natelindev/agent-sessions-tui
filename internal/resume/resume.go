package resume

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/natelindev/agent-sessions-tui/internal/session"
)

func For(provider session.Provider, id, path string) *session.Resume {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	switch provider {
	case session.Codex:
		return &session.Resume{Binary: "codex", Args: []string{"resume", id}}
	case session.Claude:
		return &session.Resume{Binary: "claude", Args: []string{"--resume", id}}
	case session.Antigravity:
		return &session.Resume{Binary: "agy", Args: []string{"--conversation", id}}
	case session.OpenCode:
		return &session.Resume{Binary: "opencode", Args: []string{"--session", id}}
	case session.Hermes:
		return &session.Resume{Binary: "hermes", Args: []string{"--resume", id}}
	case session.Copilot:
		return &session.Resume{Binary: "copilot", Args: []string{"--resume=" + id}}
	case session.Cursor:
		return &session.Resume{Binary: "cursor", Args: []string{"agent", "--resume", id}}
	case session.Pi:
		target := strings.TrimSpace(path)
		if target == "" {
			target = id
		}
		return &session.Resume{Binary: "pi", Args: []string{"--session", target}}
	default:
		return nil
	}
}

func Command(item session.Session) (*exec.Cmd, error) {
	if item.Resume == nil || item.Resume.Binary == "" {
		return nil, fmt.Errorf("%s sessions cannot be resumed from the CLI", item.Provider.Label())
	}
	binary, err := exec.LookPath(item.Resume.Binary)
	if err != nil {
		return nil, fmt.Errorf("%s is not installed or not on PATH", item.Resume.Binary)
	}
	cmd := exec.Command(binary, item.Resume.Args...)
	if item.CWD != "" {
		if info, statErr := os.Stat(item.CWD); statErr == nil && info.IsDir() {
			cmd.Dir = item.CWD
		}
	}
	return cmd, nil
}

func Display(item session.Session) string {
	if item.Resume == nil {
		return "Resume unavailable"
	}
	parts := append([]string{item.Resume.Binary}, item.Resume.Args...)
	return strings.Join(parts, " ")
}
