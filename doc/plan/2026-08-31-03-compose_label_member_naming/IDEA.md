# Default member name from cmdman compose labels

Gate: not confirmed (automatic decisions, pending user review)

## How it should be

A chat member's name should tell a human — and the other agents — *which
command* is talking, not which random token it happened to hold. Today a
joiner that reports no `--name` is called `agent-<first-8-hex-of-token>`
(e.g. `agent-01234567`), which is meaningless to everyone in the room.

cmdman-compose already stamps every command it brings up with the labels
`cmdman.compose.command` (the command's name in the compose file) and
`cmdman.compose.scale-index` (its replica index). The default member name
must be derived from those: a member of a compose-scaled fleet reads as
`claude-1`, `claude-2`, `codex-1` — instantly recognizable in `chat members`
output, in message `from` lines, and in cmdman's own status display.

## Use case

- **Actor**: an operator running a compose project of several harnesses, and
  the agents themselves addressing each other.
- **Situation**: harnesses auto-join the room via the SessionStart hook (or,
  later, the per-agent MCP server) without passing `--name`.
- **Intent**: know who is who without cross-referencing tokens.
- **Walkthrough**: the operator runs `crabswarm chat members`; every default-
  named member shows as `<command>-<scale-index>` matching the compose file.
  An agent replying to `claude-2` types exactly that name. Nothing changes
  for a member that passed an explicit `--name` — that always wins.

## Usability requirements

- Explicit `--name` continues to override everything; the label-derived name
  is only the *default*.
- A command outside the labels' reach (labels missing) degrades to the
  current token-derived fallback rather than failing the join.
- Duplicate names in a team keep being rejected clearly at join time — the
  operative rule from the chat subcommand plan
  (`doc/plan/2026-08-26-01-chat_subcommand/IDEA.md:143-145`): "duplicate
  participant name in a team → clear rejection at join time, not silent
  aliasing". The scale-index in the default name is what keeps compose
  replicas of the same command from colliding naturally.
