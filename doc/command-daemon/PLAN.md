# Command Monitor Design Plan

**Date**: 2026-03-08
**Status**: Draft v2
**Goal**: Replace `./mux/tmux` with detached, per-command monitor processes that coordinate via a shared SQLite database. Inspired by Podman's conmon architecture.

---

## 1. Overview

No central daemon. Each managed command has its own **monitor process** that detaches from the invoking CLI, allocates a PTY, runs the command, and serves a gRPC API on a per-command Unix socket. All state is shared via a **SQLite database**.

```
                          ┌──────────────────────┐
                          │   SQLite Database    │  ← shared state
                          └──────┬───────▲───────┘
                                 │       │
         ┌───────────────────────┤       └───────────────────────┐
         │                       │                               │
   ┌─────▼──────┐          ┌─────▼──────┐                 ┌────────────┐
   │ Monitor #1 │          │ Monitor #2 │                 │    CLI     │
   │ (daemon)   │          │ (daemon)   │                 │            │
   │ ┌────────┐ │          │ ┌────────┐ │                 │ ls/rm/stop │
   │ │ PTY    │ │          │ │ PTY    │ │                 │ run/attach │
   │ │ cmd #1 │ │          │ │ cmd #2 │ │                 └──────┬─────┘
   │ └────────┘ │          │ └────────┘ │                        │
   │ monitor.sock│         │ monitor.sock│              attach via gRPC UDS
   └────────────┘          └────────────┘                        │
         ▲                       ▲                               │
         │       gRPC over UDS   │                               │
         └───────────────────────────────────────────────────────┘

```

### Components

1. **Monitor** — A self-detaching process (one per command). Re-execs into a new session/process group and redirects stdio away from the invoking terminal. Owns a PTY, runs the command as its child, and serves a gRPC API on a Unix socket. Writes state to SQLite on lifecycle events.
2. **CLI** — User-facing commands. Reads SQLite for queries (`ls`, `inspect`). Sends signals via the monitor's gRPC API for `stop`. Connects to the monitor socket for interactive I/O.
3. **SQLite Database** — Single source of truth for command state, metadata, labels. Shared across all monitors and the CLI.

---

## 2. Architecture

### 2.1 Monitor Process Lifecycle

```
cmd run -- /bin/bash
  │
  ├─ (1) Insert command record into SQLite (state=created)
  ├─ (2) Fork monitor process
  │       │
  │       └─ Monitor (child):
  │           ├─ (3) Re-exec into detached monitor process, parent exits
  │           ├─ (4) setsid() — new session leader; stdio redirected away from terminal
  │           ├─ (5) Update DB: state=starting, pid=<self>
  │           ├─ (6) Create gRPC socket: <runtime_dir>/<command_id>/monitor.sock
  │           ├─ (7) Allocate PTY, start command in PTY slave
  │           ├─ (8) Start serving gRPC API; update DB: state=running
  │           ├─ (9) Accept attach connections, fan-out PTY output
  │           ├─ (10) Wait for command exit
  │           ├─ (11) Update DB: state=exited, exit_code=N, finished_at=now
  │           ├─ (12) If --rm: delete DB record
  │           ├─ (13) Clean up socket, exit
  │
  └─ (2b) CLI waits for monitor to write `running` state, then returns or attaches
```

**Detachment**: The monitor uses `exec.Command` to launch itself with a special `__monitor` subcommand (hidden). A single fork/re-exec is sufficient: start the monitor in a new session/process group and redirect stdin/stdout/stderr away from the invoking terminal (for example to `/dev/null` or log files). Classic double-fork daemonization is not required.

### 2.2 Monitor gRPC Server

Each monitor runs a gRPC server on a per-command Unix domain socket:

```
$XDG_RUNTIME_DIR/crabswarm/cmd/<command_id>/monitor.sock
```

The socket path is recorded in the SQLite database for CLI discovery. This is the single transport for attach, logs, signal, and status operations.

```
<runtime_dir>/cmd/<command_id>/
  ├── monitor.sock     # gRPC server (attach, signal, resize, logs)
  └── pid              # monitor PID file
```

**Service definition**:

