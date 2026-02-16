# Fix uneven pane sizes on client attach/detach via tmux hooks

## Context

When `Split` runs on a detached tmux session, tmux uses `default-size` (80x24). When a real client attaches, the window resizes to the client's terminal dimensions, but pane proportions get distorted (e.g. 85:19 instead of 50:50). The split algorithm itself is correct — the issue is tmux's layout reflow on resize.

The fix: install persistent tmux hooks at session creation that run `select-layout -t <window> tiled` for **every window** in the session on `client-attached` and `client-detached` events.

## Why `client-attached`/`client-detached` and not `after-resize-window`

- `after-resize-window` fires on **every** terminal resize (e.g. dragging the terminal edge), which would be disruptive during interactive use.
- `client-attached` fires once when a client connects, `client-detached` fires once when a client disconnects. These are the events that cause tmux to recalculate window size from the attached client set, which is exactly when distortion occurs.

## Key detail: `select-layout` scope

`select-layout tiled` without `-t` only affects the **current window**. The hook must iterate all windows. tmux `run-shell` can execute a shell loop:

```
run-shell 'for w in $(tmux list-windows -t <session> -F "##{window_id}"); do tmux select-layout -t "$w" tiled; done'
```

**Critical**: `run-shell` expands `#{...}` formats before invoking `/bin/sh`. To pass a literal `#{window_id}` to `list-windows -F`, escape the `#` as `##` → `"##{window_id}"`.

This ensures every window in the session is rebalanced, not just the active one.

## Avoiding hook conflicts

tmux hooks are arrays. `set-hook hook-name cmd` without an index **clears the entire array** and sets index 0. To avoid clobbering user-defined hooks from `tmux.conf`, use an explicit array index:

```
set-hook -t <session> client-attached[100] 'run-shell ...'
set-hook -t <session> client-detached[100] 'run-shell ...'
```

Index 100 avoids typical user indices (0-based, usually low). Collision at exactly [100] is possible but unlikely.

## Accepted trade-off

This force-resets the layout to `tiled` on every attach/detach. The user explicitly requested this persistent behavior. Users who manually adjust pane layouts will have them reset on next attach/detach.

## Files to modify

### `pkg/mux/tmux/tmux.go` — In `New()`, after creating the session

Replace the existing `set-hook` call (which uses bare `after-resize-window` without an index) with:

```go
// Install hooks to rebalance all window panes on client attach/detach.
// Splits done on detached sessions use default-size (80x24) and get
// distorted when the window resizes to the real terminal dimensions.
// Using run-shell to iterate all windows so every window is rebalanced.
// Array index [100] avoids clobbering user-defined hooks.
//
// NOTE: ## escapes # for tmux format expansion in run-shell.
// All dynamic values are shell-quoted with shellQuote() to prevent injection.
rebalanceCmd := fmt.Sprintf(
    "run-shell 'for w in $(%s %slist-windows -t %s -F \"##{window_id}\"); do %s %sselect-layout -t \"$w\" tiled; done'",
    shellQuote(exec.tmuxPath), exec.socketFlag(),
    shellQuote(cfg.Name),
    shellQuote(exec.tmuxPath), exec.socketFlag(),
)
for _, hook := range []string{"client-attached[100]", "client-detached[100]"} {
    _, err = exec.run(ctx, "set-hook", "-t", cfg.Name, hook, rebalanceCmd)
    if err != nil {
        return nil, err
    }
}
```

### `pkg/mux/tmux/exec.go` — Add helpers

```go
// socketFlag returns the "-L <name> " flag string for use in shell commands,
// or "" if no socket is configured.
func (e *executor) socketFlag() string {
    if e.socketName != "" {
        return "-L " + shellQuote(e.socketName) + " "
    }
    return ""
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes.
// This is safe for embedding in sh(1) commands.
func shellQuote(s string) string {
    return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
```

### `pkg/mux/tmux/tmux_test.go` — Add `TestTmuxNewInstallsHooks`

Use `show-hooks -t <session>` to verify `client-attached[100]` and `client-detached[100]` are present. Hooks won't fire in tests (no real client attach), but registration can be verified.

## Verification

```
go test ./pkg/mux/tmux/ -v -count=1
```

Manually verify with the demo tool:
```
go run ./pkg/mux/tmux/internal/cmd/splitpane/ -n 7
# After attaching, run: tmux list-panes
# All panes should be approximately equal size
```
