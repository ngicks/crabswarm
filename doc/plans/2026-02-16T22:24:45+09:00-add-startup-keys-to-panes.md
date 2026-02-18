# Add startup keys sent to panes after creation

## Context

Users need a way to run setup commands (e.g. `cd /workspace`, `source .env`, `Enter`) automatically in every new pane. Currently, callers must manually call `SendKeys` on each pane after creation. This feature adds session-level and window-level startup keys that are automatically sent to panes when they are created by `New`, `NewWindow`, or `Split`.

## Design decisions

### Don't change `mux.Session` interface

Startup keys are tmux-specific behavior. Keep the `mux.Session.NewWindow(ctx, name)` signature unchanged. Instead, add a tmux-specific method on `*Session`:

```go
func (s *Session) NewWindowWithKeys(ctx context.Context, name string, startupKeys []string) (mux.Window, error)
```

`NewWindow` delegates to `NewWindowWithKeys` with `nil` keys, satisfying the interface.

### Persist window keys in Session (not just on window struct)

Windows reconstructed via `List()`, `GetAt()`, `GetById()` are rebuilt from `parseWindows()`. If window-level keys are only stored on the `window` struct, they'd be lost on reconstruction.

Solution: `Session` holds a `windowKeys map[string][]string` (keyed by window ID), protected by a `sync.RWMutex`. When constructing any `window` (via `NewWindowWithKeys`, `parseWindows`, etc.), inject the stored keys. Dead window entries are retained lazily (no cleanup on close) — they are small and harmless, and avoiding cleanup keeps `window.Close()` simple since it doesn't need a back-reference to `Session`.

### Concurrency safety

`Session.windowKeys` is a map mutated by `NewWindowWithKeys` (write) and read by `List`/`GetAt`/`GetById`/`Split` (read). A `sync.RWMutex` on `Session` protects all accesses. Write lock in `NewWindowWithKeys`; read lock in methods that construct windows from `parseWindows`.

### Defensive copying

Copy `[]string` slices when storing (config → session, session → window) to prevent caller mutation.

### Partial failure semantics

If sending startup keys fails after pane creation, the error is returned but the pane remains. This is documented and acceptable — callers can retry `SendKeys` manually.

## Files to modify

### 1. `pkg/mux/tmux/tmux.go`

**Config**: Add `StartupKeys []string`.

**Session struct**: Add fields:
```go
mu          sync.RWMutex
startupKeys []string                 // session-level, defensively copied from Config
windowKeys  map[string][]string      // window ID → window-level keys
```

**`New()`**: Change `-P -F` to `"#{session_id}\t#{pane_id}"`, parse both. After hooks, send `startupKeys` to initial pane if non-empty.

**`NewWindow()`**: Delegate to `NewWindowWithKeys(ctx, name, nil)`.

**`NewWindowWithKeys()`** (new): Change `-P -F` to `"#{window_id}\t#{pane_id}"`, parse both. Store window keys in `s.windowKeys[windowID]`. Construct `window` with both key sets. Send session + window keys to initial pane.

**`List()`**, **`GetAt()`**, **`GetById()`**: Pass `s` (or `s.windowKeys` + `s.startupKeys`) into `parseWindows()` so reconstructed windows get the right keys.

### 2. `pkg/mux/tmux/window.go`

**window struct**: Add fields:
```go
sessionStartupKeys []string
startupKeys        []string  // window-level
```

**`Split()`**: Add `-P -F "#{pane_id}"` to `split-window`, capture new pane ID. After each split, send `sessionStartupKeys + startupKeys` to the new pane.

**`Close()`**: No change needed. Dead window entries in `Session.windowKeys` are retained lazily — they are small and harmless.

### 3. `pkg/mux/tmux/parse.go`

**`parseWindows()`**: Accept additional params for startup keys injection:
```go
func parseWindows(out string, sessionName string, exec *executor,
    sessionStartupKeys []string, windowKeys map[string][]string) []mux.Window
```

When constructing each `window`, set `sessionStartupKeys` and look up `windowKeys[windowID]`.

### 4. `pkg/mux/tmux/pane.go`

No changes. Use `pane{id: paneID, exec: exec}.SendKeys(ctx, keys)` from the helper to reuse existing logic.

### 5. Helper function

Add to `window.go`:

```go
func sendStartupKeys(ctx context.Context, exec *executor, paneID string, keys []string) error {
    p := &pane{id: paneID, exec: exec}
    return p.SendKeys(ctx, keys)
}
```

### 6. `pkg/mux/tmux/tmux_test.go`

Update existing `NewWindow` call sites (6 occurrences) — no signature change needed since `NewWindow` keeps old signature.

Add tests:

1. **`TestStartupKeysSession`**: Create session with `Config.StartupKeys`, verify initial pane receives keys via `pollCapture`.
2. **`TestStartupKeysWindow`**: Create window with `NewWindowWithKeys`, verify pane gets session + window keys.
3. **`TestStartupKeysSplit`**: Split a window that has window keys, verify new panes get session + window keys.
4. **`TestStartupKeysPersistence`**: Create window with keys, reacquire via `sess.GetById`, split, verify keys still applied to new panes.
5. **`TestStartupKeysNil`**: Nil/empty startup keys produce no errors and no extra output.
6. **`TestStartupKeysOrdering`**: Session keys (`echo A`) appear before window keys (`echo B`) in capture output.

### 7. `pkg/mux/tmux/parse_test.go`

Add unit tests for new `"id\tpane_id"` parsing (valid, missing field, empty ID).

## Verification

```bash
go test ./pkg/mux/tmux/ -v -count=1
```

Manual test with demo tool:
```bash
go run ./pkg/mux/tmux/internal/cmd/splitpane/ -n 7
# After attaching, all panes should show startup command output
```
