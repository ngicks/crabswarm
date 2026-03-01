# Add `server start` Subcommand + Config.DisallowReuse

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need a `server start` command that creates a detached tmux session named "crabswarm" and runs `serve` inside its first window, decoupling the server process from the user's terminal.

`tmux.Config` needs a `DisallowReuse` field: when false (default/zero value), `New()` attaches to an existing session instead of failing; when true, `New()` returns `mux.ErrSessionExists`.

## Plan

### 1. Add `DisallowReuse` to `tmux.Config` and update `New()`

**File**: `pkg/mux/tmux/tmux.go`

Add field to `Config`:
```go
// DisallowReuse controls behavior when a session with the same name exists.
// If false (default), New attaches to the existing session instead of failing.
// If true, New returns mux.ErrSessionExists.
DisallowReuse bool
```

Update `New()` duplicate detection block:
```go
if strings.Contains(err.Error(), "duplicate session") {
    if cfg.DisallowReuse {
        return nil, mux.ErrSessionExists
    }
    return Attach(ctx, cfg)
}
```

### 2. Update existing tests

**File**: `pkg/mux/tmux/tmux_test.go`

**Line ~92**: The test for `ErrSessionExists` must set `DisallowReuse: true` in its config before the second `New()` call (or use a config with `DisallowReuse: true`).

**Add new test**: Call `New()` twice with same name and `DisallowReuse: false` → second call returns session via attach, no error.

**Call-site audit** (only non-test callers):
- `pkg/mux/tmux/internal/cmd/splitpane/main.go:69` — internal demo tool, no change needed (attach-on-reuse is acceptable)

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
5. Create/reuse tmux session (default `DisallowReuse: false`):
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName, // "crabswarm"
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
   - StartupKeys are only sent on fresh creation, not on attach/reuse
6. Print session status to `cmd.OutOrStdout()`

### Files

| File | Action |
|------|--------|
| `pkg/mux/tmux/tmux.go` | Modify — add `DisallowReuse` to Config, update `New()` |
| `pkg/mux/tmux/tmux_test.go` | Modify — set `DisallowReuse: true` in duplicate test (~line 92), add reuse test |
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
5. Re-run `crabswarm server start` → reuses existing session, prints status, exits 0
