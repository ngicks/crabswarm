# Changelog

## Unreleased

- Delete a chat database whose `members` table lacks the `kind` or
  `state_reported_at` column and let the daemon write a fresh one. The daemon
  applies the DDL on open and never migrates, so an older file fails every
  member query. The default path is `~/.local/state/crabswarm/chat.db*`, and
  `crabswarm config` prints the resolved one.
- `--kind agent|human` replaces `crabswarm chat join --agent` and is required.
- `crabswarm chat members`, `crabswarm chat admin list`, the MCP members
  resource and the admin TUI print each member's kind.
- The MCP bridge attends the room for the whole session. It retries until the
  daemon answers and attends again after a daemon restart.
- The MCP bridge serves its tools with no identity token. Each tool reports the
  missing token instead of the process exiting during startup.
- The daemon reaps a member whose cmdman command has exited and frees the name
  that member held.
- An unnamed joiner defaults to `agent-<token prefix>` or
  `human-<token prefix>`, matching the kind it declared.
- `crabswarm chat admin register` takes over a name held by an agent whose
  cmdman command is gone.
- The `crabswarm-chat` package forwards `CMDMAN_CMD_ID`, `CRABSWARM_CHAT_TOKEN`
  and `XDG_RUNTIME_DIR` to the bridge, so Codex spawns it with an identity token
  and the runtime dir the socket path comes from.
- The default socket path probes `/run/user/<uid>` when `XDG_RUNTIME_DIR` is
  unset, and falls back to `/tmp` only when that directory is missing.
- Every command logs warnings and above to stderr without `--log`.
- `crabswarm serve` creates the socket's directory before it takes the lock.
