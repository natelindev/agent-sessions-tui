package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/natelindev/agent-sessions-tui/internal/discovery"
	"github.com/natelindev/agent-sessions-tui/internal/resume"
	"github.com/natelindev/agent-sessions-tui/internal/session"
)

type scanMsg discovery.Result
type cachedScanMsg struct {
	result discovery.Result
	ok     bool
}
type scanErrMsg struct{ err error }
type resumeDoneMsg struct{ err error }
type cursorTickMsg struct{}

type Model struct {
	scanner *discovery.Scanner
	styles  styles

	all      []session.Session
	filtered []session.Session
	warnings []string
	query    string
	focused  bool
	loading  bool
	err      string

	width    int
	height   int
	selected int
	offset   int
	cursorOn bool
	spin     int

	lastClickAt    time.Time
	lastClickIndex int
}

func New(scanner *discovery.Scanner) Model {
	return Model{
		scanner:        scanner,
		styles:         newStyles(),
		loading:        true,
		cursorOn:       true,
		lastClickIndex: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.cachedScanCmd(), cursorTick(), tea.SetWindowTitle("Agent Sessions"))
}

func (m Model) cachedScanCmd() tea.Cmd {
	return func() tea.Msg {
		result, ok := m.scanner.Cached()
		return cachedScanMsg{result: result, ok: ok}
	}
}

func (m Model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		result := m.scanner.Scan(context.Background())
		return scanMsg(result)
	}
}

func cursorTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return cursorTickMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.ensureVisible()
		return m, nil
	case cachedScanMsg:
		if typed.ok {
			selectedID := m.selectedID()
			m.all = typed.result.Sessions
			m.warnings = typed.result.Warnings
			m.applyFilter(selectedID)
		}
		return m, m.scanCmd()
	case scanMsg:
		selectedID := m.selectedID()
		result := discovery.Result(typed)
		m.all = result.Sessions
		m.warnings = result.Warnings
		m.loading = false
		m.err = ""
		m.applyFilter(selectedID)
		return m, nil
	case scanErrMsg:
		m.loading = false
		m.err = typed.err.Error()
		return m, nil
	case resumeDoneMsg:
		if typed.err != nil {
			m.err = typed.err.Error()
		}
		return m, nil
	case cursorTickMsg:
		m.cursorOn = !m.cursorOn
		if m.loading {
			m.spin++
		}
		return m, cursorTick()
	case tea.MouseMsg:
		return m.handleMouse(tea.MouseEvent(typed))
	case tea.KeyMsg:
		return m.handleKey(typed)
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	name := key.String()
	if m.focused {
		switch name {
		case "esc", "enter":
			m.focused = false
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "backspace":
			if m.query != "" {
				_, size := utf8.DecodeLastRuneInString(m.query)
				m.query = m.query[:len(m.query)-size]
				m.applyFilter("")
			}
			return m, nil
		case "ctrl+u":
			m.query = ""
			m.applyFilter("")
			return m, nil
		case "ctrl+w":
			m.query = strings.TrimRight(m.query, " ")
			if index := strings.LastIndexByte(m.query, ' '); index >= 0 {
				m.query = m.query[:index+1]
			} else {
				m.query = ""
			}
			m.applyFilter("")
			return m, nil
		}
		if (key.Type == tea.KeyRunes || key.Type == tea.KeySpace) && !key.Alt {
			m.query += string(key.Runes)
			m.applyFilter("")
		}
		return m, nil
	}

	switch name {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.focused = true
		m.cursorOn = true
	case "esc":
		if m.query != "" {
			m.query = ""
			m.applyFilter("")
		}
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup", "ctrl+u":
		m.move(-m.listHeight())
	case "pgdown", "ctrl+d":
		m.move(m.listHeight())
	case "home", "g":
		m.selected = 0
		m.ensureVisible()
	case "end", "G":
		m.selected = max(0, len(m.filtered)-1)
		m.ensureVisible()
	case "r":
		if !m.loading {
			m.loading = true
			m.err = ""
			return m, m.scanCmd()
		}
	case "enter":
		return m.resumeSelected()
	}
	return m, nil
}

