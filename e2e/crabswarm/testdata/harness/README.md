# Harness probe fixtures

Recordings of what the real harnesses do. Nothing regenerates these files, so
treat them as read-only evidence: never hand-edit the content, re-capture
instead and replace a file whole.

Captured 2026-09-17 on Linux 6.18.33.2-microsoft-standard-WSL2 (x86_64) with:

| tool | version |
| --- | --- |
| Claude Code | 2.1.274 |
| Codex | 0.154.0 |
| OpenCode | 1.18.31 |
| cmdman | 0.0.25 |

## MCP `initialize` per harness

`initialize-claude.json`, `initialize-codex.json` and `initialize-opencode.json`
each hold the first line the harness wrote to an MCP stdio server, so the
`clientInfo` in them is how a harness names itself.

The server was a two-line wrapper that tees the harness's side of the stream to
a log and then execs a build of this repository's `crabswarm mcp`:

```sh
#!/bin/sh
tee -a "$PROBE_LOG" | "$PROBE_SERVER" mcp
```

Each harness was pointed at that wrapper and started once; the first line of the
log became the fixture.

- Claude Code: an MCP config file declaring the wrapper under
  `mcpServers.probe`, passed on the command line. It names itself
  `claude-code`.
- Codex: the same wrapper registered as an MCP server in the Codex
  configuration. It names itself `codex-mcp-client`.
- OpenCode: an `opencode.json` in an empty directory declaring the wrapper as
  `mcp.probe` with `"type": "local"`, then the TUI started in that directory
  under cmdman. It names itself `opencode` and sends its own version as
  `clientInfo.version`. The MCP handshake runs at startup, before any model
  provider is configured, so no provider was needed. The `opencode` on `PATH`
  resolves to a stale install directory and does not run; the binary used was
  `1.18.31` under the mise install root, invoked by absolute path.

## Claude Code channel registration

`claude-channel-launch.md` records how an interactive session registers a
development channel and why a background (`claude --bg`) session cannot.

`claude-channel-notification.jsonl` is the one
`notifications/claude/channel` frame a minimal stdio MCP server pushed into a
registered session, taken from that server's own log:

```sh
grep '^OUT ' channel.log | grep 'notifications/claude/channel' | cut -c5-
```

The server declared `capabilities.experimental["claude/channel"] = {}` in its
`initialize` result and emitted the frame fifteen seconds after
`notifications/initialized`. Claude Code delivered it as a new turn.

## Claude Code screens

`claude-screen-idle.txt`, `claude-screen-working.txt` and
`claude-screen-dialog.txt` are full-screen snapshots of one interactive session:

```sh
cmdman run -t --rm -n claude-probe -w <scratch dir> -- claude --model haiku
cmdman send-keys -l claude-probe "<prompt text>"
cmdman send-keys claude-probe Enter
cmdman capture-screen claude-probe > <fixture>
```

`capture-screen` ran with no flags: the main screen, escape sequences stripped.
A classifier fed these fixtures has to read the same capture mode. The PTY was
cmdman's default 80x24, so long lines are truncated with a `…`.

The session's status line is `crabswarm statusline render`, configured in the
Claude Code settings as

```
crabswarm statusline render '{{.Model.DisplayName}}{{ with .Effort }} @ {{ .Level }}{{ end }} | {{.ContextWindow.UsedPercentage}}% used | {{.Workspace.CurrentDir}}'
```

so the bottom of a capture reads `Haiku 4.5 | 16% used | /tmp/…` rather than
Claude Code's own footer hints. The idle and working captures show it. The
dialog capture does not: an open permission dialog replaces the whole footer
region, status line included, and that absence is itself the strongest marker of
the dialog state.

What marks each state:

- `claude-screen-idle.txt` — the prompt line `❯` stands empty between two rule
  lines, the last turn ends in `✻ Worked for 1s · done 11:08 PM`, and the status
  line and `⏸ manual mode on · ← for agents` close the screen. No spinner.
- `claude-screen-working.txt` — the spinner line `* Generating… (19s · ↓ 193
  tokens)`, the running tool above it
  (`⎿  $ python3 -c 'import time; time.sleep(30)' (16s)` followed by
  `(ctrl+b to run in background)`), and an elapsed counter on the tool header
  (`Running Python sleep for 30 seconds · 1m 0s`). The empty `❯` prompt and the
  status line are still there, so an empty prompt alone does not mean idle.
- `claude-screen-dialog.txt` — the block
  `This command requires approval` / `Do you want to proceed?` with the numbered
  choices `❯ 1. Yes`, `2. Yes, and don’t ask again for: python3 *`, `3. No`, and
  the hint `Esc to cancel · Tab to amend` on the last row. The command under
  review is quoted above the question.

Two deviations from the obvious recipe, both forced by the harness:

- The working capture ran `python3 -c 'import time; time.sleep(30)'` instead of
  `sleep 25`. Claude Code's Bash tool refuses a bare foreground `sleep` and
  reports that it is blocked, or runs it in the background where it paints no
  spinner.
- The dialog was answered with Enter (approve) rather than Esc, because the same
  dialog gated the command the working capture needed running. The dialog
  fixture was written before the answer.

The session ran in a scratch directory outside any repository, and the launch
cleared `CLAUDE_CODE_CHILD_SESSION`, `CLAUDE_CODE_SESSION_ID`, `CLAUDE_JOB_DIR`,
`CLAUDECODE` and `CLAUDE_CODE_ENTRYPOINT` so the captures do not carry a
transcript-saving warning from the launching session.

## Codex app-server