```protobuf
syntax = "proto3";
package crabswarm.cmdmon.v1;

service CommandMonitor {
  // Bidirectional streaming — PTY I/O + control
  rpc Attach(stream AttachInput) returns (stream AttachOutput);

  // Read scrollback buffer (non-interactive)
  rpc Logs(LogsRequest) returns (stream LogsOutput);

  // Send signal to the command process
  rpc Signal(SignalRequest) returns (SignalResponse);

  // Query monitor status
  rpc Status(StatusRequest) returns (StatusResponse);
}

message AttachInput {
  oneof input {
    bytes stdin = 1;            // raw bytes to write to PTY
    ResizeEvent resize = 2;     // terminal resize (SIGWINCH)
  }
}

message AttachOutput {
  bytes stdout = 1;             // raw bytes from PTY
}

message ResizeEvent {
  uint32 rows = 1;
  uint32 cols = 2;
}

message LogsRequest {
  bool follow = 1;              // stream live output after scrollback
}

message LogsOutput {
  bytes data = 1;
}

message SignalRequest {
  int32 signal = 1;             // e.g. 15 for SIGTERM
}

message SignalResponse {}

message StatusRequest {}

message StatusResponse {
  string state = 1;             // starting, running, exited, errored
  int32 exit_code = 2;
  int32 pid = 3;
}
```

### 2.3 Multiple Attach Clients

Following Docker/Podman behavior:

- Multiple clients can attach simultaneously — each opens its own `Attach` bidi stream.
- All clients receive the same PTY output (fan-out from monitor).
- All clients can write stdin (multiplexed to PTY, serialized via mutex).
- Terminal resize: last-writer-wins.
- Detach: client closes the stream (or uses configurable detach key sequence handled client-side).
- Signal proxy: CLI intercepts signals (e.g., SIGINT) and calls `Signal` RPC instead of signaling processes directly.

### 2.4 Scrollback Buffer

Each monitor maintains an in-memory byte ring buffer (default 1 MiB). When a new client attaches:

1. Send buffered scrollback content first via the `Attach` stream.
2. Switch to live streaming.

Transition is atomic — no output is lost or duplicated.

`Logs` RPC provides non-interactive access (no stdin). With `follow=true`, it behaves like `tail -f`.

---

## 3. SQLite Database

### 3.1 Location

```
$XDG_DATA_HOME/crabswarm/commands.db
```

(fallback: `~/.local/share/crabswarm/commands.db`)

### 3.2 Schema

```sql
CREATE TABLE commands (
    id              TEXT PRIMARY KEY,       -- UUID
    name            TEXT UNIQUE,            -- human-readable name (nullable)
    command         TEXT NOT NULL,           -- JSON array of command + args
    working_dir     TEXT,                    -- -C flag
    env             TEXT,                    -- JSON array of KEY=VALUE pairs
    startup_keys    TEXT,                    -- JSON array
    restart_policy  TEXT DEFAULT 'no',       -- no | on-failure | always
    auto_remove     INTEGER DEFAULT 0,      -- --rm flag
    scrollback_bytes INTEGER DEFAULT 1048576,

    -- Runtime state
    state           TEXT NOT NULL,           -- created, starting, running, exited, errored
    exit_code       INTEGER,
    restart_count   INTEGER DEFAULT 0,
    monitor_pid     INTEGER,                -- PID of monitor process
    socket_dir      TEXT,                    -- path to monitor.sock directory

    -- Timestamps
    created_at      TEXT NOT NULL,           -- RFC3339
    started_at      TEXT,
    finished_at     TEXT,
    error           TEXT                     -- error message if errored
);

CREATE INDEX idx_commands_state ON commands(state);
CREATE INDEX idx_commands_name ON commands(name);

CREATE TABLE command_labels (
    command_id  TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    PRIMARY KEY (command_id, key),
    FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE CASCADE
);

CREATE INDEX idx_labels_kv ON command_labels(key, value);
```

### 3.3 Stale Entry Cleanup

On `cmd ls` (and other read operations), the CLI checks liveness of entries in `starting` or `running` state:

1. Check if `monitor_pid` is alive via `kill(pid, 0)`.
2. If dead: update state to `errored`, set `error = "monitor died unexpectedly"`, set `finished_at = now`.
3. If `auto_remove = 1` on a finished entry: delete the row.

This ensures the DB self-heals after reboots or crashes without a daemon.

---

## 4. CLI Commands

Subcommands under `crabswarm cmd`:

| Command                                | Description                                              |
| -------------------------------------- | -------------------------------------------------------- |
| `cmd run [flags] -- COMMAND [ARGS...]` | Spawn a monitor, start command                           |
| `cmd attach [flags] ID\|NAME`          | Attach to a running command's PTY                        |
| `cmd ls [flags]`                       | List commands (with stale cleanup)                       |
| `cmd stop [flags] ID\|NAME`            | Send signal via monitor's `Signal` RPC (default SIGTERM) |
| `cmd rm [flags] ID\|NAME`              | Remove a stopped command from DB                         |
| `cmd logs [flags] ID\|NAME`            | Dump scrollback buffer                                   |
| `cmd inspect ID\|NAME`                 | Show detailed command info (JSON)                        |