func (m Model) handleMouse(mouse tea.MouseEvent) (tea.Model, tea.Cmd) {
	switch mouse.Button {
	case tea.MouseButtonWheelUp:
		m.focused = false
		m.move(-3)
	case tea.MouseButtonWheelDown:
		m.focused = false
		m.move(3)
	case tea.MouseButtonLeft:
		if mouse.Action != tea.MouseActionPress {
			return m, nil
		}
		if mouse.Y >= 1 && mouse.Y <= 3 {
			m.focused = true
			m.cursorOn = true
			return m, nil
		}
		row := mouse.Y - m.listTop()
		index := m.offset + row
		if row >= 0 && row < m.listHeight() && index >= 0 && index < len(m.filtered) {
			double := index == m.lastClickIndex && time.Since(m.lastClickAt) <= 450*time.Millisecond
			m.selected = index
			m.focused = false
			m.lastClickAt = time.Now()
			m.lastClickIndex = index
			m.ensureVisible()
			if double {
				return m.resumeSelected()
			}
		}
	}
	return m, nil
}

func (m Model) resumeSelected() (tea.Model, tea.Cmd) {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return m, nil
	}
	item := m.filtered[m.selected]
	cmd, err := resume.Command(item)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	m.err = ""
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return resumeDoneMsg{err: err} })
}

func (m *Model) applyFilter(selectedID string) {
	m.filtered = session.Filter(m.all, m.query)
	m.selected = 0
	if selectedID != "" {
		for i := range m.filtered {
			if m.filtered[i].ID == selectedID {
				m.selected = i
				break
			}
		}
	}
	m.offset = 0
	m.ensureVisible()
}

func (m *Model) move(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected = min(max(m.selected+delta, 0), len(m.filtered)-1)
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	visible := m.listHeight()
	if visible <= 0 {
		m.offset = 0
		return
	}
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visible {
		m.offset = m.selected - visible + 1
	}
	maxOffset := max(0, len(m.filtered)-visible)
	m.offset = min(max(m.offset, 0), maxOffset)
}

func (m Model) listHeight() int { return max(1, m.height-8) }
func (m Model) listTop() int    { return 6 }

func (m Model) selectedID() string {
	if m.selected >= 0 && m.selected < len(m.filtered) {
		return m.filtered[m.selected].ID
	}
	return ""
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading agent sessions..."
	}
	width := max(24, m.width)
	lines := make([]string, 0, m.height)

	spinner := ""
	if m.loading {
		frames := []string{"|", "/", "-", "\\"}
		spinner = " " + frames[m.spin%len(frames)] + " Scanning"
	}
	title := m.styles.title.Render("Agent Sessions")
	if spinner != "" {
		spinner += "  "
	}
	count := fmt.Sprintf("%s%d%s", spinner, len(m.filtered), plural(len(m.filtered), " session", " sessions"))
	lines = append(lines, joinSides(title, m.styles.muted.Render(count), width))

	searchText := m.searchContent()
	searchStyle := m.styles.search
	if m.focused {
		searchStyle = m.styles.searchFocus
	}
	lines = append(lines, searchStyle.Width(max(1, width-2)).Render(truncate(searchText, max(1, width-6))))

	status := fmt.Sprintf("Showing %d of %d", len(m.filtered), len(m.all))
	if m.query != "" {
		status += "  -  all search terms must match"
	}
	statusStyle := m.styles.muted
	if m.query != "" && len(m.filtered) > 0 {
		statusStyle = m.styles.accent
	}
	lines = append(lines, pad(statusStyle.Render(status), width))
	lines = append(lines, m.renderColumns(width))

	height := m.listHeight()
	for row := 0; row < height; row++ {
		index := m.offset + row
		if index >= len(m.filtered) {
			if len(m.filtered) == 0 && row == 1 {
				message := "No sessions found"
				if m.loading {
					message = "Scanning local session stores..."
				} else if m.query != "" {
					message = "No sessions match your search"
				}
				lines = append(lines, pad(m.styles.muted.Render(message), width))
			} else {
				lines = append(lines, strings.Repeat(" ", width))
			}
			continue
		}
		lines = append(lines, m.renderRow(m.filtered[index], index == m.selected, width))
	}

	detail := m.detail(width)
	lines = append(lines, pad(detail, width))
	help := "↑/↓ or j/k navigate   / search   Enter resume   r refresh   q quit"
	lines = append(lines, m.styles.footer.Width(width).Render(truncate(help, width)))
	return strings.Join(lines, "\n")
}

