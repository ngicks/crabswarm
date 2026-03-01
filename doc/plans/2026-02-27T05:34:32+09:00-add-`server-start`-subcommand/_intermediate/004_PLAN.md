# Add `server start` Subcommand + Config.AllowReuse

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need a `server start` command that creates a detached tmux session named "crabswarm" and runs `serve` inside its first window, decoupling the server process from the user's terminal.

Additionally, `tmux.Config` needs an `AllowReuse` field so callers can opt in to reusing an existing session.

## Plan

### 1. Add `AllowReuse` to `tmux.Config` and update `New()`

**File**: `pkg/mux/tmux/tmux.go`

Add field to `Config`:
```go
type Config struct {
    // ... existing fields ...
    // AllowReuse controls behavior when a session with the same name already exists.
    // If false (default), New returns mux.ErrSessionExists.
    // If true, New attaches to the existing session instead of failing.
    AllowReuse bool
}
```

Update `New()`: when duplicate session is detected and `AllowReuse` is true, fall through to `Attach()` instead of returning an error.

```go
if strings.Contains(err.Error(), "duplicate session") {
    if cfg.AllowReuse {
        return Attach(ctx, cfg)
    }
    return nil, mux.ErrSessionExists
}
```

**No existing callers or tests need updating** — the default (`false`) preserves current behavior.

### 2. Add test for AllowReuse

**File**: `pkg/mux/tmux/tmux_test.go`

Add test: create a session, then call `New()` with `AllowReuse: true` on the same name → should return session without error.

### 3. Create `server` parent command

**File**: `cmd/crabswarm/commands/server.go`

- Group command with no `RunE`, registered on `rootCmd` via `init()`
- No `SilenceErrors`/`SilenceUsage`

### 4. Create `server start` subcommand

**File**: `cmd/crabswarm/commands/server_start.go`

Registration: `serverCmd.AddCommand(serverStartCmd)` in `init()`

Implementation of `runServerStart`:

1. Resolve binary path via `os.Executable()` + `filepath.EvalSymlinks()`
2. Resolve socket path via `resolveSocketPath(cmd)`
3. **Shell-quote** both paths (replicate `shellQuote` pattern locally since unexported)
4. Build startup key: `shellQuote(exe) + " serve --sock " + shellQuote(sockPath)`
5. Create/reuse tmux session with `AllowReuse: true`:
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName, // "crabswarm"
       AllowReuse:  true,
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
   - Note: StartupKeys are only sent on fresh creation, not on attach/reuse
6. Print session status to `cmd.OutOrStdout()`

### Files

| File | Action |
|------|--------|
| `pkg/mux/tmux/tmux.go` | Modify — add `AllowReuse` to Config, update `New()` |
| `pkg/mux/tmux/tmux_test.go` | Modify — add reuse test |
| `cmd/crabswarm/commands/server.go` | Create — parent `server` command |
| `cmd/crabswarm/commands/server_start.go` | Create — `server start` implementation |

### Reused code

- `tmux.New()` / `tmux.Attach()` — `pkg/mux/tmux/tmux.go`
- `resolveSocketPath()` — `cmd/crabswarm/commands/sockpath.go:14`
- Shell quoting pattern from `pkg/mux/tmux/exec.go:58`

## Verification

1. `go test ./pkg/mux/tmux/...` — existing tests unaffected + new AllowReuse test passes
2. `go build ./cmd/crabswarm` — compiles
3. `go vet ./cmd/crabswarm/...` — no issues
4. Manual: `crabswarm server start` → `tmux ls` shows "crabswarm", first pane runs `crabswarm serve`
5. Re-run `crabswarm server start` → reuses existing session, prints status, exits 0
