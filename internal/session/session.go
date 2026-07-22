package session

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Provider string

const (
	Codex       Provider = "codex"
	Claude      Provider = "claude"
	Antigravity Provider = "antigravity"
	OpenCode    Provider = "opencode"
	Hermes      Provider = "hermes"
	Copilot     Provider = "copilot"
	Droid       Provider = "droid"
	OpenClaw    Provider = "openclaw"
	Cursor      Provider = "cursor"
	Pi          Provider = "pi"
)

func (p Provider) Label() string {
	switch p {
	case OpenCode:
		return "OpenCode"
	case OpenClaw:
		return "OpenClaw"
	case Antigravity:
		return "Antigravity"
	case Copilot:
		return "Copilot"
	default:
		name := string(p)
		if name == "" {
			return "Unknown"
		}
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

type Resume struct {
	Binary string
	Args   []string
}

type Session struct {
	Provider   Provider
	ID         string
	Title      string
	Project    string
	CWD        string
	Model      string
	Path       string
	StartedAt  time.Time
	UpdatedAt  time.Time
	SearchText string
	Resume     *Resume
}

func (s Session) CanResume() bool { return s.Resume != nil && s.Resume.Binary != "" }

func (s Session) DisplayTitle() string {
	if strings.TrimSpace(s.Title) != "" {
		return strings.TrimSpace(s.Title)
	}
	if s.Project != "" {
		return "Session in " + s.Project
	}
	return "Untitled session"
}

func (s Session) DisplayProject() string {
	if s.Project != "" {
		return s.Project
	}
	if s.CWD != "" {
		return filepath.Base(s.CWD)
	}
	return "-"
}

func SortNewest(items []Session) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
}

func Filter(items []Session, query string) []Session {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return items
	}
	out := make([]Session, 0, len(items))
	for _, item := range items {
		haystack := item.SearchText
		if haystack == "" {
			haystack = strings.ToLower(strings.Join([]string{
				string(item.Provider), item.Provider.Label(), item.ID, item.Title,
				item.Project, item.CWD, item.Model, item.Path,
			}, " "))
		}
		matched := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, item)
		}
	}
	return out
}
