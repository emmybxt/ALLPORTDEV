//go:build darwin || linux

package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"time"

	"allportdev/internal/config"
)

type Status string

const (
	Stopped  Status = "stopped"
	Running  Status = "running"
	Stopping Status = "stopping"
	Exited   Status = "exited"
	Failed   Status = "failed"
)

type ServiceView struct {
	Name   string
	Status Status
	PID    int
}

type process struct {
	config config.Service
	status Status
	cmd    *exec.Cmd
	done   chan struct{}
	logs   logBuffer
}

type Manager struct {
	mu          sync.Mutex
	services    []*process
	sequence    uint64
	closed      bool
	stopTimeout time.Duration
}

func New(services []config.Service) *Manager {
	m := &Manager{stopTimeout: 3 * time.Second}
	for _, s := range services {
		m.services = append(m.services, &process{config: s, status: Stopped})
	}
	return m
}

func (m *Manager) Start(index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.services[index]
	if m.closed {
		return fmt.Errorf("runner is shutting down")
	}
	if p.status == Running || p.status == Stopping {
		return nil
	}
	cmd := exec.Command("/bin/sh", "-c", p.config.Command)
	cmd.Dir = p.config.Dir
	cmd.Env = serviceEnv(p.config.Env)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// A child inheriting stdout must not keep Wait blocked after its shell exits.
	cmd.WaitDelay = 250 * time.Millisecond
	stdout := m.writer(p, "out")
	stderr := m.writer(p, "err")
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Start(); err != nil {
		p.status = Failed
		m.appendLog(p, "sys", err.Error())
		return err
	}
	p.cmd, p.done, p.status = cmd, make(chan struct{}), Running
	m.appendLog(p, "sys", fmt.Sprintf("started (pid %d)", cmd.Process.Pid))
	go m.wait(p, cmd, stdout, stderr)
	return nil
}

func serviceEnv(overrides map[string]string) []string {
	env := os.Environ()
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func (m *Manager) writer(p *process, stream string) *lineWriter {
	return &lineWriter{emit: func(text string) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.appendLog(p, stream, text)
	}}
}

// appendLog is called with the manager lock held.
func (m *Manager) appendLog(p *process, stream, text string) {
	m.sequence++
	p.logs.append(Log{Sequence: m.sequence, Time: time.Now(), Service: p.config.Name, Stream: stream, Text: text})
}

func (m *Manager) wait(p *process, cmd *exec.Cmd, stdout, stderr *lineWriter) {
	err := cmd.Wait()
	stdout.Flush()
	stderr.Flush()
	m.mu.Lock()
	defer m.mu.Unlock()
	// Commands often spawn watchers. Clean up their group even if the shell exited first.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	p.status = exitStatus(p.status, err)
	message := string(p.status)
	if err != nil {
		message += ": " + err.Error()
	}
	m.appendLog(p, "sys", message)
	p.cmd = nil
	close(p.done)
}

func exitStatus(previous Status, err error) Status {
	if previous == Stopping {
		return Stopped
	}
	if err != nil {
		return Failed
	}
	return Exited
}

func (m *Manager) Stop(index int) error {
	m.mu.Lock()
	p := m.services[index]
	if p.cmd == nil {
		m.mu.Unlock()
		return nil
	}
	done := p.done
	if p.status == Stopping {
		m.mu.Unlock()
		<-done
		return nil
	}
	p.status = Stopping
	err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
	m.mu.Unlock()
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	timer := time.NewTimer(m.stopTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		m.mu.Lock()
		if p.cmd != nil && p.done == done {
			_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		}
		m.mu.Unlock()
		<-done
	}
	return nil
}

func (m *Manager) Restart(index int) error {
	if err := m.Stop(index); err != nil {
		return err
	}
	return m.Start(index)
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	var wg sync.WaitGroup
	for i := range m.services {
		wg.Add(1)
		go func(index int) { defer wg.Done(); _ = m.Stop(index) }(i)
	}
	wg.Wait()
}

// Snapshot uses -1 for the merged log view.
func (m *Manager) Snapshot(filter int) ([]ServiceView, []Log) {
	m.mu.Lock()
	defer m.mu.Unlock()
	views := make([]ServiceView, 0, len(m.services))
	var logs []Log
	for i, p := range m.services {
		view := ServiceView{Name: p.config.Name, Status: p.status}
		if p.cmd != nil {
			view.PID = p.cmd.Process.Pid
		}
		views = append(views, view)
		if filter == -1 || filter == i {
			logs = append(logs, p.logs.snapshot()...)
		}
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].Sequence < logs[j].Sequence })
	return views, logs
}
