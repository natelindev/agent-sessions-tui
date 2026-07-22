package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/natelindev/agent-sessions-tui/internal/session"
)

type styles struct {
	title       lipgloss.Style
	muted       lipgloss.Style
	accent      lipgloss.Style
	error       lipgloss.Style
	search      lipgloss.Style
	searchFocus lipgloss.Style
	column      lipgloss.Style
	row         lipgloss.Style
	age         lipgloss.Style
	project     lipgloss.Style
	selected    lipgloss.Style
	footer      lipgloss.Style
	selectedBG  lipgloss.TerminalColor
}

func newStyles() styles {
	accent := lipgloss.AdaptiveColor{Light: "#5B4FDB", Dark: "#A79CFF"}
	muted := lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B93A7"}
	border := lipgloss.AdaptiveColor{Light: "#D7DAE0", Dark: "#343A4A"}
	selectedBG := lipgloss.AdaptiveColor{Light: "#E7E4FF", Dark: "#302B4F"}
	text := lipgloss.AdaptiveColor{Light: "#171923", Dark: "#ECEEF4"}
	project := lipgloss.AdaptiveColor{Light: "#386A8C", Dark: "#7DB9DE"}

	return styles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(text),
		muted:       lipgloss.NewStyle().Foreground(muted),
		accent:      lipgloss.NewStyle().Bold(true).Foreground(accent),
		error:       lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF8A80"}),
		search:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1),
		searchFocus: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1),
		column:      lipgloss.NewStyle().Bold(true).Foreground(muted),
		row:         lipgloss.NewStyle().Foreground(text),
		age:         lipgloss.NewStyle().Foreground(muted),
		project:     lipgloss.NewStyle().Foreground(project),
		selected:    lipgloss.NewStyle().Foreground(text).Background(selectedBG).Bold(true),
		footer:      lipgloss.NewStyle().Foreground(muted).BorderTop(true).BorderForeground(border),
		selectedBG:  selectedBG,
	}
}

func providerColor(provider session.Provider) lipgloss.TerminalColor {
	switch provider {
	case session.Codex:
		return lipgloss.AdaptiveColor{Light: "#006C9C", Dark: "#55C2FF"}
	case session.Claude:
		return lipgloss.AdaptiveColor{Light: "#A64B16", Dark: "#FF9B62"}
	case session.Antigravity:
		return lipgloss.AdaptiveColor{Light: "#7147B8", Dark: "#C09CFF"}
	case session.OpenCode:
		return lipgloss.AdaptiveColor{Light: "#00786F", Dark: "#55D6C9"}
	case session.Hermes:
		return lipgloss.AdaptiveColor{Light: "#6546A5", Dark: "#B69CFF"}
	case session.Copilot:
		return lipgloss.AdaptiveColor{Light: "#A43B74", Dark: "#FF8CC8"}
	case session.Droid:
		return lipgloss.AdaptiveColor{Light: "#8A6400", Dark: "#F6CE5A"}
	case session.OpenClaw:
		return lipgloss.AdaptiveColor{Light: "#A33D3D", Dark: "#FF8585"}
	case session.Cursor:
		return lipgloss.AdaptiveColor{Light: "#2859A6", Dark: "#79A8FF"}
	case session.Pi:
		return lipgloss.AdaptiveColor{Light: "#347348", Dark: "#75D68F"}
	default:
		return lipgloss.AdaptiveColor{Light: "#5B4FDB", Dark: "#A79CFF"}
	}
}
