# Command Monitor Design Plan

**Date**: 2026-03-08
**Status**: Draft v2
**Goal**: Replace `./mux/tmux` with detached, per-command monitor processes that execute generated `config.json` definitions from per-command directories, while storing command config, runtime state, and exit history in SQLite. Inspired by Podman's conmon architecture.

---

## 1. Overview

No central daemon. Each managed command has its own **monitor process** that detaches from the invoking CLI, allocates a PTY, reads its command definition from `<command-dir>/config.json`, runs the command, and serves a gRPC API on a per-command Unix socket. SQLite stores `CommandConfig`, `CommandState`, and `CommandExitCode`; `config.json` is generated from the stored config JSON before the monitor starts.

```
                          ┌──────────────────────┐
                          │   SQLite Database    │  ← config + state + exit history
                          └──────┬───────▲───────┘
                                 │       │
         ┌───────────────────────┤       └───────────────────────┐
         │                       │                               │
   ┌─────▼──────┐          ┌─────▼──────┐                 ┌────────────┐
   │ Monitor #1 │          │ Monitor #2 │                 │    CLI     │
   │ (daemon)   │          │ (daemon)   │                 │            │
   │ │config.json│         │ │config.json│                │ ls/rm/stop │
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

1. **Monitor** — A self-detaching process (one per command). Re-execs into a new session/process group and redirects stdio away from the invoking terminal. Reads `<command-dir>/config.json`, owns a PTY, runs the configured command as its child, and serves a gRPC API on a Unix socket. Writes runtime state to SQLite on lifecycle events.
2. **CLI** — User-facing commands. Stores command config JSON in SQLite, generates per-command `config.json`, updates runtime bookkeeping, sends signals via the monitor's gRPC API for `stop`, and connects to the monitor socket for interactive I/O.
3. **Command Directory** — Per-command persistent directory containing generated `config.json` and runtime-adjacent files.
4. **SQLite Database** — Shared store for `CommandConfig`, `CommandState`, and `CommandExitCode`. Used by monitors and the CLI for `ls`, `inspect`, name lookup, stale cleanup, and exit-code history.

---

## 2. Architecture

### 2.1 Monitor Process Lifecycle

```
cmd run -- /bin/bash
  │
  ├─ (1) Insert CommandConfig row into SQLite
  ├─ (2) Create <command-dir>/ and materialize config.json from SQLite
  ├─ (3) Insert CommandState row into SQLite (state=created)
  ├─ (4) Fork monitor process with <command-dir>
  │       │
  │       └─ Monitor (child):
  │           ├─ (5) Re-exec into detached monitor process, parent exits
  │           ├─ (6) setsid() — new session leader; stdio redirected away from terminal
  │           ├─ (7) Read <command-dir>/config.json
  │           ├─ (8) Update CommandState: state=starting
  │           ├─ (9) Create gRPC socket: <runtime_dir>/<command_id>/monitor.sock
  │           ├─ (10) Allocate PTY, start configured command in PTY slave
  │           ├─ (11) Start serving gRPC API; update CommandState: state=running
  │           ├─ (12) Accept attach connections, fan-out PTY output
  │           ├─ (13) Wait for command exit
  │           ├─ (14) Update CommandState: state=exited, exit_code=N
  │           ├─ (15) Insert CommandExitCode row: (id, timestamp, exit_code)
  │           ├─ (16) If the stored config JSON requests auto-remove: delete DB rows and command-dir
  │           ├─ (17) Clean up socket, exit
  │
  └─ (4b) CLI waits for monitor to write `running` state, then returns or attaches
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

### 2.5 Command Directory

Each command gets a persistent directory:

```
<data_dir>/commands/<command_id>/
  └── config.json      # generated execution artifact
```

The CLI writes `config.json` from the command's config JSON stored in SQLite before spawning the monitor. The monitor receives `command-dir` as an argument, reads `config.json`, and executes according to that file.