`codex-app-server.client.ndjson` and `codex-app-server.server.ndjson` are one
session split by direction: every frame the client sent, in order, and every
frame the server sent, in order. The split keeps each message byte-for-byte as
it crossed the wire; a single interleaved file would need a direction field
inside the messages.

The server ran as

```sh
codex app-server --listen unix:///tmp/crabswarm-probe.sock
```

The socket path has to stay short — a path under a long scratch directory fails
with `Error: path must be shorter than SUN_LEN`.

The endpoint speaks WebSocket, not newline-delimited JSON. A raw upgrade request
over the unix socket answers:

```
HTTP/1.1 101 Switching Protocols
connection: Upgrade
upgrade: websocket
sec-websocket-accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

A TUI was attached first so a thread existed, and given one short prompt:

```sh
cmdman run -t --rm -n codex-probe -w <scratch dir> -- codex --remote unix:///tmp/crabswarm-probe.sock
```

The recording itself came from a throwaway Go client written outside this
repository (`github.com/coder/websocket` plus `encoding/json`), which dialed the
unix socket, wrote each frame it sent and each frame it received to its own
file, and drove this exchange: `initialize` with `clientInfo.name`
`crabswarm-probe`, an `initialized` notification, `thread/loaded/list`,
`thread/list`, `thread/read`, `thread/resume`, a first `turn/start` read through
to `turn/completed`, then a second `turn/start` interrupted twelve seconds in
with `turn/interrupt`.

Method names and parameter shapes came from the server's own schema rather than
from documentation:

```sh
codex app-server generate-json-schema --out <dir>
```

Worth knowing when replaying this against a fake server:

- `initialize` answers with the server's `userAgent`, `codexHome`,
  `platformFamily` and `platformOs`. It carries no thread information.
- `initialized` is a notification with no params. The protocol's client
  notification set holds only that one method.
- `thread/loaded/list` answers `{"data": [<thread id>], "nextCursor": null}` —
  bare ids, not objects. `thread/list` answers thread objects under `data`. The
  call here passes `limit: 1`: the listing is sorted newest first and covers
  every Codex session on the host, so a wider page would put unrelated sessions'
  first messages and working directories into the recording.
- `turn/start` answers with the turn object, so the id to interrupt with is
  `result.turn.id`; there is no `turnId` field in the response.
- `turn/interrupt` answers `{}`, which says nothing about the outcome. The
  outcome shows up in `turn/completed`, whose `turn.status` reads `interrupted`
  for the interrupted turn and `completed` for the first one.
- `thread/status/changed` alternates `{"type": "active", "activeFlags": []}` and
  `{"type": "idle"}`. `activeFlags` stayed empty for the whole session,
  including while a shell command ran.
- The server sent no approval request. Its approval policy let
  `zsh -lc 'sleep 20'` run unprompted, so this recording holds no
  `item/commandExecution/requestApproval` exchange. Anything that needs the
  approval path has to force it with a stricter `approvalPolicy` on
  `turn/start` and re-capture.
- `hook/started` and `hook/completed` frames appear around every turn because
  the host has Codex hooks configured. A fake server does not have to emit them.
- Notification frames carry an `emittedAtMs` field alongside `params`.

### Helper threads beside the session

`codex-app-server-helpers.client.ndjson` and
`codex-app-server-helpers.server.ndjson` are a second pair in the same split,
captured 2026-09-22 with Codex 0.155.1 and cmdman 0.0.26, of an app server
holding more than the one session thread. They hold `initialize`,
`initialized`, `thread/loaded/list`, and one `thread/read` with
`includeTurns: false` per loaded id. No `thread/list`: that listing is
host-wide and would put every session on the host into the fixture.

The server and a TUI ran under cmdman:

```sh
cmdman run --rm -n cx-probe-server -w <scratch dir> -- codex app-server --listen unix:///tmp/cxprobe.sock
cmdman run --rm -t -n cx-probe-tui -w <scratch dir> -- codex --remote unix:///tmp/cxprobe.sock
cmdman send-keys -l cx-probe-tui 'Reply with the single word ok.'
sleep 0.3
cmdman send-keys cx-probe-tui Enter
```

The prompt and the Enter go in separate `send-keys` calls with a pause between
them; sent together, the TUI reads the newline as part of the text. The
recorder ran within a minute of the turn ending. The TUI was then stopped and
a second one started on the same socket and given one more turn, which is the
state the pair records.

What it shows, and what it means for a client binding a delivery to a thread:

- Three ids are loaded: the first TUI's thread, the second TUI's thread, and a
  helper Codex opened by itself.
- Both TUI threads answer `threadSource: "user"`. The helper answers
  `threadSource: "system"`, `ephemeral: true` and `name: null`.
- Every other candidate field is the same on all three: `source: "vscode"`,
  `parentThreadId: null`, `canAcceptDirectInput: true`, `originator:
  "codex-tui"`.
- `thread/list` was probed separately and left out of the fixture. The
  `system` helper appears in no listing; guardian reviewer threads appear
  with `source: {"subAgent": {"other": "guardian"}}` and `threadSource: null`,
  as does every thread in a listing.

Timing seen while polling, not recorded in the pair:

- The `system` helper appears after a session's first turn, when Codex names
  the thread, and unloads within about a minute.
- A stopped TUI's thread stays loaded for about two minutes.
- A guardian thread never stayed loaded while a command was auto-reviewed,
  polled at 1.5 s.
- A connection that has only done the handshake receives `thread/started`
  when a TUI attaches to the app server and starts its thread, so a client
  that connected before the session existed hears about it without polling.

## Not captured

Nothing on the list is missing. The OpenCode `initialize` was captured live
rather than read out of the project's source, which is a better record than the
source would have been.
