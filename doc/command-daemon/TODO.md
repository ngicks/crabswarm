# Command Monitor TODO

Derived from [PLAN.md](/home/watage/gitrepo/github.com/ngicks/crabswarm/doc/command-daemon/PLAN.md).

> **Note**: Phases here are reordered from PLAN.md for a bottom-up implementation
> sequence (storage → monitor → gRPC → CLI queries → restart → migration).
> PLAN.md groups by user-facing capability; this TODO groups by implementation dependency.

## Phase 1: Storage + Run Path

- [ ] Add SQLite open/bootstrap path for the command monitor subsystem.
- [ ] Configure SQLite connection setup in one place.
- [ ] Enable `PRAGMA journal_mode=WAL`.
- [ ] Set `PRAGMA busy_timeout` for concurrent monitor/CLI write contention.
- [ ] Enable foreign keys on every SQLite connection.
- [ ] Verify SQLite JSON query support in the chosen driver/build and fail fast if unavailable.
- [ ] Add schema creation for `CommandConfig`.
- [ ] Add schema creation for `CommandState`.
- [ ] Add schema creation for `CommandExitCode`.
- [ ] Add `CHECK (ExitCode BETWEEN -1 AND 255)` to `CommandState.ExitCode`.
- [ ] Add `CHECK (ExitCode BETWEEN -1 AND 255)` to `CommandExitCode.ExitCode`.
- [ ] Add deferred foreign key from `CommandState.ID` to `CommandConfig.ID`.
- [ ] Add deferred foreign key from `CommandExitCode.ID` to `CommandConfig.ID`.
- [ ] Add index for `CommandConfig.Name`.
- [ ] Add index for `CommandState.State`.
- [ ] Add index for `CommandExitCode(ID, Timestamp)`.
- [ ] Define the canonical `CommandConfig.JSON` shape (argv, working directory, environment, startup keys, restart policy, scrollback limit, labels, annotations, command-dir metadata).
- [ ] Define the `CommandState.JSON` shape (monitor PID, socket path, timestamps, restart count, error details).
- [ ] Implement config JSON storage helpers.
- [ ] Implement `config.json` materialization from `CommandConfig.JSON`.
- [ ] Create per-command command-dir layout under the data directory.
- [ ] Add hidden `cmd __monitor` entrypoint.
- [ ] Implement monitor detachment via single fork/re-exec.
- [ ] Add `cmd run` command skeleton.
- [ ] Build `cmd run` config JSON from flags and argv.
- [ ] Support `--scrollback-bytes` flag on `cmd run`.
- [ ] Insert `CommandConfig` row before spawning the monitor.
- [ ] Materialize `<command-dir>/config.json` from `CommandConfig.JSON`.
- [ ] Insert initial `CommandState` row with `State=created`.
- [ ] Spawn the hidden monitor entrypoint with `command-dir`.
- [ ] Implement CLI-to-monitor handshake: define mechanism for waiting on `State=running` (e.g. poll SQLite, pipe, inotify).
- [ ] Handle monitor spawn failure: detect early monitor death before `running` and clean up DB/command-dir.
- [ ] Support `--attach` after `State=running`.

## Phase 2: Monitor Core

- [ ] Redirect stdio away from the invoking terminal.
- [ ] Read `<command-dir>/config.json` on monitor startup.
- [ ] Allocate the PTY for the configured command.
- [ ] Start the configured command in the PTY slave.
- [ ] Update `CommandState` to `starting` during monitor startup.
- [ ] Write monitor PID into `CommandState.JSON`.
- [ ] Create `<runtime_dir>/cmd/<command_id>/` directory with user-only permissions (`0700`).
- [ ] Write monitor PID file at `<runtime_dir>/cmd/<command_id>/pid`.
- [ ] Start the gRPC server on `<runtime_dir>/cmd/<command_id>/monitor.sock`.
- [ ] Write socket path into `CommandState.JSON`.
- [ ] Write startup timestamp into `CommandState.JSON`.
- [ ] Update `CommandState` to `running` only after the monitor is ready.
- [ ] Handle monitor graceful shutdown: on SIGTERM, forward signal to child, wait for exit, then clean up.
- [ ] Update `CommandState.ExitCode` on process exit.
- [ ] Update `CommandState.State=exited` on process exit.
- [ ] Write finished timestamp into `CommandState.JSON` on process exit.
- [ ] Write runtime error details into `CommandState.JSON` on failures.
- [ ] Insert `CommandExitCode` row on each process exit.
- [ ] Clean up Unix socket and PID file on monitor exit.
- [ ] Implement annotation-driven auto-remove using `CommandConfig.JSON`.
- [ ] Resolve command-dir from `CommandConfig.JSON` during cleanup/removal paths.
- [ ] Delete `CommandConfig`, `CommandState`, `CommandExitCode`, and command-dir on auto-remove.

