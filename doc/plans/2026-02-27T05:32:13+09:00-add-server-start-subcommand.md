# Add `server start` Subcommand

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. To support a managed server lifecycle, we need a `server start` command that creates a tmux session named "crabswarm" and runs `serve` inside its first window. This decouples the server process from the user's terminal.

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
3. **Shell-quote** both binary path and socket path when building the command string (replicate `shellQuote` from `pkg/mux/tmux/exec.go:58` locally since it's unexported)
4. Build startup key: `shellQuote(exe) + " serve --sock " + shellQuote(sockPath)`
5. Create detached tmux session:
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName, // "crabswarm"
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
6. Handle `errors.Is(err, mux.ErrSessionExists)` → print message, return nil
7. Handle other errors → return error
8. On success → print session info via `fmt.Fprintf(cmd.OutOrStdout(), ...)`

### Files to create

| File | Action |
|------|--------|
| `cmd/crabswarm/commands/server.go` | **Create** — parent `server` command |
| `cmd/crabswarm/commands/server_start.go` | **Create** — `server start` implementation |

### Existing code to reuse

- `tmux.New()` — creates detached tmux session with startup keys (`pkg/mux/tmux/tmux.go:45`)
- `resolveSocketPath()` — resolves `--sock` flag / env / default (`cmd/crabswarm/commands/sockpath.go:14`)
- `mux.ErrSessionExists` — sentinel error for duplicate session (`pkg/mux/errors.go`)
- Shell quoting pattern from `pkg/mux/tmux/exec.go:58` (replicated since unexported)

### Known limitations

- If a tmux session named "crabswarm" already exists from a different source, `server start` will report it as already running. Accepted for now; future config support will allow custom session names.

## Verification

1. `go build ./cmd/crabswarm` — confirms compilation
2. `go vet ./cmd/crabswarm/...` — no vet issues
3. Manual test: `crabswarm server start` → verify tmux session "crabswarm" exists with `tmux ls`, verify `crabswarm serve` is running in the first pane with `tmux capture-pane`
4. Run again → should print "already running" message instead of erroring

## Review feedback addressed

- **Shell safety**: Command string now uses shell-quoting for binary path and socket path
- **Registration wiring**: Explicitly documents `serverCmd.AddCommand(serverStartCmd)`
- **SilenceErrors clarification**: Plan notes that `server` parent does NOT use `SilenceErrors`/`SilenceUsage`
- **Session collision**: Documented as accepted limitation
