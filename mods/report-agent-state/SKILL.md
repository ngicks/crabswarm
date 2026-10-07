---
name: report-agent-state
description: Explains the `report-agent-state` mod that reports this Claude Code session's state (working, waiting, done) from `claude agents --json` by running a configured command, crabswarm's `chat report-state` by default. Use when the user asks why a state is reported for this session, or wants to change or debug it.
disable-model-invocation: true
---

# report-agent-state

This skill folder is also a Claude Code plugin named `report-agent-state`.
Claude Code adopts every folder under `~/.claude/skills/` that holds `.claude-plugin/plugin.json`,
and loads its hooks module (`hooks/register.tsx`) as a mod.

## What it reports

The mod maps the session's entry in `claude agents --json`:

| Listing | Reported |
| --- | --- |
| `status: busy` | `working` |
| `status: waiting` | `waiting` |
| `status: idle` | `done` |
| no `status`, `state: working` | `working` |
| no `status`, `state: blocked` | `waiting` |
| no `status`, `state: done/failed/stopped` | `done` |

- `status` is read first; `state` is the fallback for an entry whose process the listing could not see.
- An entry without a `pid` is never picked.
  - Those are background job records, often stale.

## When it reports

- At session start, then every 2 seconds.
- The session id is read on every tick.
  - `/clear`, `/resume` and the like switch it inside the same process without loading the mod again.
- A state is reported when it changes, and again every 10 seconds while it stays.
  - The re-send lets a reporter that restarted hear the state again.
- A listing without the session, or a word the mod does not know, reports nothing.

## Options

- `reportCommand`: the command and its arguments as a list, run with the state word appended; empty turns the mod off.
  - Default `["crabswarm", "chat", "report-state"]`.
- `requireEnv`: a list of variable names; the mod runs only when at least one is set; empty always runs.
  - Default `["CRABSWARM_CHAT_TOKEN", "CMDMAN_CMD_ID"]`.
  - Checked once at session start, through `printenv`.
- Override them under `pluginConfigs["report-agent-state@skills-dir"].options` in the user settings file or a `--settings` file.
  - Project settings are not read for `pluginConfigs`.
  - Under `--plugin-dir` the key is `report-agent-state` or `report-agent-state@inline`.

## Files

- `hooks/register.tsx`: reads the options and wires the loop to the engine at session start.
- `hooks/watch.ts`: `agentStates`, an async iterable of the listed state, and `reportStates`, which hands each state to a `Report`.
- `hooks/state.ts`: picks the session's entry and maps it.
- `tests/`: run with `claude plugin test <this folder>`.

## Debugging

- Failures go to the debug log (`claude --debug`), prefixed `report-agent-state:`.
  - The first failure of a run is logged, then one in every 150.
- `claude plugin disable report-agent-state@skills-dir` turns it off.