## Phase 3: Attach, Logs, and Status

- [ ] Define and generate `cmdmon.proto` with Attach, Logs, Signal, and Status RPCs.
- [ ] Implement monitor gRPC server.
- [ ] Implement attach fan-out for multiple clients.
- [ ] Implement attach stdin multiplexing.
- [ ] Implement resize handling through `AttachInput.resize`.
- [ ] Implement in-memory byte ring buffer for scrollback.
- [ ] Read scrollback limit from `CommandConfig.JSON`.
- [ ] Implement attach replay from the ring buffer.
- [ ] Implement `Logs(follow=true)` behavior.
- [ ] Add `cmd attach`.
- [ ] Add `cmd logs`.
- [ ] Add `cmd stop` (signal through monitor's `Signal` RPC).
- [ ] Add client-side detach key handling.
- [ ] Add `--no-stdin` support.
- [ ] Add `--sig-proxy` support.

## Phase 4: Query and Cleanup

- [ ] Add `cmd ls`.
- [ ] Resolve IDs and names from `CommandConfig`.
- [ ] Query labels from `CommandConfig.JSON` with SQLite JSON functions.
- [ ] Support `cmd ls -l KEY=VALUE`.
- [ ] Support `cmd stop -l KEY=VALUE` (extends `cmd stop` from Phase 3 with label-based bulk targeting).
- [ ] Support `cmd rm -l KEY=VALUE`.
- [ ] Implement stale-entry liveness checks using monitor PID from `CommandState.JSON`.
- [ ] On stale monitor detection, mark the command errored in `CommandState`.
- [ ] Add `cmd rm`.
- [ ] Resolve command-dir from `CommandConfig.JSON` for explicit `cmd rm`.
- [ ] Make `cmd rm` delete DB rows and command-dir.
- [ ] Add forced remove path for running commands.
- [ ] Add `cmd inspect`.
- [ ] Make `cmd inspect` return merged `CommandConfig`, `CommandState`, and `CommandExitCode`.
- [ ] Include generated `config.json` path in inspect output when useful.

## Phase 5: Restart and Lifecycle Policy

- [ ] Implement restart policy parsing in `CommandConfig.JSON`.
- [ ] Implement `no` restart behavior.
- [ ] Implement `on-failure` restart behavior.
- [ ] Implement `always` restart behavior.
- [ ] Re-read `config.json` on restart.
- [ ] Increment restart count in `CommandState.JSON` on restart.
- [ ] Preserve attach sessions across restarts.
- [ ] Record exit history for each restart cycle in `CommandExitCode`.

## Phase 6: Migration and Cleanup

- [ ] Remove `crabswarm server start`.
- [ ] Deprecate `pkg/mux/tmux`.
- [ ] Port interpolation logic from `pkg/mux/tmux/interpolate.go`.
- [ ] Support `#{COMMAND_ID}`, `#{COMMAND_NAME}`, and `#{INJECT_META}` interpolation.
- [ ] Apply interpolation to startup keys at send time.

## Verification

- [ ] Add schema tests for `CommandConfig`, `CommandState`, and `CommandExitCode`.
- [ ] Add tests for deferred foreign key behavior.
- [ ] Add tests for exit code range checks.
- [ ] Add tests for config JSON materialization to `config.json`.
- [ ] Add tests for `cmd run` -> monitor startup -> `State=running`.
- [ ] Add tests for monitor spawn failure detection and cleanup.
- [ ] Add tests for monitor graceful shutdown.
- [ ] Add tests for attach fan-out with multiple clients.
- [ ] Add tests for label filtering via SQLite JSON queries.
- [ ] Add tests for stale-entry cleanup.
- [ ] Add tests for auto-remove behavior.
- [ ] Add tests for restart policies.
- [ ] Add tests for `cmd inspect` merged output.
- [ ] Add end-to-end lifecycle test: `cmd run` → `cmd attach` → `cmd stop` → `cmd rm`.
