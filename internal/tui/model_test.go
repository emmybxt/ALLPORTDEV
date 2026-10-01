package tui

import (
	"fmt"
	"strings"
	"testing"

	"allportdev/internal/config"
	"allportdev/internal/runner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestArrowScrollingKeepsSelectedService(t *testing.T) {
	m := logHistoryModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.selected != 0 {
		t.Fatalf("scroll changed selected service from api to index %d", m.selected)
	}
	if m.offset != 1 || m.cutoff != 50 {
		t.Fatalf("up did not scroll and pause history: offset=%d cutoff=%d", m.offset, m.cutoff)
	}
	before := strings.Join(m.logLines(80, m.logHeight()), "\n")
	m.logs = append(m.logs, runner.Log{Sequence: 51, Service: "api", Text: "new live output"})
	if after := strings.Join(m.logLines(80, m.logHeight()), "\n"); after != before {
		t.Fatal("new output moved the scrolled log view")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.selected != 0 || m.offset != 0 || m.cutoff != 0 {
		t.Fatalf("down should return to live logs without switching service: %+v", m)
	}
}

func logHistoryModel() Model {
	m := New(runner.New([]config.Service{{Name: "api"}, {Name: "web"}}))
	m.selected = 0
	for i := 1; i <= 50; i++ {
		m.logs = append(m.logs, runner.Log{Sequence: uint64(i), Service: "api", Text: fmt.Sprintf("line %d", i)})
	}
	return m
}

func TestWheelScrollAndClickSelection(t *testing.T) {
	m := logHistoryModel()
	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp, X: 60, Y: 10})
	m = updated.(Model)
	if m.selected != 0 || m.offset != 3 {
		t.Fatalf("wheel should scroll api logs: service=%d offset=%d", m.selected, m.offset)
	}
	updated, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: 60, Y: 10})
	m = updated.(Model)
	if m.selected != 0 || m.offset != 0 || m.cutoff != 0 {
		t.Fatal("wheel down did not resume live api logs")
	}
	updated, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 5, Y: 6})
	m = updated.(Model)
	if m.selected != 1 {
		t.Fatal("click on web did not select web")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if updated.(Model).selected != 0 {
		t.Fatal("shift+tab did not select previous service")
	}
}

func TestLogScrollingBoundsAndLiveFollowing(t *testing.T) {
	m := logHistoryModel()
	m.scrollLogs(1000)
	if m.offset != 50-m.logHeight() {
		t.Fatal("scroll exceeded oldest retained log")
	}
	m.scrollLogs(-1000)
	if m.offset != 0 || m.cutoff != 0 {
		t.Fatal("scrolling to bottom did not resume live logs")
	}
	m.logs = m.logs[:1]
	m.scrollLogs(1)
	if m.offset != 0 || m.cutoff != 0 {
		t.Fatal("short log history should remain live")
	}
}

func TestSelectionSearchAndViewBounds(t *testing.T) {
	m := New(runner.New([]config.Service{{Name: strings.Repeat("long-service-name", 5)}, {Name: "web"}}))
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.selected != 0 {
		t.Fatal("tab did not select api")
	}
	m.logs = []runner.Log{{Service: "api", Text: "connected"}, {Service: "api", Text: "ERROR database unavailable"}}
	m.query = "error"
	if logs := m.filteredLogs(); len(logs) != 1 || logs[0].Text != "ERROR database unavailable" {
		t.Fatal("search filter failed")
	}
	for _, size := range [][2]int{{60, 14}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		lines := strings.Split(m.View(), "\n")
		if len(lines) > m.height {
			t.Fatal("view exceeds terminal height")
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatalf("view exceeds width: %q", line)
			}
		}
	}
}
