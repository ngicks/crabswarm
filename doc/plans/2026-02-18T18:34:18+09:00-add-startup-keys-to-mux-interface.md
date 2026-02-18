# Add `StartupKeys()` to mux interfaces and merge `NewWindowWithKeys` into `NewWindow`

## Context

The `mux.Session` and `mux.Window` interfaces lack any notion of startup keys. Startup keys are currently a tmux-specific concern: `NewWindowWithKeys` lives only on the concrete `*Session` type, not the interface. This means generic code using `mux.Session` cannot create windows with startup keys or query what keys are configured. Additionally, `NewWindow` should subsume `NewWindowWithKeys` so there's one method on the interface.

## Files to modify

### 1. `pkg/mux/session.go` — interface changes

- Add `StartupKeys() []string` to `Session` with doc: returns session-level keys sent to every new pane before window-level keys
- Add `StartupKeys() []string` to `Window` with doc: returns window-level keys sent to every new pane after session-level keys
- Change `NewWindow` signature to `NewWindow(ctx context.Context, name string, startupKeys []string) (Window, error)` with doc explaining startupKeys are window-level, pass `nil` for none, persisted for future `Split` calls

### 2. `pkg/mux/tmux/tmux.go` — Session implementation

- Add `StartupKeys()` method returning `slices.Clone(s.startupKeys)` (defensive copy)
- **Merge `NewWindowWithKeys` body into `NewWindow`** with the new 3-arg signature
- **Delete `NewWindowWithKeys`** entirely

### 3. `pkg/mux/tmux/window.go` — Window implementation

- Add `StartupKeys() []string` method returning `slices.Clone(w.startupKeys)` (defensive copy, window-level only)

### 4. `pkg/mux/tmux/tmux_test.go` — call site updates + new tests

**Call site updates (15 sites):**
- 10 `sess.NewWindow(ctx, name)` calls → add `nil` as third arg
- 5 `sess.NewWindowWithKeys(ctx, name, keys)` calls → change to `sess.NewWindow(ctx, name, keys)`

**New tests for `StartupKeys()` accessor:**
- `TestSessionStartupKeys`: create session with `Config.StartupKeys`, verify `sess.StartupKeys()` returns correct values
- `TestSessionStartupKeysDefensiveCopy`: verify mutating the returned slice does not affect session state
- `TestWindowStartupKeys`: create window via `NewWindow(ctx, name, keys)`, verify `w.StartupKeys()` returns the window-level keys
- `TestWindowStartupKeysDefensiveCopy`: verify mutating the returned slice does not affect window state
- `TestWindowStartupKeysViaList`: create window with keys, re-acquire via `sess.GetById()`, verify `w.StartupKeys()` still returns correct keys

**Validate initial window from `New()` semantics:**
- `TestInitialWindowStartupKeysEmpty`: after `New()`, get initial window via `GetAt(0)`, verify `w.StartupKeys()` returns nil/empty (initial window has no window-level keys, only session-level)

### 5. `pkg/mux/tmux/internal/cmd/splitpane/main.go` — all references

- `sess.NewWindowWithKeys(ctx, name, windowKeys)` → `sess.NewWindow(ctx, name, windowKeys)` (line 120)
- `log.Fatalf("NewWindowWithKeys(%s): %v", ...)` → `log.Fatalf("NewWindow(%s): %v", ...)` (line 122)
- Package doc comment: update `-windows` flag description from `NewWindowWithKeys` to `NewWindow` (line 13)
- Flag help comment: update `-windows` help text (line 43)

## Verification

```bash
go build ./pkg/mux/...
go test ./pkg/mux/tmux/ -v -count=1
```
