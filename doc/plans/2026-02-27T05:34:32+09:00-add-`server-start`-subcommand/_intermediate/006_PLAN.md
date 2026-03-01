# Add `server start` Subcommand + Config.DisallowReuse + Socket Lock

## Context

The `serve` subcommand starts the crabswarm gRPC server in the foreground. We need:
1. A file lock on the socket file so `serve` can detect if another server is already running
2. A `server start` command that creates a detached tmux session and runs `serve` inside it
3. A `DisallowReuse` field on `tmux.Config` to control duplicate session behavior

## Plan

### 1. Add file lock to `serve`

**File**: `pkg/crabswarm/server.go`

Before listening, acquire an exclusive flock on a lockfile (`<sockPath>.lock`) next to the socket:

```go
func (s *Server) Serve(ctx context.Context) error {
    // Acquire lock on <sockPath>.lock
    lockPath := s.sockPath + ".lock"
    lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
    if err != nil {
        return fmt.Errorf("open lock file: %w", err)
    }
    defer lockFile.Close()

    if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
        return fmt.Errorf("server already running (lock held on %s)", lockPath)
    }
    defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

    // ... existing listen + serve logic ...
}
```

- Uses `LOCK_NB` (non-blocking) — fails immediately if lock is held
- Lock is released when the process exits (or explicitly via defer)
- Lockfile is separate from socket file so `os.Remove` of the socket doesn't affect the lock

### 2. Add `DisallowReuse` to `tmux.Config`

**File**: `pkg/mux/tmux/tmux.go`

```go
// DisallowReuse controls behavior when a session with the same name exists.
// If false (default), New attaches to the existing session instead of failing.
// If true, New returns mux.ErrSessionExists.
DisallowReuse bool
```

Update `New()`:
```go
if strings.Contains(err.Error(), "duplicate session") {
    if cfg.DisallowReuse {
        return nil, mux.ErrSessionExists
    }
    return Attach(ctx, cfg)
}
```

### 3. Update tests for DisallowReuse

**File**: `pkg/mux/tmux/tmux_test.go`

- Line ~92: set `DisallowReuse: true` on the config before testing duplicate `New()` → `ErrSessionExists`
- Add test: `New()` twice with `DisallowReuse: false` → second call succeeds via attach

### 4. Create `server` parent command

**File**: `cmd/crabswarm/commands/server.go`

Group command, no `RunE`, registered on `rootCmd`.

### 5. Create `server start` subcommand

**File**: `cmd/crabswarm/commands/server_start.go`

Implementation:

1. Resolve binary path via `os.Executable()` + `filepath.EvalSymlinks()`
2. Resolve socket path via `resolveSocketPath(cmd)`
3. **Check if server already running**: try `flock` on `<sockPath>.lock` (non-blocking). If lock is held → print "server already running", exit 0. If lock is acquired → release it immediately and proceed to start.
4. Shell-quote both paths for tmux startup key
5. Create/reuse tmux session (`DisallowReuse: false` — default):
   ```go
   _, err := tmux.New(ctx, tmux.Config{
       Name:        defaultSessionName,
       StartupKeys: []string{serveCmd, "Enter"},
   })
   ```
6. Print status to `cmd.OutOrStdout()`

### Files

| File | Action |
|------|--------|
| `pkg/crabswarm/server.go` | Modify — add flock on `<sockPath>.lock` before listen |
| `pkg/mux/tmux/tmux.go` | Modify — add `DisallowReuse` to Config, update `New()` |
| `pkg/mux/tmux/tmux_test.go` | Modify — fix duplicate test, add reuse test |
| `cmd/crabswarm/commands/server.go` | Create — parent `server` command |
| `cmd/crabswarm/commands/server_start.go` | Create — `server start` implementation |

### Reused code

- `tmux.New()` / `tmux.Attach()` — `pkg/mux/tmux/tmux.go`
- `resolveSocketPath()` — `cmd/crabswarm/commands/sockpath.go:14`
- Shell quoting pattern from `pkg/mux/tmux/exec.go:58`

## Verification

1. `go test ./pkg/mux/tmux/...` — all tests pass
2. `go build ./cmd/crabswarm` — compiles
3. `go vet ./...` — no issues
4. Manual: `crabswarm server start` → tmux session created, `serve` running, lock held
5. Re-run `crabswarm server start` → detects lock, prints "already running", exits 0
6. Kill server → lock released → `crabswarm server start` creates fresh session
