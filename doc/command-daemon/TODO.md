# Command Monitor TODO

Derived from [PLAN.md](./PLAN.md).

**Implementation Package**: `pkg/cmdman`

> **Note**: Phases here are reordered from PLAN.md for a bottom-up implementation
> sequence (storage → monitor → gRPC → CLI queries → restart → migration).
> PLAN.md groups by user-facing capability; this TODO groups by implementation dependency.

## Phase 1: Storage + Run Path

- [x] Add SQLite open/bootstrap path for the command monitor subsystem.
- [x] Configure SQLite connection setup in one place.
- [x] Enable `PRAGMA journal_mode=WAL`.
- [x] Set `PRAGMA busy_timeout` for concurrent monitor/CLI write contention.
- [x] Enable foreign keys on every SQLite connection.
- [x] Verify SQLite JSON query support in the chosen driver/build and fail fast if unavailable.
- [x] Add schema creation for `CommandConfig`.
- [x] Add schema creation for `CommandState`.
- [x] Add schema creation for `CommandExitCode`.
- [x] Add `CHECK (ExitCode BETWEEN -1 AND 255)` to `CommandState.ExitCode`.
- [x] Add `CHECK (ExitCode BETWEEN -1 AND 255)` to `CommandExitCode.ExitCode`.
- [x] Add deferred foreign key from `CommandState.ID` to `CommandConfig.ID`.
- [x] Add deferred foreign key from `CommandExitCode.ID` to `CommandConfig.ID`.
- [x] Add index for `CommandConfig.Name`.
- [x] Add index for `CommandState.State`.
- [x] Add index for `CommandExitCode(ID, Timestamp)`.
- [x] Define the canonical `CommandConfig.JSON` shape (argv, working directory, environment, startup keys, restart policy, scrollback limit, labels, annotations, command-dir metadata).
- [x] Define the `CommandState.JSON` shape (monitor PID, socket path, timestamps, restart count, error details).
- [x] Implement config JSON storage helpers.
- [x] Implement `config.json` materialization from `CommandConfig.JSON`.
- [x] Create per-command command-dir layout under the data directory.
- [x] Add hidden `cmd __monitor` entrypoint.
- [x] Implement monitor detachment via single fork/re-exec.
- [x] Add `cmd run` command skeleton.
- [x] Build `cmd run` config JSON from flags and argv.
- [x] Support `--scrollback-bytes` flag on `cmd run`.
- [x] Insert `CommandConfig` row before spawning the monitor.
- [x] Materialize `<command-dir>/config.json` from `CommandConfig.JSON`.
- [x] Insert initial `CommandState` row with `State=created`.
- [x] Spawn the hidden monitor entrypoint with `command-dir`.
- [x] Implement CLI-to-monitor handshake: define mechanism for waiting on `State=running` (e.g. poll SQLite, pipe, inotify).
- [x] Handle monitor spawn failure: detect early monitor death before `running` and clean up DB/command-dir.
- [x] Support `--attach` after `State=running`.

## Phase 2: Monitor Core

- [x] Redirect stdio away from the invoking terminal.
- [x] Read `<command-dir>/config.json` on monitor startup.
- [x] Allocate the PTY for the configured command.
- [x] Start the configured command in the PTY slave.
- [x] Update `CommandState` to `starting` during monitor startup.
- [x] Write monitor PID into `CommandState.JSON`.
- [x] Create `<runtime_dir>/cmd/<command_id>/` directory with user-only permissions (`0700`).
- [x] Write monitor PID file at `<runtime_dir>/cmd/<command_id>/pid`.
- [x] Start the gRPC server on `<runtime_dir>/cmd/<command_id>/monitor.sock`.
- [x] Write socket path into `CommandState.JSON`.
- [x] Write startup timestamp into `CommandState.JSON`.
- [x] Update `CommandState` to `running` only after the monitor is ready.
- [x] Handle monitor graceful shutdown: on SIGTERM, forward signal to child, wait for exit, then clean up.
- [x] Update `CommandState.ExitCode` on process exit.
- [x] Update `CommandState.State=exited` on process exit.
- [x] Write finished timestamp into `CommandState.JSON` on process exit.
- [x] Write runtime error details into `CommandState.JSON` on failures.
- [x] Insert `CommandExitCode` row on each process exit.
- [x] Clean up Unix socket and PID file on monitor exit.
- [x] Implement annotation-driven auto-remove using `CommandConfig.JSON`.
- [x] Resolve command-dir from `CommandConfig.JSON` during cleanup/removal paths.
- [x] Delete `CommandConfig`, `CommandState`, `CommandExitCode`, and command-dir on auto-remove.