### 4.1 `cmd run` Flags

| Flag                    | Description                                                              |
| ----------------------- | ------------------------------------------------------------------------ |
| `-n, --name NAME`       | Human-readable unique name                                               |
| `-C DIR`                | Working directory for the command                                        |
| `-E KEY=VALUE`          | Environment variable (repeatable)                                        |
| `-l, --label KEY=VALUE` | Metadata label (repeatable)                                              |
| `--startup-keys KEYS`   | Keys to send to PTY after command starts (repeatable)                    |
| `--restart POLICY`      | `no` (default), `on-failure`, `always`                                   |
| `--rm`                  | Auto-remove DB entry on command exit                                     |
| `--scrollback-bytes N`  | Scrollback buffer size in bytes (default 1048576)                        |
| `--attach`              | Attach after the monitor reaches `running`                               |

### 4.2 `cmd attach` Flags

| Flag                | Description                                      |
| ------------------- | ------------------------------------------------ |
| `--no-stdin`        | Output-only mode                                 |
| `--sig-proxy`       | Forward signals to command (default true)        |
| `--detach-keys SEQ` | Key sequence to detach (default `ctrl-p,ctrl-q`) |

### 4.3 `cmd ls` Flags

| Flag                    | Description                                             |
| ----------------------- | ------------------------------------------------------- |
| `-l, --label KEY=VALUE` | Filter by label (repeatable, AND logic)                 |
| `-a, --all`             | Show all (including exited). Default shows running only |
| `-q, --quiet`           | Print IDs only                                          |
| `--format FORMAT`       | Output format: `table` (default), `json`                |

### 4.4 `cmd stop` / `cmd rm` Flags

| Flag                    | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `-l, --label KEY=VALUE` | Target commands matching labels                               |
| `-s, --signal SIG`      | Signal to send (stop only, default SIGTERM)                   |
| `-f, --force`           | Force remove running commands (rm only — sends SIGKILL first) |

---

## 5. Interpolation and Startup Keys

Adapted from `pkg/mux/tmux/interpolate.go`:

| Placeholder       | Value                                                  |
| ----------------- | ------------------------------------------------------ |
| `#{COMMAND_ID}`   | Command UUID                                           |
| `#{COMMAND_NAME}` | Human-readable name                                    |
| `#{INJECT_META}`  | `export CRAB_COMMAND_ID='...' CRAB_COMMAND_NAME='...'` |

Escaping: `##{...}` produces literal `#{...}`.

Startup keys are sent to the PTY stdin after the command process starts and the monitor has confirmed it is running. Interpolation is applied at send time.

---

## 6. Restart Policies

The monitor handles restarts internally (no daemon needed):

| Policy       | Behavior                                                                           |
| ------------ | ---------------------------------------------------------------------------------- |
| `no`         | Command exits → monitor records state and exits                                    |
| `on-failure` | Non-zero exit → monitor restarts command. Zero exit → stop                         |
| `always`     | Command exits for any reason → monitor restarts. Only stops on explicit `cmd stop` |

On restart:

1. Release old PTY.
2. Update DB: increment `restart_count`.
3. Allocate new PTY, start command again.
4. Attached clients remain connected — they see the new output seamlessly.

---

## 7. PTY Management

### 7.1 Library

`github.com/creack/pty` — supports Linux and macOS:

- `pty.Start(cmd)` — allocate PTY and start command
- `pty.Setsize(f, winsize)` — resize PTY
- Proper fd cleanup on close

### 7.2 I/O Architecture within Monitor

```
                ┌─────────────┐
                │ PTY master  │
                └──┬──────┬───┘
          read     │      │  write
                   ▼      ▲
        ┌──────────────┐  │
        │ Read routine │  │
        │              │  │  ┌──────────────┐
        │ → scrollback │  │  │ Write routine │
        │ → fan-out ───┼──┼──│ ← mux stdin  │
        └──────────────┘  │  └──────────────┘
               │          │         ▲
          ┌────┼──────────┼─────────┼────┐
          │    ▼          │         │    │
          │ stream #1  stream #2   ... │
          │ (gRPC Attach bidi streams) │
          └─────────────────────────────┘
```

