# Add `server start` Subcommand

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need a `server start` command that creates a detached tmux session named "crabswarm" and runs `serve` inside its first window, decoupling the server process from the user's terminal.

## Plan

### 1. Create `server` parent command

**File**: `cmd/crabswarm/commands/server.go`

- Group command with no `RunE`, registered on `rootCmd` via `init()`
- No `SilenceErrors`/`SilenceUsage` (those are hook-specific in `hook.go`)

### 2. Create `server start` subcommand

**File**: `cmd/crabswarm/commands/server_start.go`

Registration: `serverCmd.AddCommand(serverStartCmd)` in `init()`

Implementation of `runServerStart`:

1. Resolve binary path via `os.Executable()` + `filepath.EvalSymlinks()`
2. Resolve socket path via `resolveSocketPath(cmd)`
3. **Shell-quote** both binary path and socket path when building the command string (use single-quote wrapping like `shellQuote` in `pkg/mux/tmux/exec.go:58`, replicated locally since it's unexported)
4. Build startup key: `shellQuote(exe) + " serve --sock " + shellQuote(sockPath)`
5. Create detached tmux session:
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName, // "crabswarm"
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
6. Handle `errors.Is(err, mux.ErrSessionExists)` → print message, return nil (accepted limitation: does not verify the session is actually running crabswarm serve)
7. Handle other errors → return error
8. On success → print `fmt.Fprintf(cmd.OutOrStdout(), ...)` with session name

### Files

| File | Action |
|------|--------|
| `cmd/crabswarm/commands/server.go` | Create — parent `server` command |
| `cmd/crabswarm/commands/server_start.go` | Create — `server start` implementation |

### Reused code

- `tmux.New()` — `pkg/mux/tmux/tmux.go:45`
- `resolveSocketPath()` — `cmd/crabswarm/commands/sockpath.go:14`
- `mux.ErrSessionExists` — `pkg/mux/errors.go`
- Shell quoting pattern from `pkg/mux/tmux/exec.go:58` (replicated since unexported)

### Known limitations

- If a tmux session named "crabswarm" already exists from a different source, `server start` will report it as already running. Accepted for now; future config support will allow custom session names.

## Verification

1. `go build ./cmd/crabswarm` — compiles
2. `go vet ./cmd/crabswarm/...` — no issues
3. Manual: `crabswarm server start` → `tmux ls` shows "crabswarm", first pane runs `crabswarm serve`
4. Re-run → prints "already running" message, exits 0
