package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/natelindev/agent-sessions-tui/internal/discovery"
	"github.com/natelindev/agent-sessions-tui/internal/session"
)

func testModel() Model {
	m := New(discovery.New("/nonexistent"))
	m.width = 90
	m.height = 24
	updated, _ := m.Update(scanMsg(discovery.Result{Sessions: []session.Session{
		{ID: "one", Provider: session.Codex, Title: "Payment retry", SearchText: "payment retry"},
		{ID: "two", Provider: session.Claude, Title: "Dashboard", SearchText: "dashboard charts"},
	}}))
	return updated.(Model)
}

func TestTypingFiltersImmediately(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("retry")})
	m = updated.(Model)
	if m.query != "retry" || len(m.filtered) != 1 || m.filtered[0].ID != "one" {
		t.Fatalf("filtered state = query %q, sessions %#v", m.query, m.filtered)
	}
}

func TestMouseSelectsVisibleRow(t *testing.T) {
	m := testModel()
	mouse := tea.MouseMsg{X: 2, Y: m.listTop() + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	updated, _ := m.Update(mouse)
	m = updated.(Model)
	if m.selected != 1 {
		t.Fatalf("selected = %d, want 1", m.selected)
	}
}

func TestViewFitsWindowHeight(t *testing.T) {
	m := testModel()
	view := m.View()
	if lines := strings.Count(view, "\n") + 1; lines != m.height {
		t.Fatalf("View() has %d lines, want %d", lines, m.height)
	}
}

func TestProvidersHaveDistinctColors(t *testing.T) {
	providers := []session.Provider{
		session.Codex, session.Claude, session.Antigravity, session.OpenCode,
		session.Hermes, session.Copilot, session.Droid, session.OpenClaw,
		session.Cursor, session.Pi,
	}
	seen := make(map[any]session.Provider, len(providers))
	for _, provider := range providers {
		color := providerColor(provider)
		if previous, exists := seen[color]; exists {
			t.Fatalf("%s and %s share provider color %#v", previous, provider, color)
		}
		seen[color] = provider
	}
}

func TestSearchCursorStartsWhereTypedTextStarts(t *testing.T) {
	m := testModel()
	m.focused = true
	m.cursorOn = true

	empty := ansi.Strip(m.searchContent())
	emptyCursor := strings.Index(empty, "|")
	if emptyCursor < 0 {
		t.Fatalf("empty search content has no cursor: %q", empty)
	}

	m.query = "r"
	typed := ansi.Strip(m.searchContent())
	typedStart := strings.LastIndex(typed, "r")
	if typedStart != emptyCursor {
		t.Fatalf("typed input starts at column %d, empty cursor was at %d: empty %q, typed %q", typedStart, emptyCursor, empty, typed)
	}
}
