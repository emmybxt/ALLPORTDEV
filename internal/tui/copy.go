package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) handleCopyKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "c", "esc":
		m.copying = false
		m.refresh()
		return m, m.mouseCommand()
	case "q":
		return m, tea.Quit
	case "up", "down", "k", "j", "pgup", "pgdown", "end", "G":
		m.handleLogKey(key.String())
	}
	return m, nil
}

func (m Model) copyView() string {
	label := "all services"
	if m.selected >= 0 {
		label = m.services[m.selected].Name
	}
	lines := []string{
		"COPY MODE · " + label + " · display paused",
		"Drag to highlight text, then use your terminal's Copy command.",
		"Cmd+C on macOS · Ctrl+Shift+C on many Linux terminals",
		"",
	}
	lines = append(lines, m.logLines(m.width, m.logHeight())...)
	lines = append(lines, "", "↑/↓ or PgUp/PgDn to browse · Esc/c dashboard · q quit")
	for i, line := range lines {
		lines[i] = ansi.Truncate(ansi.Strip(line), m.width, "…")
	}
	return strings.Join(lines, "\n")
}
