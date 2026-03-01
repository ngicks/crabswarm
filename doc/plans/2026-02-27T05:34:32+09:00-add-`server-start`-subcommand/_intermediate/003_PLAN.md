# Add `server start` Subcommand + Config.DisallowReuse

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need a `server start` command that creates a detached tmux session named "crabswarm" and runs `serve` inside its first window, decoupling the server process from the user's terminal.

Additionally, `tmux.Config` needs a `DisallowReuse` field: when false (zero value), `New()` attaches to an existing session instead of erroring; when true, `New()` fails on duplicate session name (current behavior).

## Plan

### 1. Add `DisallowReuse` to `tmux.Config` and update `New()`

**File**: `pkg/mux/tmux/tmux.go`

Add field to `Config`:
```go
type Config struct {
    // ... existing fields ...
    // DisallowReuse controls behavior when a session with the same name exists.
    // If false (default), New attaches to the existing session instead of failing.
    // If true, New returns mux.ErrSessionExists.
    DisallowReuse bool
}
```

Update `New()`: when `ErrSessionExists` is detected and `DisallowReuse` is false, fall through to `Attach()` instead of returning an error.

```go
func New(ctx context.Context, cfg Config) (*Session, error) {
    exec := newExecutor(cfg.TmuxPath, cfg.SocketName)

    out, err := exec.run(ctx, "new-session", "-d", "-s", cfg.Name, "-P", "-F", "...")
    if err != nil {
        if strings.Contains(err.Error(), "duplicate session") {
            if cfg.DisallowReuse {
                return nil, mux.ErrSessionExists
            }
            return Attach(ctx, cfg)
        }
        return nil, err
    }
    // ... rest of existing New() logic unchanged ...
}
```

### 2. Update existing tests

**File**: `pkg/mux/tmux/tmux_test.go`

- Existing tests that expect `mux.ErrSessionExists` from `New()` must set `DisallowReuse: true` in their config, since the default behavior now attaches.
- Add a test for the reuse path: create a session, then call `New()` with `DisallowReuse: false` → should return session without error.

### 3. Create `server` parent command

**File**: `cmd/crabswarm/commands/server.go`

- Group command with no `RunE`, registered on `rootCmd` via `init()`
- No `SilenceErrors`/`SilenceUsage` (those are hook-specific)

### 4. Create `server start` subcommand

**File**: `cmd/crabswarm/commands/server_start.go`

Registration: `serverCmd.AddCommand(serverStartCmd)` in `init()`

Implementation of `runServerStart`:

1. Resolve binary path via `os.Executable()` + `filepath.EvalSymlinks()`
2. Resolve socket path via `resolveSocketPath(cmd)`
3. **Shell-quote** both paths (replicate `shellQuote` from `pkg/mux/tmux/exec.go:58` locally since it's unexported)
4. Build startup key: `shellQuote(exe) + " serve --sock " + shellQuote(sockPath)`
5. Create/reuse tmux session with `DisallowReuse: false` (default):
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName, // "crabswarm"
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
   - If session already exists, `New()` attaches and returns it (no error)
   - Note: StartupKeys are only sent on fresh session creation, not on attach/reuse
6. Print session status to `cmd.OutOrStdout()`

### Files

| File | Action |
|------|--------|
| `pkg/mux/tmux/tmux.go` | Modify — add `DisallowReuse` to Config, update `New()` |
| `pkg/mux/tmux/tmux_test.go` | Modify — fix tests expecting `ErrSessionExists`, add reuse test |
| `cmd/crabswarm/commands/server.go` | Create — parent `server` command |
| `cmd/crabswarm/commands/server_start.go` | Create — `server start` implementation |

### Reused code

- `tmux.New()` / `tmux.Attach()` — `pkg/mux/tmux/tmux.go`
- `resolveSocketPath()` — `cmd/crabswarm/commands/sockpath.go:14`
- Shell quoting pattern from `pkg/mux/tmux/exec.go:58`

## Verification

1. `go test ./pkg/mux/tmux/...` — existing + new tests pass
2. `go build ./cmd/crabswarm` — compiles
3. `go vet ./cmd/crabswarm/...` — no issues
4. Manual: `crabswarm server start` → `tmux ls` shows "crabswarm", first pane runs `crabswarm serve`
5. Re-run `crabswarm server start` → attaches to existing session, prints status, exits 0
