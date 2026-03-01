# Add `server start` Subcommand

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need a `server start` command that creates a detached tmux session named "crabswarm" and runs `serve` inside its first window, decoupling the server process from the user's terminal.

## Plan

### 1. Create `server` parent command

**File**: `cmd/crabswarm/commands/server.go`

Following the `hook.go` pattern — a group command with no `RunE`, registered on `rootCmd`.

### 2. Create `server start` subcommand

**File**: `cmd/crabswarm/commands/server_start.go`

Steps:
1. Resolve current binary path via `os.Executable()` + `filepath.EvalSymlinks()`
2. Resolve socket path via existing `resolveSocketPath(cmd)`
3. Build serve command string: `<binary> serve --sock <path>`
4. Create detached tmux session via `tmux.New(ctx, tmux.Config{Name: "crabswarm", StartupKeys: [serveCmd, "Enter"]})`
5. If `mux.ErrSessionExists` → print informative message, exit 0
6. On success → print session name to stdout

Session name `"crabswarm"` stored as `const defaultSessionName` for future configurability.

### Files

| File | Action |
|------|--------|
| `cmd/crabswarm/commands/server.go` | Create |
| `cmd/crabswarm/commands/server_start.go` | Create |

### Reused code

- `pkg/mux/tmux.New()` — `pkg/mux/tmux/tmux.go:45`
- `resolveSocketPath()` — `cmd/crabswarm/commands/sockpath.go:14`
- `mux.ErrSessionExists` — `pkg/mux/errors.go`

## Verification

1. `go build ./cmd/crabswarm` — compiles
2. `go vet ./cmd/crabswarm/...` — no issues
3. Manual: `crabswarm server start` → `tmux ls` shows "crabswarm" session, first pane runs `serve`
4. Re-run → prints "already running" message, exits 0
