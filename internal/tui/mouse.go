package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if !m.mouseEnabled || m.copying || msg.Action != tea.MouseActionPress {
		return
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollLogs(3)
	case tea.MouseButtonWheelDown:
		m.scrollLogs(-3)
	case tea.MouseButtonLeft:
		m.selectClickedService(msg.X, msg.Y)
	}
}

func (m Model) mouseCommand() tea.Cmd {
	if m.mouseEnabled {
		return tea.EnableMouseCellMotion
	}
	return tea.DisableMouse
}

func (m *Model) selectClickedService(x, y int) {
	if m.width < 60 || m.height < 14 {
		return
	}
	if x < 0 || x >= m.sidebarWidth() || y < 4 || y >= 4+m.logHeight() {
		return
	}
	index := m.serviceListStart() + y - 4 - 1
	if index >= len(m.services) {
		return
	}
	m.selected, m.offset, m.cutoff = index, 0, 0
	m.refresh()
}
