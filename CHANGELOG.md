# Changelog

## Unreleased

- The `crabswarm-chat` and `crabswarm-issues-lint` packages ship their Claude
  Code wiring as skills-directory plugins. apm copies each package's skill
  directory, which now carries `.claude-plugin/plugin.json`, `hooks/hooks.json`
  and, for `crabswarm-chat`, `.mcp.json`; Claude Code loads the directory as
  `<name>@skills-dir` and merges nothing into `settings.json`. A hook a later
  version drops disappears with its file. Codex keeps a merged hooks file,
  routed to Codex alone by its `codex-hooks.json` stem.
- After upgrading, delete every `crabswarm chat` and `crabswarm issues lint`
  hook entry from `~/.claude/settings.json`, a project's
  `.claude/settings.json` and the `apm-hooks.json` beside each. apm never
  removes them, and each such hook otherwise runs twice on Claude Code.
- `crabswarm-issues-lint` gains a skill that explains a lint finding and how to
  fix the issue text.
- `crabswarm-chat` targets OpenCode. Its skill directory carries `opencode.ts`,
  an OpenCode plugin that declares the bridge, reports the session's state and
  delivers messages, wired once through `"plugin": ["./skills/crabswarm-chat/opencode.ts"]`
  in `opencode.json`.
- The chat database has a new layout: rooms, messages, mentions and read
  positions, with no members table and no token on disk. Delete the old
  `~/.local/state/crabswarm/chat.db*` before the first start; `crabswarm config`
  prints the resolved path. The daemon applies the DDL on open and never
  migrates, so an old file starts the daemon and then fails every send and read.
- Attendance is an open `Attend` stream and nothing else, so
  `crabswarm chat join` and `crabswarm chat leave` are gone. A bridge attends
  for as long as its session lasts and leaves the listing the moment it exits.
  `crabswarm chat admin register` puts a person in attendance until the daemon
  restarts, which is the one attendance no stream holds.
- `crabswarm chat send <everyone|role[,role...]|""> <text>` addresses a message
  by its first argument: the whole room, a comma-separated list of roles, or the
  literal `""` for a board post that mentions nobody. A role nobody is attending
  under is warned about on stderr and its mention waits at that role's read
  position; a role that has never attended the room is refused outright.
- `crabswarm chat broadcast` and `crabswarm chat history` are gone.
  `chat send everyone` and `chat read` cover what they did.
- `crabswarm chat read` takes `--cursor unread|head|tail`, `--range`, `--to`,
  `--since` and `--until`. Every read moves the caller's read position to the
  newest message it showed, the tail included, so reading old history never
  un-reads what came after it. A read that leaves unread mentions behind ends
  with an `N more unread` line.
- `crabswarm chat admin delete-room <room>` removes a room's messages, the
  mentions in them and its read positions, and is refused while anybody attends
  the room. Nothing else removes one: a room outlives every session that used it
  and stays in `crabswarm chat admin list` until it is deleted.
- `crabswarm chat admin move` is gone. It rewrote a member's team in a table
  that no longer exists; a team now comes from the token at attendance, or from
  `crabswarm chat admin register`, and lasts as long as that attendance does.
- `crabswarm chat admin log` takes the same filter as `chat read` and still
  moves no read position. The unread cursor is refused there: unread is measured
  from a read position, and an operator attending no room has none.
- The MCP bridge drops the `chat_broadcast` tool and the history resource.
  `chat_send` takes the target argument the CLI takes and `chat_read` the same
  filter, so an agent and a person address the room the same way.
- `chat.history_limit` caps how many messages each room keeps, pruned as new
  ones arrive. Zero means 1000; a negative value prunes nothing, for a host
  whose rooms are theirs to clear by hand.
- `crabswarm chat members`, `crabswarm chat admin list`, the MCP members
  resource and the admin TUI print each member's kind.
- The MCP bridge attends the room for the whole session. It retries until the
  daemon answers and attends again after a daemon restart.
- The MCP bridge serves its tools with no identity token. Each tool reports the
  missing token instead of the process exiting during startup.
- An attendee that asks for no name, and whose provider derives none, defaults
  to `agent-<token prefix>` or `human-<token prefix>`, matching the kind it
  declared.
- The `crabswarm-chat` package forwards `CMDMAN_CMD_ID`, `CRABSWARM_CHAT_TOKEN`
  and `XDG_RUNTIME_DIR` to the bridge, so Codex spawns it with an identity token
  and the runtime dir the socket path comes from.
- The default socket path probes `/run/user/<uid>` when `XDG_RUNTIME_DIR` is
  unset, and falls back to `/tmp` only when that directory is missing.
- Every command logs warnings and above to stderr without `--log`.
- `crabswarm serve` creates the socket's directory before it takes the lock.
