//go:build darwin || linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"allportdev/internal/config"
	"allportdev/internal/runner"
	"allportdev/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "allportdev:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("allportdev", flag.ContinueOnError)
	path := flags.String("config", "allportdev.yaml", "service configuration file")
	initConfig := flags.Bool("init", false, "create an example allportdev.yaml without overwriting an existing file")
	noStart := flags.Bool("no-start", false, "open dashboard with every service stopped")
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), "Usage: allportdev [flags] [service names...]\n\nRun every autostart service, or only the named services.\n\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *initConfig {
		return writeExample(*path)
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return fmt.Errorf("%w (create a config with allportdev -init)", err)
	}
	indices, err := startupServices(cfg.Services, flags.Args(), *noStart)
	if err != nil {
		return err
	}
	manager := runner.New(cfg.Services)
	defer manager.Shutdown()
	program := tea.NewProgram(tui.New(manager), tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithoutSignalHandler())
	quitSignals := make(chan os.Signal, 1)
	signal.Notify(quitSignals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(quitSignals)
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-quitSignals:
			program.Quit()
		case <-finished:
		}
	}()
	for _, index := range indices {
		_ = manager.Start(index)
	}
	_, err = program.Run()
	return err
}

func startupServices(services []config.Service, names []string, noStart bool) ([]int, error) {
	selected := make(map[string]bool)
	for _, name := range names {
		selected[name] = true
	}
	var indices []int
	for i, s := range services {
		start := s.StartsAutomatically()
		if len(names) > 0 {
			start = selected[s.Name]
		}
		delete(selected, s.Name)
		if start && !noStart {
			indices = append(indices, i)
		}
	}
	for name := range selected {
		return nil, fmt.Errorf("unknown service %q", name)
	}
	return indices, nil
}

func writeExample(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(exampleConfig)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Println("Created", path, "— edit the service directories and commands, then run allportdev.")
	return nil
}

const exampleConfig = `services:
  - name: backend
    dir: ./backend
    command: go run .
    env:
      PORT: "8080"

  - name: frontend
    dir: ./frontend
    command: npm run dev

  - name: worker
    dir: ./backend
    command: go run ./cmd/worker
    autostart: false
`