---

## 3. SQLite Database

### 3.1 Location

```
$XDG_DATA_HOME/crabswarm/commands.db
```

(fallback: `~/.local/share/crabswarm/commands.db`)

SQLite stores `CommandConfig`, `CommandState`, and `CommandExitCode`. `config.json` is generated from `CommandConfig.JSON` as the execution artifact used by the monitor.

### 3.2 Schema

```sql
CREATE TABLE CommandConfig (
    ID              TEXT PRIMARY KEY,       -- UUID
    Name            TEXT UNIQUE,            -- human-readable name (nullable)
    JSON            TEXT NOT NULL           -- canonical command config JSON
);

CREATE INDEX idx_command_config_name ON CommandConfig(Name);

CREATE TABLE CommandState (
    ID              TEXT PRIMARY KEY,
    State           TEXT NOT NULL,           -- created, starting, running, exited, errored
    ExitCode        INTEGER CHECK (ExitCode BETWEEN -1 AND 255),
    JSON            TEXT NOT NULL,           -- runtime state JSON
    FOREIGN KEY (ID) REFERENCES CommandConfig(ID)
        ON DELETE CASCADE
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_command_state_state ON CommandState(State);

CREATE TABLE CommandExitCode (
    ID              TEXT NOT NULL,
    Timestamp       TEXT NOT NULL,           -- RFC3339
    ExitCode        INTEGER NOT NULL CHECK (ExitCode BETWEEN -1 AND 255),
    FOREIGN KEY (ID) REFERENCES CommandConfig(ID)
        ON DELETE CASCADE
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_command_exit_code_id_ts ON CommandExitCode(ID, Timestamp);
```

### 3.3 Stale Entry Cleanup

On `cmd ls` (and other read operations), the CLI checks liveness of entries in `starting` or `running` state:

1. Read the relevant runtime fields from `CommandState.JSON` and check if the recorded monitor PID is alive via `kill(pid, 0)`.
2. If dead: update `CommandState` to `errored`, set `error = "monitor died unexpectedly"` in `CommandState.JSON`.
3. If the command config JSON requests auto-remove on a finished entry: delete the rows and remove the command-dir.

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
| `cmd rm [flags] ID\|NAME`              | Remove a stopped command from DB and command-dir         |
| `cmd logs [flags] ID\|NAME`            | Dump scrollback buffer                                   |
| `cmd inspect ID\|NAME`                 | Show merged command definition, runtime state, and exit history (JSON) |

### 4.1 `cmd run` Flags

| Flag                    | Description                                                              |
| ----------------------- | ------------------------------------------------------------------------ |
| `-n, --name NAME`       | Human-readable unique name                                               |
| `-C DIR`                | Working directory for the command                                        |
| `-E KEY=VALUE`          | Environment variable (repeatable)                                        |
| `-l, --label KEY=VALUE` | Metadata label (repeatable)                                              |
| `--startup-keys KEYS`   | Keys to send to PTY after command starts (repeatable)                    |
| `--restart POLICY`      | `no` (default), `on-failure`, `always`                                   |
| `--rm`                  | Write auto-remove annotation into the stored config JSON                 |
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
| `-l, --label KEY=VALUE` | Filter by label (repeatable, AND logic; queried from config JSON) |
| `-a, --all`             | Show all (including exited). Default shows running only |
| `-q, --quiet`           | Print IDs only                                          |
| `--format FORMAT`       | Output format: `table` (default), `json`                |

### 4.4 `cmd stop` / `cmd rm` Flags

| Flag                    | Description                                                   |
| ----------------------- | ------------------------------------------------------------- |
| `-l, --label KEY=VALUE` | Target commands matching labels (queried from config JSON)    |
| `-s, --signal SIG`      | Signal to send (stop only, default SIGTERM)                   |
| `-f, --force`           | Force remove running commands (rm only — sends SIGKILL first) |

