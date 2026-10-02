package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.copying {
		return m.handleCopyKey(key)
	}
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.searching {
		return m.handleSearch(key), nil
	}
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "c":
		m.copying = true
		return m, tea.DisableMouse
	case "s", "x", "r":
		return m.action(key.String())
	case "shift+tab":
		m.selectService(-1)
	case "tab":
		m.selectService(1)
	case "a":
		m.selected, m.offset, m.cutoff = -1, 0, 0
		m.refresh()
	default:
		m.handleLogKey(key.String())
	}
	return m, nil
}

func (m *Model) selectService(delta int) {
	m.selected += delta
	if m.selected >= len(m.services) {
		m.selected = -1
	}
	if m.selected < -1 {
		m.selected = len(m.services) - 1
	}
	m.offset, m.cutoff = 0, 0
	m.refresh()
}

func (m *Model) handleLogKey(key string) {
	switch key {
	case "/":
		m.searching = true
	case "esc":
		m.query = ""
	case "pgup":
		m.scrollLogs(m.logHeight())
	case "pgdown":
		m.scrollLogs(-m.logHeight())
	case "up", "k":
		m.scrollLogs(1)
	case "down", "j":
		m.scrollLogs(-1)
	case "end", "G":
		m.offset, m.cutoff = 0, 0
	}
}

func (m *Model) scrollLogs(lines int) {
	if m.cutoff == 0 && len(m.logs) > 0 {
		m.cutoff = m.logs[len(m.logs)-1].Sequence
	}
	limit := max(0, len(m.filteredLogs())-m.logHeight())
	m.offset = min(max(0, m.offset+lines), limit)
	if m.offset == 0 {
		m.cutoff = 0
	}
}

func (m Model) handleSearch(key tea.KeyMsg) Model {
	switch key.String() {
	case "enter":
		m.searching = false
	case "esc":
		m.searching, m.query = false, ""
	case "backspace":
		runes := []rune(m.query)
		if len(runes) > 0 {
			m.query = string(runes[:len(runes)-1])
		}
	default:
		if key.Type == tea.KeyRunes {
			m.query += string(key.Runes)
		}
		if key.Type == tea.KeySpace {
			m.query += " "
		}
	}
	m.offset, m.cutoff = 0, 0
	return m
}