- **Read goroutine**: Reads PTY master → appends to ring buffer → sends `AttachOutput` to all active Attach streams.
- **Write goroutine**: Receives `AttachInput.stdin` from all Attach streams (each handled in its own goroutine) → serializes via channel → writes to PTY master.
- **Resize handling**: `AttachInput.resize` messages call `pty.Setsize()` directly (last-writer-wins).
- **Ring buffer**: Fixed-size byte ring buffer (`scrollback_bytes`). Thread-safe with mutex.

### 7.3 Terminal Resize

1. Attach client detects SIGWINCH.
2. Client sends `AttachInput{resize: ResizeEvent{rows: R, cols: C}}` on its Attach stream.
3. Monitor calls `pty.Setsize()`.
4. Last-writer-wins when multiple clients have different sizes.

---

## 8. Package Layout

```
pkg/
  cmdmon/                           # command monitor core
    monitor.go                      # Monitor process entry point, lifecycle
    monitor_test.go
    daemonize.go                    # Monitor detachment logic
    daemonize_test.go
    server.go                       # gRPC server (Attach, Logs, Signal, Status)
    server_test.go
    fanout.go                       # Fan-out + mux for PTY I/O across streams
    fanout_test.go
    store.go                        # SQLite persistence layer
    store_test.go
    interpolate.go                  # Placeholder interpolation (ported)
    interpolate_test.go
    scrollback.go                   # Ring buffer for output capture
    scrollback_test.go
  cmdmon/api/v1/                    # Generated protobuf/gRPC code
    cmdmon.proto
    cmdmon.pb.go
    cmdmon_grpc.pb.go
cmd/
  crabswarm/commands/
    cmd.go                          # `crabswarm cmd` parent command
    cmd_run.go                      # `cmd run` — creates DB entry, spawns monitor
    cmd_attach.go                   # `cmd attach` — connects via gRPC Attach stream
    cmd_ls.go                       # `cmd ls` — reads DB, cleans stale entries
    cmd_stop.go                     # `cmd stop` — calls monitor's Signal RPC
    cmd_rm.go                       # `cmd rm` — deletes DB entry + socket dir
    cmd_logs.go                     # `cmd logs` — calls monitor's Logs RPC
    cmd_inspect.go                  # `cmd inspect` — reads DB + calls Status RPC
    cmd_monitor.go                  # Hidden `__monitor` subcommand (monitor entry point)
```

---

## 9. Implementation Phases

### Phase 1: Monitor Core + Run/Ls/Stop/Rm

- Monitor process: detach, allocate PTY, run command, wait for exit
- SQLite store: CRUD for commands table
- Stale entry cleanup on read
- CLI: `cmd run`, `cmd ls`, `cmd stop`, `cmd rm`
- Explicit `starting` and `running` states
- Basic `-C`, `-E`, `-n`, `-l` flags

### Phase 2: Attach + Interactive I/O

- gRPC monitor socket: Attach, Logs, Signal, Status
- Scrollback ring buffer with drain-on-attach
- Multiple simultaneous clients
- CLI: `cmd attach`, `cmd logs`
- Detach key handling, `--no-stdin`, `--sig-proxy`

### Phase 3: Startup Keys + Interpolation

- Port interpolation from `pkg/mux/tmux/interpolate.go`
- `#{COMMAND_ID}`, `#{COMMAND_NAME}`, `#{INJECT_META}`
- `--startup-keys` flag

### Phase 4: Restart Policies

- `--restart no|on-failure|always`
- Monitor re-exec loop with new PTY allocation
- Attached clients survive restart seamlessly
- `--rm` auto-remove on exit

### Phase 5: Integration + Migration

- Label-based bulk operations (`cmd stop -l ...`, `cmd rm -l ...`)
- `cmd inspect` (JSON output)
- Remove `crabswarm server start`
- Deprecate `pkg/mux/tmux`

---

## 10. Open Questions

1. **Scrollback persistence**: In-memory only (lost on monitor exit), or persist to a log file per command? In-memory is simpler; log files allow `cmd logs` on exited commands.

2. **Compose-like definition file**: A YAML/TOML file declaring a group of commands. Deferred to a future plan.

3. **Monitor binary**: Should the monitor be a separate binary (`crabswarm-monitor`) or a hidden subcommand (`crabswarm cmd __monitor`)? Hidden subcommand is simpler (single binary), but a separate binary allows independent versioning.

4. **WAL mode for SQLite**: Use `PRAGMA journal_mode=WAL` to allow concurrent reads from CLI while monitors write. Almost certainly yes.

5. **PID file vs pidfd**: Linux 5.3+ supports `pidfd_open(2)` for race-free process liveness checks. Worth using on Linux, fallback to `kill(pid, 0)` + PID file on macOS.