### 4.5 `cmd inspect`

`cmd inspect ID|NAME` reads `CommandConfig.JSON` as the canonical command definition, merges it with `CommandState`, includes recent `CommandExitCode` history, and may augment the result with live `Status` RPC data when the monitor is reachable. It may also include the generated `config.json` path for debugging.

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

`CommandConfig.JSON` is the canonical command definition. At minimum it contains the command argv, working directory, environment, startup keys, restart policy, scrollback limit, labels, annotations, and command-dir metadata needed to generate `config.json`. `--rm` is represented as an annotation inside that JSON rather than a dedicated SQL column.

`CommandState.JSON` stores mutable runtime-only fields beyond `State` and `ExitCode`, such as monitor PID, socket path, timestamps, restart count, and error details.

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
2. Re-read `config.json` (materialized from `CommandConfig.JSON`) and update `CommandState.JSON` to increment `restart_count`.
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
- **Ring buffer**: Fixed-size byte ring buffer; the configured limit is stored in `CommandConfig.JSON`. Thread-safe with mutex.

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
    store.go                        # SQLite CommandConfig/CommandState/CommandExitCode layer
    store_test.go
    config.go                       # config JSON storage and config.json materialization
    config_test.go
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
    cmd_run.go                      # `cmd run` — stores CommandConfig, materializes config.json, creates CommandState, spawns monitor
    cmd_attach.go                   # `cmd attach` — connects via gRPC Attach stream
    cmd_ls.go                       # `cmd ls` — reads DB, queries labels from CommandConfig.JSON, cleans stale entries
    cmd_stop.go                     # `cmd stop` — calls monitor's Signal RPC
    cmd_rm.go                       # `cmd rm` — deletes CommandConfig/CommandState/CommandExitCode rows + command-dir
    cmd_logs.go                     # `cmd logs` — calls monitor's Logs RPC
    cmd_inspect.go                  # `cmd inspect` — reads CommandConfig, CommandState, CommandExitCode, and optional live Status RPC
    cmd_monitor.go                  # Hidden `__monitor` subcommand (monitor entry point)
```

---

## 9. Implementation Phases

### Phase 1: Monitor Core + Run/Ls/Stop/Rm

- Monitor process: detach, allocate PTY, run command, wait for exit
- SQLite store: CommandConfig + CommandState + CommandExitCode
- config.json materialization under per-command command-dir
- Stale entry cleanup on read
- CLI: `cmd run`, `cmd ls`, `cmd stop`, `cmd rm`
- Explicit `starting` and `running` states
- Basic `-C`, `-E`, `-n`, `-l` flags backed by config JSON

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
- `--rm` annotation-driven auto-remove on exit

### Phase 5: Integration + Migration

- Label-based bulk operations (`cmd stop -l ...`, `cmd rm -l ...`)
- `cmd inspect` (JSON output from CommandConfig + CommandState + CommandExitCode)
- Remove `crabswarm server start`
- Deprecate `pkg/mux/tmux`

---

## 10. Open Questions

1. **Scrollback persistence**: In-memory only (lost on monitor exit), or persist to a log file per command? In-memory is simpler; log files allow `cmd logs` on exited commands.

2. **Compose-like definition file**: A YAML/TOML file declaring a group of commands. Deferred to a future plan.

3. **Monitor binary**: Should the monitor be a separate binary (`crabswarm-monitor`) or a hidden subcommand (`crabswarm cmd __monitor`)? Hidden subcommand is simpler (single binary), but a separate binary allows independent versioning.

4. **WAL mode for SQLite**: Use `PRAGMA journal_mode=WAL` to allow concurrent reads from CLI while monitors write. Almost certainly yes.

5. **PID file vs pidfd**: Linux 5.3+ supports `pidfd_open(2)` for race-free process liveness checks. Worth using on Linux, fallback to `kill(pid, 0)` + PID file on macOS.
