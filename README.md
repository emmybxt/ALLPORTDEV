# AllPortDev

One terminal for your local development services. Run backend, frontend, and workers together, see their output in one dashboard, and filter logs by service.

Supports **macOS and Linux**. Requires Go 1.24+ to build and an interactive terminal to run.

## Try it

```sh
go build -o bin/allportdev .
./bin/allportdev -config examples/demo.yaml
```

The demo starts two shell-based services; select `worker` and press `s` to start the third. No application setup is required.

To install the command in your Go binary directory:

```sh
go install .
```

Ensure `$(go env GOPATH)/bin` is on your `PATH` (or use your configured `GOBIN`).

## Configure your projects

Run `allportdev -init` to create `allportdev.yaml`, then edit the directories and commands:

```yaml
services:
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
```

Directories resolve relative to the YAML file, so services can live in different repositories using paths such as `../my-api` or absolute paths. Omit `dir` to use the configuration directory. Environment overrides extend your current shell's environment; quote numeric values. Keep secrets out of shared configuration files.

Commands run through `/bin/sh -c`, supporting arguments, pipes, and shell operators. Use foreground commands such as `npm run dev`, `air`, or `go run .`. Interactive input, shell aliases, detached daemons, and containers' internal process lifecycles are not managed. Only run configuration files you trust: their commands execute on your machine.

## Run

```sh
allportdev                              # Start services with autostart enabled (default)
allportdev backend frontend             # Start only these services
allportdev -no-start                     # Open the dashboard with everything stopped
allportdev -config /path/to/dev.yaml     # Use a different configuration
```

Place flags before service names. Services launch without waiting for one another to become ready. All configured services remain available in the dashboard, including ones not started initially. `autostart: false` affects startup only; `s` on All starts every configured service.

## Dashboard controls

| Key | Action |
| --- | --- |
| Tab / Shift+Tab | Select a service and show its logs |
| a | Select All and merge every service's logs |
| s | Start the selection |
| x | Stop the selection |
| r | Restart the selection |
| / | Search log text, case insensitive; Enter applies |
| Esc | Clear the search |
| m | Toggle dashboard mouse controls (service clicks and wheel scrolling); off by default |
| c | Optional paused, full-width copy view |
| ↑ / ↓, k / j | Scroll the selected service's log history |
| PgUp / PgDn | Scroll log history one page at a time |
| End / G | Return to live logs |
| q / Ctrl+C | Quit and stop services |

Start, stop, and restart apply to **all services when All is selected**. Status labels distinguish running, stopping, stopped, exited, and failed commands. A running process is not a readiness/health check.

Scrolling keeps the current service selected and pauses live following so new output does not move the lines you are reading. Press End (or scroll back to the bottom) to resume live logs. Press **m** to enable service clicks and wheel scrolling in terminals with mouse reporting support; press **m** again to return the mouse to native text selection.

**Drag to highlight and copy normally—no mode key is required.** By default the dashboard leaves the mouse to your terminal. Use your terminal's Copy command (`Cmd+C` on macOS, often `Ctrl+Shift+C` on Linux). Select services with Tab/Shift+Tab and scroll with the arrow keys.

For a stable, full-width view without the sidebar, optionally press **c**. This freezes the displayed history while services keep running. Use arrow keys or PgUp/PgDn to browse before selecting. Press **Esc** or **c** to restore the dashboard and your previous mouse setting. Plain Ctrl+C does not quit from this optional copy mode; `q` still quits. Copied text is the visible terminal text; lines wider than the terminal remain truncated.

Every log has a timestamp, service name, and `out`, `err`, or `sys` label. The runner keeps the newest 2,000 lines **per service** in memory, splits output at 16 KiB per line, and removes terminal escape sequences. Long lines are truncated to the terminal width in the dashboard. Logs are emitted on a newline/carriage return or when a process exits; programs may need their own unbuffered-output option. Logs are not persisted. Scrolling pauses the view; older entries still expire when the buffer fills.

Stopping sends SIGTERM to the service's process group, waits up to three seconds, then sends SIGKILL. Exiting the dashboard also stops services. Remaining children in the group are cleaned up when the main command exits. Processes that explicitly detach into a different session/group are outside this lifecycle. SIGKILL of the runner itself cannot run cleanup.

## Development

```sh
go test -race ./...
go vet ./...
go build -o bin/allportdev .
```

`internal/config` loads and validates YAML; `internal/runner` owns processes and bounded logs; `internal/tui` renders the dashboard and handles keyboard commands. The UI uses Bubble Tea and Lip Gloss. This version does not include dependency ordering, readiness probes, automatic crash restarts, or Windows support.
