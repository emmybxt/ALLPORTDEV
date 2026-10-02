package tui

import (
	"fmt"
	"strings"
	"time"

	"allportdev/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time
type actionMsg struct{ err error }

type Model struct {
	manager       *runner.Manager
	services      []runner.ServiceView
	logs          []runner.Log
	selected      int
	width, height int
	query         string
	searching     bool
	copying       bool
	offset        int
	cutoff        uint64
	busy          bool
	notice        string
}

func New(manager *runner.Manager) Model {
	m := Model{manager: manager, selected: -1, width: 100, height: 30}
	m.refresh()
	return m
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.refresh()
		return m, tick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case actionMsg:
		m.busy = false
		m.notice = ""
		if msg.err != nil {
			m.notice = msg.err.Error()
		}
		m.refresh()
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		m.handleMouse(msg)
	}
	return m, nil
}

func (m *Model) refresh() {
	if m.copying {
		return
	}
	m.services, m.logs = m.manager.Snapshot(m.selected)
}

func (m Model) action(key string) (tea.Model, tea.Cmd) {
	if m.busy {
		return m, nil
	}
	m.busy = true
	m.notice = "Applying " + key + "…"
	selected, count := m.selected, len(m.services)
	operation := map[string]func(int) error{"s": m.manager.Start, "x": m.manager.Stop, "r": m.manager.Restart}[key]
	return m, func() tea.Msg {
		if selected >= 0 {
			return actionMsg{operation(selected)}
		}
		// Stop/restart services concurrently so each gets its full grace period.
		results := make(chan error, count)
		for i := 0; i < count; i++ {
			go func(index int) { results <- operation(index) }(i)
		}
		var failures []string
		for i := 0; i < count; i++ {
			if err := <-results; err != nil {
				failures = append(failures, err.Error())
			}
		}
		if len(failures) > 0 {
			return actionMsg{fmt.Errorf("%s", strings.Join(failures, "; "))}
		}
		return actionMsg{}
	}
}

func (m Model) filteredLogs() []runner.Log {
	var logs []runner.Log
	query := strings.ToLower(m.query)
	for _, entry := range m.logs {
		if m.cutoff != 0 && entry.Sequence > m.cutoff {
			continue
		}
		if !strings.Contains(strings.ToLower(entry.Text), query) {
			continue
		}
		logs = append(logs, entry)
	}
	return logs
}

func (m Model) logHeight() int { return max(1, m.height-9) }
