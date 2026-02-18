# Replace `InjectMetadata` flag with placeholder interpolation in keys

## Context

The `InjectMetadata` bool flag on `Config` is a rigid, all-or-nothing mechanism for injecting `CRAB_SESSION_ID`/`CRAB_WINDOW_ID`/`CRAB_PANE_ID` env vars. Replacing it with placeholder interpolation (`#{SESSION_ID}`, `#{WINDOW_ID}`, `#{PANE_ID}`, `#{INJECT_META}`) in keys gives callers full control over *what* gets injected, *where* in the key sequence, and *how* (e.g. custom env var names). Escaping via `##{...}` produces the literal `#{...}`.

## Interpolation semantics

| Placeholder | Expands to | Example |
|---|---|---|
| `#{SESSION_ID}` | tmux session ID (e.g. `$0`) | `echo #{SESSION_ID}` → `echo $0` |
| `#{WINDOW_ID}` | tmux window ID (e.g. `@1`) | `echo #{WINDOW_ID}` → `echo @1` |
| `#{PANE_ID}` | tmux pane ID (e.g. `%3`) | `echo #{PANE_ID}` → `echo %3` |
| `#{INJECT_META}` | `export CRAB_SESSION_ID='<sid>' CRAB_WINDOW_ID='<wid>' CRAB_PANE_ID='<pid>'` | Caller adds `"Enter"` as next key |
| `##{...}` | Literal `#{...}` (escape) | `##{PANE_ID}` → `#{PANE_ID}` |

All replacements happen per-key-string via simple string substitution. `#{INJECT_META}` produces only the export text (no Enter); callers add `"Enter"` as a separate key entry, consistent with how all other commands work.

Interpolation occurs in `pane.SendKeys` — every call to SendKeys interpolates. `sendStartupKeys` delegates to `pane.SendKeys` so it gets interpolation for free.

## Files to modify

### 1. `pkg/mux/tmux/interpolate.go` — new file, interpolation logic

Create `interpolateKeys(keys []string, sessionID, windowID, paneID string) []string`:
- For each key string, run `interpolateKey(key, sessionID, windowID, paneID) string`
- `interpolateKey` algorithm:
  1. Replace `##{` with sentinel `\x00{`
  2. Replace `#{INJECT_META}` with formatted export command
  3. Replace `#{SESSION_ID}` with sessionID
  4. Replace `#{WINDOW_ID}` with windowID
  5. Replace `#{PANE_ID}` with paneID
  6. Replace sentinel `\x00{` back to `#{`
- Return new slice (never mutate input)

Add unit tests in `pkg/mux/tmux/interpolate_test.go`:
- Basic replacement for each placeholder
- Multiple placeholders in one string
- Escaping `##{...}` → `#{...}`
- No placeholders → unchanged
- Empty/nil input

### 2. `pkg/mux/tmux/pane.go` — add IDs to pane, interpolate in SendKeys

Add `sessionID` and `windowID` fields to `pane` struct.

Update `SendKeys` to interpolate before sending.

### 3. `pkg/mux/tmux/parse.go` — pass IDs through to pane construction

Update `parsePanes` signature: replace `windowTarget string` with `sessionID, windowID string`.
Set `sessionID` and `windowID` on each constructed pane.

### 4. `pkg/mux/tmux/window.go` — remove injectMetadata, update sendStartupKeys

- Remove `injectMetadata` field from `window` struct
- Delete `paneMetadata` struct and `metadataKeys()` method
- Update `sendStartupKeys`: remove `meta *paneMetadata` param, add `sessionID, windowID string`. Assembles keys and delegates to `pane.SendKeys` (which handles interpolation).
- Update `window.List` call to `parsePanes`: pass `w.sessionID, w.id`
- Update `Split`: remove `injectMetadata` conditional and `meta` variable, pass IDs to `sendStartupKeys`

### 5. `pkg/mux/tmux/tmux.go` — remove InjectMetadata from Config and Session

- Remove `InjectMetadata bool` from `Config`. Update `StartupKeys` doc to mention available placeholders.
- Remove `injectMetadata` field from `Session` struct
- `New()`: remove `injectMetadata` from Session construction, remove `initialMeta`, update `sendStartupKeys` call to pass IDs
- `Attach()`: remove `injectMetadata` from Session construction
- `NewWindow()`: remove `injectMetadata` from window construction, remove `meta`, update `sendStartupKeys` call
- `List()`: remove `injectMetadata` from `parseWindows` call

### 6. `pkg/mux/tmux/parse.go` — remove injectMetadata from parseWindows

Remove `injectMetadata bool` param from `parseWindows`.
Remove `injectMetadata` field assignment in window construction.

### 7. `pkg/mux/tmux/tmux_test.go` — update tests

**Remove:** `newTestSessionWithMetadata` helper, `TestInjectMetadata`, `TestInjectMetadataSplit`, `TestInjectMetadataAfterStartupKeys`, `TestInjectMetadataDisabled`

**Add new interpolation integration tests:**
- `TestInterpolateSessionID`: startup keys with `#{SESSION_ID}`, verify pane captures the actual session ID
- `TestInterpolatePaneID`: startup keys with `echo #{PANE_ID}`, create window + split, verify each pane sees its own pane ID
- `TestInterpolateInjectMeta`: startup keys `["#{INJECT_META}", "Enter"]`, verify CRAB_PANE_ID is set
- `TestInterpolateInjectMetaSplit`: same but via Split
- `TestInterpolateEscape`: startup keys with `##{PANE_ID}`, verify literal `#{PANE_ID}` appears
- `TestInterpolateSendKeys`: plain pane.SendKeys with `#{PANE_ID}`, verify it expands

### 8. `pkg/mux/tmux/internal/cmd/splitpane/main.go` — remove inject-metadata flag

- Remove `-inject-metadata` flag, its usage, and `InjectMetadata` from Config construction
- Update package doc comment

## Verification

```bash
go build ./pkg/mux/...
go test ./pkg/mux/tmux/ -v -count=1
```
