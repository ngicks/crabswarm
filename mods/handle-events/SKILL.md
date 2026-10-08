---
name: handle-events
description: Explains the `handle-events` mod that follows an event source (crabswarm's `chat follow` by default), shows its status records on the status line, injects flagged messages as a prompt between turns, and acks each one with a configured command. Use when the user asks why a prompt or a status line came from this mod, or wants to change or debug it.
disable-model-invocation: true
---

# handle-events

This skill folder is also a Claude Code plugin named `handle-events`.
Claude Code adopts every folder under `~/.claude/skills/` that holds `.claude-plugin/plugin.json`,
and loads its hooks module (`hooks/register.tsx`) as a mod.

## What it reads

The source prints one JSON object per line on standard output.

| Record | Effect |
| --- | --- |
| `type: status` | `message` becomes the status line; an empty one clears it |
| `type: message`, `inject: true` | queued for injection |
| `type: message`, any other `inject` | ignored |
| any other `type` | ignored |
| not a JSON object with string `type` and `message` | dropped, logged |

- Standard error is not read for records.
- With a `mapper`, each source line goes to the mapper's standard input, and each non-empty line it prints is a record.
  - A mapper that exits non-zero drops the line.

## When it injects

- The main loop is idle: at once.
- A main-loop turn runs: when it ends. A subagent's turn does not count.
- A prompt is on its way in: when its turn ends.
- One prompt carries every queued record, one line each: `<prefix> <message>`, in arrival order.
  - A line equal to an earlier line of the same prompt is left out.
- The prompt is a plugin prompt, never folded into a running turn.

## Acks

- After the prompt entered, `ack` runs once per record of the prompt, in order, one after another.
- An item written exactly `{name}` is the record's top-level field `name`: a string, a number or a boolean.
  - A missing field or any other value skips that record's ack.
- A dropped prompt acks nothing.
- The default, `crabswarm chat read --skip {seq}`, moves the read position, one sequence number per room, through `seq`.
  - Earlier unread mentions are marked read with it.

## Restarts

- A source that exits is restarted after 1 second, doubling up to 30 seconds.
- A run that printed a valid record or lasted 30 seconds resets the pause.
- The status line says the source is restarting until the next valid record.

## Options

- `source`: the source command as a list; empty turns the mod off.
  - Default `["crabswarm", "chat", "follow"]`.
- `mapper`: a command as a list, run per line; empty reads lines as records.
  - Default `[]`.
- `prefix`: the text before each injected message.
  - Default `[crabswarm chat]`.
- `ack`: the command as a list, run per injected record, with `{name}` placeholders; empty acks nothing.
  - Default `["crabswarm", "chat", "read", "--skip", "{seq}"]`.
- `requireEnv`: a list of variable names; the mod runs only when at least one is set; empty always runs.
  - Default `["CRABSWARM_CHAT_TOKEN", "CMDMAN_CMD_ID"]`.
  - Checked once at session start, through `printenv`.
- Override them under `pluginConfigs["handle-events@skills-dir"].options` in the user settings file or a `--settings` file.
  - Project settings are not read for `pluginConfigs`.
  - Under `--plugin-dir` the key is `handle-events` or `handle-events@inline`.

## Files

- `hooks/register.tsx`: reads the options, gates on `requireEnv`, and wires the parts below to the engine.
- `hooks/protocol.ts`: parses a line into a record and says what it asks for.
- `hooks/lines.ts`: cuts output pieces into lines.
- `hooks/source.ts`: runs the source with restarts, and the mapper per line.
- `hooks/queue.ts`: tracks the main loop's turns and writes the prompt.
- `hooks/ack.ts`: fills the placeholders and runs the ack per record.
- `hooks/host.ts`: the engine calls the parts take, and the `requireEnv` check.
- `tests/`: run with `claude plugin test <this folder>`.

## Debugging

- Drops, ack failures and restarts go to the debug log (`claude --debug`), prefixed `handle-events:`.
  - The first of a run is logged, then one in every 100.
- `claude plugin disable handle-events@skills-dir` turns it off.
