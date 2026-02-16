# Implement tmux mux provider

## Context

The crabswarm project ("Swarm Claude Code using tmux sessions") needs a mux abstraction to manage a tmux session's windows and panes programmatically. The current `pkg/mux/session.go` has minimal scaffolding interfaces without `context.Context` or error returns. The empty `pkg/mux/tmux/` directory awaits the tmux implementation.

The scope is **single-session** — one tmux session per mux instance. Session creation is handled by the tmux implementation's constructor (`New`), not a separate `Provider` interface.

## Files to modify

- `pkg/mux/session.go` — refine interfaces (add `context.Context`, errors; no `Provider`)

## Files to create

- `pkg/mux/errors.go` — sentinel errors
- `pkg/mux/tmux/exec.go` — command executor (`os/exec` wrapper)
- `pkg/mux/tmux/parse.go` — tmux output parsing helpers
- `pkg/mux/tmux/tmux.go` — `Session` concrete type + `New` constructor (creates/attaches tmux session)
- `pkg/mux/tmux/window.go` — `window` type implementing `mux.Window`
- `pkg/mux/tmux/pane.go` — `pane` type implementing `mux.Pane`

## Step 1: Refine interfaces in `pkg/mux/session.go`

No `Provider` interface. Implementations create sessions via their own constructors (e.g. `tmux.New(cfg)`).

```go
package mux

import "context"

// Session represents a terminal multiplexer session containing windows.
type Session interface {
    Id() string
    Name(ctx context.Context) (string, error)
    NewWindow(ctx context.Context, name string) (Window, error)
    GetAt(ctx context.Context, i int) (Window, error)
    GetById(ctx context.Context, id string) (Window, error)
    List(ctx context.Context) ([]Window, error)
    Close(ctx context.Context) error
}

// Window represents a window within a session, containing one or more panes.
type Window interface {
    Id() string
    Index(ctx context.Context) (int, error)
    Name(ctx context.Context) (string, error)
    // Split splits the window into n additional panes.
    // After Split returns, the window contains its current pane count plus n panes.
    Split(ctx context.Context, n int) error
    List(ctx context.Context) ([]Pane, error)
    GetAt(ctx context.Context, i int) (Pane, error)
    GetById(ctx context.Context, id string) (Pane, error)
    Close(ctx context.Context) error
}

// Pane represents a single terminal pane.
type Pane interface {
    Id() string
    Index(ctx context.Context) (int, error)
    Name(ctx context.Context) (string, error)
    SendKeys(ctx context.Context, keys []string) error
    Capture(ctx context.Context, from int, limit int) (string, error)
    Close(ctx context.Context) error
}
```

## Step 2: Create `pkg/mux/errors.go`

```go
var (
    ErrSessionNotFound = errors.New("mux: session not found")
    ErrWindowNotFound  = errors.New("mux: window not found")
    ErrPaneNotFound    = errors.New("mux: pane not found")
    ErrSessionExists   = errors.New("mux: session already exists")
)
```

## Step 3: Create `pkg/mux/tmux/exec.go`

`Executor` struct wrapping `os/exec` — single bottleneck for all tmux CLI calls:

- Fields: `TmuxPath string`, `SocketName string`
- `Run(ctx, args...) (string, error)` — runs `tmux [args...]`, returns trimmed stdout, wraps stderr in errors
- Uses `exec.CommandContext` for context cancellation
- Supports `-L` socket name for test isolation

## Step 4: Create `pkg/mux/tmux/parse.go`

Parsing helpers for tab-delimited tmux `-F` format output:

- `parseWindows(out, sessionName, exec) []mux.Window`
- `parsePanes(out, windowTarget, exec) []mux.Pane`
- `parsePaneInfo(line) (id string, index int, err error)`

## Step 5: Create tmux types (tmux.go, window.go, pane.go)

**`tmux.go`** — exported `Session` struct implementing `mux.Session`, with `Config` pattern:

- `Config` struct: `Name string`, `TmuxPath string`, `SocketName string`
- `New(ctx, cfg) (*Session, error)` — creates a detached tmux session via `new-session -d -s <name>` and returns the `*Session`
- `Attach(ctx, cfg) (*Session, error)` — attaches to an existing session via `has-session -t <name>`
- Session methods: `Id`, `Name`, `NewWindow`, `GetAt`, `GetById`, `List`, `Close`
- Maps tmux error strings ("duplicate session", "no server running") to sentinel errors

**`window.go`** — unexported `window` struct (id, sessionName, exec):

- Target string: `session:id`
- `List` → `tmux list-panes -t <target> -F #{pane_id}\t#{pane_index}`
- `GetAt(i)` / `GetById(id)` — get pane by index or tmux pane ID

### Split algorithm

`Split(ctx, n)` adds exactly `n` panes. No stored state needed — the current round and step are computed from the pane count queried via `list-panes`.

**Round derivation:** given pane count `P`, if `2^r ≤ P < 2^(r+1)`, we are in round `r`. The step within the round is `P - 2^r`.

**Sequence by round:**

- Round 0 (H): split {0} → 2 panes
- Round 1 (V): split {0, 2} → 4 panes
- Round 2 (H): split {0, 2, 4, 6} → 8 panes
- Round 3 (V): split {0, 2, 4, ..., 14} → 16 panes

**Per-step logic:**

1. Query pane count `P` via `tmux list-panes -t <target>` (count lines)
2. Compute round and step:
   - `r = bits.Len(uint(P)) - 1` (equivalent to `floor(log2(P))`)
   - `s = P - (1 << r)` (step within the round: how many splits already done this round)
3. Determine direction: even `r` → `-h` (horizontal), odd `r` → `-v` (vertical)
4. Compute target pane position:
   - Run `tmux list-panes -t <target> -F '#{pane_id}'` to get pane IDs in visual order
   - The target is the pane at **position `2*s`** in this list
   - Why `2*s`: at the start of the round there were `2^r` panes. The `s` prior splits in this round each inserted a new pane between the originals, so original pane `s` has shifted from position `s` to position `2*s` in the visual ordering.
5. Run: `tmux split-window -h|-v -t {pane_id_at_2s}`
6. Repeat n times (re-query pane count each iteration since `P` increments)

**Example trace (round 2, H-split, starting with 4 panes at positions [A,B,C,D]):**

- Step 0 (P=4, s=0): split pane at pos 0 (A) → [A, new0, B, C, D] — 5 panes
- Step 1 (P=5, s=1): split pane at pos 2 (B) → [A, new0, B, new1, C, D] — 6 panes
- Step 2 (P=6, s=2): split pane at pos 4 (C) → 7 panes
- Step 3 (P=7, s=3): split pane at pos 6 (D) → 8 panes

**`pane.go`** — unexported `pane` struct (id, target, exec):

- `SendKeys(keys []string)` → iterates over the slice, calling `tmux send-keys -t <target> <key>` once per element (one `send-keys` invocation per string)
- `Capture(from, limit)` → `tmux capture-pane -t <target> -p -S <from> -E <from+limit>`

## Key design decisions

- **No `Provider` interface** — single-session scope; constructor creates the session directly
- **`Config` struct** instead of functional options — matches `hook/internal/server/` pattern
- **`os/exec` only** — no third-party tmux library needed
- **Socket name isolation** — `Config.SocketName` maps to `tmux -L`, enabling isolated test servers
- **`SendKeys` takes `[]string`** — iterates over the slice, invoking `tmux send-keys -t <target> <key>` once per element. One call per string gives caller full control over key names (e.g. `"Enter"`, `"C-c"`)
- **All methods take `context.Context`** — enables cancellation via `exec.CommandContext`
- **Split is stateless** — round and step computed from pane count (`2^r ≤ P < 2^(r+1)` → round `r`, step `P - 2^r`). No stored state in the window struct.

## Verification

1. `go build ./pkg/mux/...` compiles
2. `go vet ./pkg/mux/...` passes
3. Manual test: create a session, split panes, send keys, capture output (requires tmux binary)
