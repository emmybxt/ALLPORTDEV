package tui

import (
	"fmt"
	"strings"

	"allportdev/internal/runner"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	muted         = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	accent        = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("61"))
	statusStyles  = map[runner.Status]lipgloss.Style{
		runner.Running:  lipgloss.NewStyle().Foreground(lipgloss.Color("78")),
		runner.Failed:   lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		runner.Stopping: lipgloss.NewStyle().Foreground(lipgloss.Color("221")),
	}
)

func (m Model) View() string {
	if m.width < 60 || m.height < 14 {
		return "AllPortDev — enlarge terminal to 60×14 or more. q / Ctrl+C quits.\n"
	}
	if m.copying {
		return m.copyView()
	}
	leftWidth := m.sidebarWidth()
	rightWidth := m.width - leftWidth - 3
	height := m.logHeight()
	left := m.serviceLines(leftWidth, height)
	right := m.logLines(rightWidth, height)
	lines := []string{accent.Render("  ALLPORTDEV") + muted.Render("  /  your local services, together"), "", accent.Render("  SERVICES") + strings.Repeat(" ", leftWidth-10) + accent.Render("LOGS · "+m.filterLabel()), ""}
	for i := 0; i < height; i++ {
		lines = append(lines, fit(left[i], leftWidth)+muted.Render(" │ ")+fit(right[i], rightWidth))
	}
	lines = append(lines, "", fit(m.footer(), m.width), fit("  c select/copy · Tab service · s start · x stop · r restart", m.width), fit("  ↑/↓/wheel scroll · PgUp/PgDn page · End live · / search · q quit", m.width))
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "…")
	}
	return strings.Join(lines, "\n")
}

func fit(text string, width int) string {
	text = ansi.Truncate(text, width, "…")
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text)))
}

func (m Model) filterLabel() string {
	label := "all services"
	if m.selected >= 0 {
		label = m.services[m.selected].Name
	}
	if m.cutoff != 0 {
		return label + " · SCROLLED"
	}
	return label + " · LIVE"
}

func (m Model) serviceLines(width, height int) []string {
	rows := []string{fmt.Sprintf("  All services (%d)", len(m.services))}
	for _, s := range m.services {
		style, ok := statusStyles[s.Status]
		if !ok {
			style = muted
		}
		rows = append(rows, "  "+s.Name+" "+style.Render(string(s.Status)))
	}
	selected := m.selected + 1
	rows[selected] = selectedStyle.Render(fit("› "+ansi.Strip(rows[selected])[2:], width))
	start := m.serviceListStart()
	lines := make([]string, height)
	copy(lines, rows[start:min(len(rows), start+height)])
	return lines
}

func (m Model) sidebarWidth() int { return min(30, m.width/3) }

func (m Model) serviceListStart() int { return max(0, m.selected+2-m.logHeight()) }

func (m Model) logLines(width, height int) []string {
	logs := m.filteredLogs()
	end := max(0, len(logs)-m.offset)
	start := max(0, end-height)
	lines := make([]string, height)
	for i, entry := range logs[start:end] {
		prefix := muted.Render(entry.Time.Format("15:04:05")) + " " + accent.Render(entry.Service) + " "
		stream := muted.Render(entry.Stream)
		if entry.Stream == "err" {
			stream = statusStyles[runner.Failed].Render(entry.Stream)
		}
		lines[i] = ansi.Truncate(prefix+stream+"  "+entry.Text, width, "…")
	}
	if len(logs) == 0 {
		lines[0] = muted.Render("Waiting for logs…  s starts the selection.")
	}
	return lines
}

func (m Model) footer() string {
	if m.searching {
		return accent.Render("  Search: "+m.query+"▏") + muted.Render("  Enter apply · Esc clear")
	}
	if m.notice != "" {
		return "  " + m.notice
	}
	if m.query != "" {
		return "  Filter: " + m.query
	}
	return muted.Render("  Commands apply to the selected service, or all services when All is selected.")
}