func (m Model) searchContent() string {
	prefix := m.styles.accent.Render("Search  ")
	placeholder := "Type to filter by title, project, ID, model, or transcript"
	if m.query == "" {
		if !m.focused {
			return prefix + m.styles.muted.Render(placeholder)
		}
		cursor := " "
		if m.cursorOn {
			cursor = m.styles.accent.Render("|")
		}
		return prefix + cursor + m.styles.muted.Render(" "+placeholder)
	}
	content := prefix + m.query
	if m.focused && m.cursorOn {
		content += m.styles.accent.Render("|")
	}
	return content
}

func (m Model) renderColumns(width int) string {
	providerWidth, ageWidth, projectWidth, titleWidth := columns(width)
	parts := []string{
		fit("AGENT", providerWidth),
		fit("UPDATED", ageWidth),
	}
	if projectWidth > 0 {
		parts = append(parts, fit("PROJECT", projectWidth))
	}
	parts = append(parts, fit("SESSION", titleWidth))
	return m.styles.column.Render(pad(strings.Join(parts, " "), width))
}

func (m Model) renderRow(item session.Session, selected bool, width int) string {
	providerWidth, ageWidth, projectWidth, titleWidth := columns(width)
	provider := item.Provider.Label()
	if !item.CanResume() {
		provider += "*"
	}
	providerStyle := lipgloss.NewStyle().Bold(true).Foreground(providerColor(item.Provider))
	ageStyle := m.styles.age
	projectStyle := m.styles.project
	titleStyle := m.styles.row
	separatorStyle := lipgloss.NewStyle()
	if selected {
		providerStyle = providerStyle.Background(m.styles.selectedBG)
		ageStyle = ageStyle.Background(m.styles.selectedBG)
		projectStyle = projectStyle.Background(m.styles.selectedBG)
		titleStyle = titleStyle.Background(m.styles.selectedBG).Bold(true)
		separatorStyle = separatorStyle.Background(m.styles.selectedBG)
	}
	parts := []string{
		providerStyle.Render(fit(provider, providerWidth)),
		ageStyle.Render(fit(relativeTime(item.UpdatedAt), ageWidth)),
	}
	if projectWidth > 0 {
		parts = append(parts, projectStyle.Render(fit(item.DisplayProject(), projectWidth)))
	}
	parts = append(parts, titleStyle.Render(fit(item.DisplayTitle(), titleWidth)))
	line := strings.Join(parts, separatorStyle.Render(" "))
	if selected {
		return m.styles.selected.Width(width).Render(line)
	}
	return pad(line, width)
}

func (m Model) detail(width int) string {
	if m.err != "" {
		return m.styles.error.Render(truncate("Error: "+m.err, width))
	}
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		if len(m.warnings) > 0 {
			return m.styles.error.Render(truncate(m.warnings[0], width))
		}
		return ""
	}
	item := m.filtered[m.selected]
	id := item.ID
	if len(id) > 24 {
		id = id[:24] + "…"
	}
	provider := lipgloss.NewStyle().Bold(true).Foreground(providerColor(item.Provider)).Render(item.Provider.Label())
	parts := []string{provider, m.styles.muted.Render(id)}
	if item.Model != "" {
		parts = append(parts, m.styles.project.Render(item.Model))
	}
	if item.CanResume() {
		parts = append(parts, m.styles.accent.Render(resume.Display(item)))
	} else {
		parts = append(parts, m.styles.muted.Render("resume unavailable"))
	}
	return truncate(strings.Join(parts, "  -  "), width)
}

func columns(width int) (provider, age, project, title int) {
	provider, age = 12, 11
	if width < 66 {
		provider, age = 10, 8
		project = 0
		title = max(8, width-provider-age-3)
		return
	}
	project = min(24, max(14, width/5))
	title = max(12, width-provider-age-project-4)
	return
}

func relativeTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	delta := time.Since(value)
	if delta < 0 {
		delta = 0
	}
	switch {
	case delta < time.Minute:
		return "now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	case delta < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	default:
		return value.Local().Format("Jan 02")
	}
}

func fit(value string, width int) string {
	return pad(truncate(value, width), width)
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

func pad(value string, width int) string {
	missing := width - lipgloss.Width(value)
	if missing <= 0 {
		return value
	}
	return value + strings.Repeat(" ", missing)
}

func joinSides(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncate(left, max(1, width-lipgloss.Width(right)-1)) + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
