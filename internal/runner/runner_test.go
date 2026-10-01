//go:build darwin || linux

package runner

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"allportdev/internal/config"
)

func waitFor(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for process state")
}

func TestCapturesOutputEnvironmentAndExit(t *testing.T) {
	dir := t.TempDir()
	m := New([]config.Service{{Name: "api", Dir: dir, Command: `printf '%s\n' "$ALLPORTDEV_TEST"; pwd; printf 'warning\n' >&2; printf 'tail'; exit 7`, Env: map[string]string{"ALLPORTDEV_TEST": "configured"}}})
	t.Cleanup(m.Shutdown)
	if err := m.Start(0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { views, _ := m.Snapshot(0); return views[0].Status == Failed })
	_, logs := m.Snapshot(0)
	found := map[string]bool{}
	for _, log := range logs {
		found[log.Stream+":"+log.Text] = true
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"out:configured", "out:" + resolved, "err:warning", "out:tail"} {
		if !found[want] {
			t.Errorf("missing %q in %+v", want, logs)
		}
	}
}

func TestStopKillsStubbornProcessGroupAndRestart(t *testing.T) {
	dir := t.TempDir()
	command := `trap '' TERM; sh -c 'trap "" TERM; echo $$ > child.pid; while :; do sleep 1; done' & wait`
	m := New([]config.Service{{Name: "worker", Dir: dir, Command: command}})
	m.stopTimeout = 80 * time.Millisecond
	t.Cleanup(m.Shutdown)
	if err := m.Start(0); err != nil {
		t.Fatal(err)
	}
	var childPID int
	waitFor(t, func() bool {
		data, err := os.ReadFile(filepath.Join(dir, "child.pid"))
		if err != nil {
			return false
		}
		childPID, err = strconv.Atoi(strings.TrimSpace(string(data)))
		return err == nil
	})
	views, _ := m.Snapshot(0)
	oldPID := views[0].PID
	if err := m.Stop(0); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return syscall.Kill(childPID, 0) == syscall.ESRCH })
	views, _ = m.Snapshot(0)
	if views[0].Status != Stopped {
		t.Fatalf("got %s", views[0].Status)
	}
	if err := m.Restart(0); err != nil {
		t.Fatal(err)
	}
	views, _ = m.Snapshot(0)
	if views[0].PID == oldPID || views[0].Status != Running {
		t.Fatalf("restart: %+v", views[0])
	}
}

func TestConcurrentStartIsIdempotentAndShutdownPreventsRestart(t *testing.T) {
	m := New([]config.Service{{Name: "api", Command: "sleep 60"}})
	t.Cleanup(m.Shutdown)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Start(0); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	_, logs := m.Snapshot(0)
	if len(logs) != 1 {
		t.Fatalf("expected exactly one start, got %+v", logs)
	}
	m.Shutdown()
	if err := m.Start(0); err == nil {
		t.Fatal("start allowed after shutdown")
	}
}

func TestMergedLogsAndServiceFilter(t *testing.T) {
	m := New([]config.Service{{Name: "api", Command: "echo api"}, {Name: "web", Command: "echo web"}})
	t.Cleanup(m.Shutdown)
	for i := 0; i < 2; i++ {
		if err := m.Start(i); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, func() bool { v, _ := m.Snapshot(-1); return v[0].Status == Exited && v[1].Status == Exited })
	_, all := m.Snapshot(-1)
	for i := 1; i < len(all); i++ {
		if all[i].Sequence <= all[i-1].Sequence {
			t.Fatal("merged logs out of order")
		}
	}
	_, filtered := m.Snapshot(1)
	if len(filtered) != 3 {
		t.Fatalf("expected start, output, exit: %+v", filtered)
	}
	for _, entry := range filtered {
		if entry.Service != "web" {
			t.Fatal("service filter leaked other logs")
		}
	}
}

func TestLogBufferBoundsAndTerminalSanitizing(t *testing.T) {
	var buffer logBuffer
	for i := 0; i < LogCapacity+5; i++ {
		buffer.append(Log{Sequence: uint64(i)})
	}
	entries := buffer.snapshot()
	if len(entries) != LogCapacity || entries[0].Sequence != 5 {
		t.Fatal("log ring lost ordering or limit")
	}
	var lines []string
	w := &lineWriter{emit: func(s string) { lines = append(lines, s) }}
	_, _ = w.Write([]byte("\x1b[31mred\x1b[0m\r\npartial"))
	w.Flush()
	if strings.Join(lines, "|") != "red|partial" {
		t.Fatalf("got %q", lines)
	}
	lines = nil
	_, _ = w.Write([]byte(strings.Repeat("x", maxLineBytes*2+1)))
	w.Flush()
	if len(lines) != 3 || len(lines[0]) != maxLineBytes {
		t.Fatal("long output is not bounded")
	}
}
