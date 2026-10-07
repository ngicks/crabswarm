# report-agent-state

A Claude Code mod that reports what its session is doing by running a command.

Every 2 seconds the mod reads `claude agents --json`, picks the entry of the current session, and runs the report command with a state word appended: `working`, `waiting` or `done`.
The listing is Claude Code's own account of the session:

- A turn interrupted with Esc reads `idle` and is reported `done`.
- A background subagent, shell or monitor running behind an idle main loop reads `busy` and is reported `working`.
- A permission prompt or another open dialog reads `waiting`.

The mod infers nothing from the session's own events, since an interrupted turn fires none.
`SKILL.md` describes the details.

## Options

| Option | Default | Meaning |
| --- | --- | --- |
| `reportCommand` | `["crabswarm", "chat", "report-state"]` | The command and its arguments, run with the state word appended. Empty turns the mod off. |
| `requireEnv` | `["CRABSWARM_CHAT_TOKEN", "CMDMAN_CMD_ID"]` | The mod runs only when the session's environment sets at least one of these. Empty always runs. |

Both are lists of strings (`"type": "string", "multiple": true` in `plugin.json`).

The defaults report to a [crabswarm](https://github.com/ngicks/crabswarm) chat room.

To override them, add `pluginConfigs.<plugin id>.options` to one of these settings files:

- the user settings file, `~/.claude/settings.json` (`$CLAUDE_CONFIG_DIR/settings.json` when that variable is set);
- a file passed with `claude --settings <file>`, for one session;
- managed settings.

Claude Code does not read `pluginConfigs` from a project's `.claude/settings.json` or `.claude/settings.local.json`.

The plugin id depends on how the mod was loaded:

- `report-agent-state@skills-dir` when apm installed it into `~/.claude/skills/`;
- `report-agent-state` or `report-agent-state@inline` under `claude --plugin-dir`.

```json
{
  "pluginConfigs": {
    "report-agent-state@skills-dir": {
      "options": {
        "reportCommand": ["my-tool", "state", "set"],
        "requireEnv": ["MY_TOKEN"]
      }
    }
  }
}
```

`claude --debug` logs the keys it looked for when it finds none.

The report command runs in the session's environment.
A non-zero exit counts as a failed report, which is retried on the next tick.

## Requirements

- `claude`, `printenv` and the report command on `PATH` of the Claude Code process.

The mod runs inside the Claude Code process.
The listing is therefore read in the session's own PID namespace and config directory, wherever whatever records the state runs.

## Install

Install it globally with apm:

```bash
apm install -g ngicks/crabswarm/mods/report-agent-state
```

apm copies this folder to `~/.claude/skills/report-agent-state/`.
Claude Code adopts it as the plugin `report-agent-state@skills-dir` in the next session.
Run `/reload-plugins` to load it into a running session.

For a one-off session from a checkout:

```bash
claude --plugin-dir ./mods/report-agent-state
```

## Develop

```bash
claude plugin validate mods/report-agent-state
claude plugin test mods/report-agent-state
```

Claude Code writes the API types to `.claude-plugin/types/` whenever it loads the mod from this folder.
After that, `tsc -p mods/report-agent-state` type-checks it.

## Layout notes

- The name carries no `claude`.
  - `claude plugin validate` warns on a plugin name that reads as one of Anthropic's own.
- `hooks/hooks.json` carries a `description` key.
  - apm merges any `hooks/*.json` whose values are all lists into `settings.json` as settings hooks.
  - A string value stops that merge, so `modules` stays out of the user's settings.
- `.claude-plugin/plugin.json` makes apm treat this folder as a Claude plugin and deploy it whole into `~/.claude/skills/`.
