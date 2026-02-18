# Add `InjectMetadata` option to tmux panes

## Context

Users need panes to know their own identity (pane ID, window ID, session ID) so that scripts running in the pane can report back which pane they are. This adds an `InjectMetadata` boolean to `Config` that, when enabled, sends `export CRAB_SESSION_ID='$0' CRAB_WINDOW_ID='@1' CRAB_PANE_ID='%2'` + `Enter` to each new pane **before** startup keys, so startup keys can reference these env vars.

## Design decisions

### Inject all three IDs

The marginal cost of sending two extra exports is negligible (`CRAB_SESSION_ID`, `CRAB_WINDOW_ID`, `CRAB_PANE_ID`). Having all three available makes the feature much more useful.

### Single-quote values in export command

tmux IDs like `$0`, `@1`, `%2` contain shell-special characters. Without quoting, `$0` would expand to the shell name. The export command uses single quotes: `export CRAB_SESSION_ID='$0' CRAB_WINDOW_ID='@1' CRAB_PANE_ID='%2'`.

### Combine into one export command

All three env vars are set in a single `export` command + one `Enter`, minimizing send-keys invocations and visual clutter.

### Metadata before startup keys

Metadata export is sent before session/window startup keys so that startup keys can reference `$CRAB_PANE_ID` etc.

### No configurable prefix

The `CRAB_` prefix is hardcoded. Adding a prefix option adds complexity with little benefit.

## Files to modify

### 1. `pkg/mux/tmux/window.go`

- Add `paneMetadata` struct with `sessionID`, `windowID`, `paneID` fields
- Add `metadataKeys()` method returning `[]string{"export CRAB_SESSION_ID='...' CRAB_WINDOW_ID='...' CRAB_PANE_ID='...'", "Enter"}`
- Add `sessionID string` and `injectMetadata bool` fields to `window` struct
- Update `sendStartupKeys()` signature: add `meta *paneMetadata` parameter; prepend `meta.metadataKeys()` before session+window keys
- Update `Split()`: build `paneMetadata` when `w.injectMetadata` is true, pass to `sendStartupKeys`

### 2. `pkg/mux/tmux/tmux.go`

- Add `InjectMetadata bool` to `Config`
- Add `injectMetadata bool` to `Session` struct
- **`New()`**: change format to `"#{session_id}\t#{window_id}\t#{pane_id}"` (3 parts), store `injectMetadata`, build metadata, pass to `sendStartupKeys`
- **`Attach()`**: store `cfg.InjectMetadata`
- **`NewWindowWithKeys()`**: set `sessionID` and `injectMetadata` on `window`, build metadata, pass to `sendStartupKeys`
- **`List()`**: pass `s.id` and `s.injectMetadata` to `parseWindows`

### 3. `pkg/mux/tmux/parse.go`

- Update `parseWindows()` signature: add `sessionID string` and `injectMetadata bool` params
- Set `sessionID` and `injectMetadata` on each constructed `window`

### 4. `pkg/mux/tmux/parse_test.go`

- Update 3 `parseWindows()` call sites to pass `"$0"` for sessionID and `false` for injectMetadata

### 5. `pkg/mux/tmux/tmux_test.go`

Add tests:
- **`TestInjectMetadata`**: session with `InjectMetadata: true`, verify initial pane has `CRAB_PANE_ID` set
- **`TestInjectMetadataSplit`**: split window, verify new pane's `CRAB_PANE_ID` differs from original
- **`TestInjectMetadataBeforeStartupKeys`**: session with metadata + StartupKeys using `$CRAB_PANE_ID`, verify the var is available
- **`TestInjectMetadataDisabled`**: session without metadata, verify env var not set

### 6. `pkg/mux/tmux/internal/cmd/splitpane/main.go`

- Add `-inject-metadata` bool flag, wire to `cfg.InjectMetadata`

## Verification

```bash
go test ./pkg/mux/tmux/ -v -count=1
go run ./pkg/mux/tmux/internal/cmd/splitpane/ -n 3 -inject-metadata -no-attach
# Then: tmux -L splitpane-demo attach, check env vars with: echo $CRAB_PANE_ID
```