## Phase 3: Attach, Logs, and Status

- [x] Define and generate `cmdman.proto` with Attach, Logs, Signal, and Status RPCs.
- [x] Implement monitor gRPC server.
- [x] Implement attach fan-out for multiple clients.
- [x] Implement attach stdin multiplexing.
- [x] Implement resize handling through `AttachInput.resize`.
- [x] Implement in-memory byte ring buffer for scrollback.
- [x] Read scrollback limit from `CommandConfig.JSON`.
- [x] Implement attach replay from the ring buffer.
- [x] Implement `Logs(follow=true)` behavior.
- [x] Add `cmd attach`.
- [x] Add `cmd logs`.
- [x] Add `cmd stop` (signal through monitor's `Signal` RPC).
- [ ] Add client-side detach key handling.
- [x] Add `--no-stdin` support.
- [x] Add `--sig-proxy` support.

## Phase 4: Query and Cleanup

- [x] Add `cmd ls`.
- [x] Resolve IDs and names from `CommandConfig`.
- [x] Query labels from `CommandConfig.JSON` with SQLite JSON functions.
- [x] Support `cmd ls -l KEY=VALUE`.
- [x] Support `cmd stop -l KEY=VALUE` (extends `cmd stop` from Phase 3 with label-based bulk targeting).
- [x] Support `cmd rm -l KEY=VALUE`.
- [x] Implement stale-entry liveness checks using monitor PID from `CommandState.JSON`.
- [x] On stale monitor detection, mark the command failed in `CommandState`.
- [x] Add `cmd rm`.
- [x] Resolve command-dir from `CommandConfig.JSON` for explicit `cmd rm`.
- [x] Make `cmd rm` delete DB rows and command-dir.
- [x] Add forced remove path for running commands.
- [x] Add `cmd inspect`.
- [x] Make `cmd inspect` return merged `CommandConfig`, `CommandState`, and `CommandExitCode`.
- [x] Include generated `config.json` path in inspect output when useful.

## Phase 5: Restart and Lifecycle Policy

- [x] Implement restart policy parsing in `CommandConfig.JSON`.
- [x] Implement `no` restart behavior.
- [x] Implement `on-failure` restart behavior.
- [x] Implement `always` restart behavior.
- [x] Re-read `config.json` on restart.
- [x] Increment restart count in `CommandState.JSON` on restart.
- [x] Preserve attach sessions across restarts.
- [x] Record exit history for each restart cycle in `CommandExitCode`.

## Phase 6: Migration and Cleanup

- [ ] Remove `crabswarm server start`.
- [ ] Deprecate `pkg/mux/tmux`.
- [x] Port interpolation logic from `pkg/mux/tmux/interpolate.go`.
- [x] Support `#{COMMAND_ID}`, `#{COMMAND_NAME}`, and `#{INJECT_META}` interpolation.
- [x] Apply interpolation to startup keys at send time.

## Verification

- [x] Add schema tests for `CommandConfig`, `CommandState`, and `CommandExitCode`.
- [x] Add tests for deferred foreign key behavior.
- [x] Add tests for exit code range checks.
- [x] Add tests for config JSON materialization to `config.json`.
- [x] Add tests for `cmd run` -> monitor startup -> `State=running`.
- [ ] Add tests for monitor spawn failure detection and cleanup.
- [x] Add tests for monitor graceful shutdown.
- [ ] Add tests for attach fan-out with multiple clients.
- [x] Add tests for label filtering via SQLite JSON queries.
- [x] Add tests for stale-entry cleanup.
- [x] Add tests for auto-remove behavior.
- [x] Add tests for restart policies.
- [x] Add tests for `cmd inspect` merged output.
- [x] Add end-to-end lifecycle test: `cmd run` → `cmd attach` → `cmd stop` → `cmd rm`.
