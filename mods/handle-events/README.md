# handle-events

A Claude Code mod that follows an event source for the whole session.

The source is a command that prints one JSON record per line.
The mod shows `status` records on its status line.
It injects `message` records flagged `inject: true` into the session as a prompt, between turns only.
After the prompt entered, the mod runs an ack command once per injected record.

The defaults follow a [crabswarm](https://github.com/ngicks/crabswarm) chat room with `crabswarm chat follow`, inject the messages that mention this session, and mark each one read with `crabswarm chat read --skip <seq>`.
`SKILL.md` describes the details.

## Protocol

- The source writes one JSON object per line to standard output, UTF-8, each line ended with `\n` and flushed as soon as it is written.
- Every record carries a string `type` and a string `message`.
- `{"type":"status","message":"..."}` sets the mod's status line to `message`. An empty `message` clears it. The mod clears the status line itself 5 seconds after the latest entry.
- `{"type":"message","message":"...","inject":true}` is injected. A message whose `inject` is anything but `true` is ignored.
- Other types and other fields are ignored. The ack command reads fields by name.
- A line that is not such an object is dropped and logged to the debug log.
- Standard error is free for the source's own use. The mod does not read records from it; its last line goes into the restart message.

```json
{"type":"status","message":"following room-a"}
{"type":"message","message":"alice: please review the plan","inject":true,"seq":42}
```

## When a message is injected

- A record that arrives while the main loop is idle is submitted at once.
- A record that arrives while a main-loop turn runs waits until that turn ends. A subagent's turn does not count.
- Every waiting record goes into one prompt: one line per record, `<prefix> <message>`, in arrival order. A line equal to an earlier line of the same prompt is left out.
- The prompt is submitted as a plugin prompt, so the transcript names the mod as its sender. Claude Code never folds it into a running turn.
- Once the prompt entered, the ack command runs once per record of the prompt, in order, the left-out duplicates included.

## When the source exits

- The mod restarts it after a pause of 1 second, doubled after each restart up to 30 seconds.
- A run that printed a valid record or lasted 30 seconds resets the pause to 1 second.
- When it restarts, the status line says why the source ended and how long the restart waits. The next valid record clears it, and so does the 5-second hold.

## Options

| Option | Default | Meaning |
| --- | --- | --- |
| `source` | `["crabswarm", "chat", "follow"]` | The source command and its arguments. Empty turns the mod off. |
| `mapper` | `[]` | A command run once per source line, with the line on standard input. Each non-empty line it prints is a record. A non-zero exit drops the line. Empty reads the source lines as records. |
| `prefix` | `"[crabswarm chat]"` | The text before each injected message, separated by a space. Empty puts nothing. |
| `ack` | `["crabswarm", "chat", "read", "--skip", "{seq}"]` | The command run once per injected record. An item written exactly `{name}` is replaced by the record's top-level field `name`. Empty acks nothing. |
| `requireEnv` | `["CRABSWARM_CHAT_TOKEN", "CMDMAN_CMD_ID"]` | The mod runs only when the session's environment sets at least one of these. Empty always runs. |

`source`, `mapper`, `ack` and `requireEnv` are lists of strings (`"type": "string", "multiple": true` in `plugin.json`); `prefix` is a string.

- A placeholder takes a string, a number or a boolean. A record whose field is missing or holds anything else is not acked, and the debug log says so.
- The ack command comes from the options alone. A record cannot name a command.
- A mapper suits a source that does not speak the protocol, for example `["jq", "-c", "{type: \"message\", message: .text, inject: true, id}"]`.

To override them, add `pluginConfigs.<plugin id>.options` to one of these settings files:

- the user settings file, `~/.claude/settings.json` (`$CLAUDE_CONFIG_DIR/settings.json` when that variable is set);
- a file passed with `claude --settings <file>`, for one session;
- managed settings.

Claude Code does not read `pluginConfigs` from a project's `.claude/settings.json` or `.claude/settings.local.json`.

The plugin id depends on how the mod was loaded:

- `handle-events@skills-dir` when apm installed it into `~/.claude/skills/`;
- `handle-events` or `handle-events@inline` under `claude --plugin-dir`.

```json
{
  "pluginConfigs": {
    "handle-events@skills-dir": {
      "options": {
        "source": ["my-tool", "events", "--json"],
        "prefix": "[my-tool]",
        "ack": ["my-tool", "events", "ack", "{id}"],
        "requireEnv": []
      }
    }
  }
}
```

`claude --debug` logs the keys it looked for when it finds none.

Every command runs in the session's environment and working directory.

## Known behaviour

- The default ack, `crabswarm chat read --skip {seq}`, also marks earlier unread mentions read.
  - The read position is one sequence number per room, and `--skip` moves it through `seq`.
  - A mention that arrived before the injected one, and that the source never flagged, is marked read with it.
- A record stays acked when the turn of its prompt is interrupted. The prompt had already entered the session.
- A prompt that does not enter (a hook drops it) is not retried, and its records are not acked.

## Requirements

- `printenv`, and the source, mapper and ack commands, on `PATH` of the Claude Code process.

## Install

Install it globally with apm:

```bash
apm install -g ngicks/crabswarm/mods/handle-events
```

apm copies this folder to `~/.claude/skills/handle-events/`.
Claude Code adopts it as the plugin `handle-events@skills-dir` in the next session.
Run `/reload-plugins` to load it into a running session.

For a one-off session from a checkout:

```bash
claude --plugin-dir ./mods/handle-events
```

## Develop

```bash
claude plugin validate mods/handle-events
claude plugin test mods/handle-events
```

Claude Code writes the API types to `.claude-plugin/types/` whenever it loads the mod from this folder.
After that, `tsc -p mods/handle-events` type-checks it.

## Layout notes

- The name carries no `claude`.
  - `claude plugin validate` warns on a plugin name that reads as one of Anthropic's own.
- `hooks/hooks.json` carries a `description` key.
  - apm merges any `hooks/*.json` whose values are all lists into `settings.json` as settings hooks.
  - A string value stops that merge, so `modules` stays out of the user's settings.
- `.claude-plugin/plugin.json` makes apm treat this folder as a Claude plugin and deploy it whole into `~/.claude/skills/`.
