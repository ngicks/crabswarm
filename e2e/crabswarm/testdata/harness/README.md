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

## Claude Code agents listing

`claude-agents.json` is the output of

```sh
claude agents --json
```

captured 2026-09-24 with Claude Code 2.1.281, from a shell beside an
interactive session: a Bash tool call of that session itself, which runs in the
session's container and PID namespace. The command needs no TTY and prints every
session recorded under `<config home>/sessions/` as one JSON array; `--all` adds
finished background sessions. The capture holds one finished background session,
with `state` and no `status`, and the live interactive session, with `pid` and
`status` and no `state`. The file is the busy moment below, verbatim.

### The launcher isolation

The session ran under the `claude` command of a cmdman compose project, one
podman container per replica, launched as

```sh
FAKECHAT_PORT=9943 CRABSWARM_CLAUDE_CHANNEL=1 claude --channels plugin:fakechat@claude-plugins-official
```

Inside the container `claude` is pid 2 under `podman-init`, in a PID namespace of
its own. The registry directory `<config home>/sessions/` is a tmpfs mount of
its own, and `cc-socks/` sits under `/run/user/1000/`, a tmpfs of the container,
so no other session shares either. `df -aT` shows both mounts; the record
`sessions/2.json` carries `"pid":2`, the session's UUID and
`"pidDomain":"linux::pid:[…]"`, the namespace `readlink /proc/self/ns/pid`
prints from inside.

### The identity the spawned server sees

The environment of the `crabswarm mcp` process the session spawned, read from
`/proc/<pid>/environ`, carried

```
CLAUDE_CODE_SESSION_ID=76d3d5de-09b1-4cef-b786-1db9974f53ed
CLAUDE_CONFIG_DIR=/root/.config/claude
CLAUDE_CODE_ENTRYPOINT=cli
CLAUDECODE=1
CRABSWARM_CLAUDE_CHANNEL=1
FAKECHAT_PORT=9943
CMDMAN_CMD_ID=aa7a8c63a3cd8a425dade306d689cbff
XDG_RUNTIME_DIR=/run/user/1000/
```

and the listing's interactive entry carries that same UUID as `sessionId`. The
feed matches on it (`pkg/harnessctl/claudeagents.go`).

### The three moments

One sampler ran `claude agents --json` every 0.3 s and kept each output whose
status differed from the previous one. Each block is verbatim; the finished
background entry that precedes the interactive one in every output is left out
here and kept in the fixture.

Busy, while the main turn was over and a background subagent kept building and
testing; the status held without a change for the whole subagent run:

```json
{
  "pid": 2,
  "cwd": "/home/watage/gitrepo/github.com/ngicks/crabswarm",
  "kind": "interactive",
  "startedAt": 1790197322621,
  "sessionId": "76d3d5de-09b1-4cef-b786-1db9974f53ed",
  "name": "crabswarm-44",
  "status": "busy"
}
```

Waiting, while a permission dialog for a Bash tool call stood open:

```json
{
  "pid": 2,
  "cwd": "/home/watage/gitrepo/github.com/ngicks/crabswarm",
  "kind": "interactive",
  "startedAt": 1790197322621,
  "sessionId": "76d3d5de-09b1-4cef-b786-1db9974f53ed",
  "name": "crabswarm-44",
  "status": "waiting",
  "waitingFor": "permission prompt"
}
```

Idle, after a turn was interrupted with Esc and the session sat at the prompt:

```json
{
  "pid": 2,
  "cwd": "/home/watage/gitrepo/github.com/ngicks/crabswarm",
  "kind": "interactive",
  "startedAt": 1790197322621,
  "sessionId": "76d3d5de-09b1-4cef-b786-1db9974f53ed",
  "name": "crabswarm-44",
  "status": "idle"
}
```

Two things the capture taught about how to take one:

- A permission dialog only opens in a permission mode that asks. Under auto
  mode the classifier answers in the dialog's place and the status never leaves
  busy, so the waiting moment was recorded after switching the session to the
  default mode.
- A process the session's own Bash tool starts, detached or not, does not run
  once the turn ends, so a sampler inside the session sees neither the idle
  between turns nor the interruption. The idle moment was read from the host
  with `podman exec <container> claude agents --json`.

### The fields

The fields a state feed decodes, as the Claude Code documentation lists them on
the agent-view page under "List sessions as JSON":

| field | values | present when |
| --- | --- | --- |
| `kind` | `interactive`, `background` | always |
| `sessionId` | the session's UUID | when set |
| `pid`, `status` | `busy`, `waiting`, `idle` | the process is alive |
| `waitingFor` | `permission prompt`, `input needed`, `sandbox request`, `worker request`, `dialog open` | `status` is `waiting` |
| `state` | `working`, `blocked`, `done`, `failed`, `stopped` | background sessions only |

An interactive session carries `status` and never `state`; a background session
carries `state` always and `status` while its process lives. `status` is the
live answer and `state` the session's own account of its progress, so a feed
reads `status` first and falls back to `state` when it is absent. `waitingFor`
is display text, not a state of its own.

Two properties of the registry shape what a reader can rely on:

- The record is `<config home>/sessions/<pid>.json`, keyed by the harness's pid
  in its own PID namespace, and the listing drops a record whose pid or process
  start time no longer matches a live process in the caller's namespace. Two
  sessions in different PID namespaces sharing one config home can overwrite
  each other's record, so the listing is only trustworthy when each namespace
  has a registry of its own.
- A record from another PID namespace is printed without `status`, whatever the
  session is doing, so only a listing run beside the session sees its live
  status. Match entries by `sessionId` and never by `cwd`: the listing covers
  every session on the host.

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

## fakechat plugin channel

`fakechat-page.html`, `fakechat-upload.http`,
`fakechat-channel-notification.jsonl` and `fakechat-launch.md` record the
official `fakechat` plugin: the loopback HTTP server it runs beside its MCP
server, and the frame that server pushes into a Claude Code session which
registered the plugin as a channel. These four files supersede
`claude-channel-launch.md` and `claude-channel-notification.jsonl`, which stay
as history of the development-channel approach.

Captured 2026-09-22 on the same host with:

| tool | version |
| --- | --- |
| Claude Code | 2.1.278 |
| bun | 1.3.13 |
| fakechat plugin | 0.0.1 |
| cmdman | 0.0.26 |

The plugin's own `server.ts` made the recording. It ran by absolute path, with
its stdin held open so the MCP transport stayed up and its stdout stayed a log:

```sh
tail -f /dev/null | FAKECHAT_PORT=18787 \
  bun "$CLAUDE_CONFIG_DIR/plugins/cache/claude-plugins-official/fakechat/0.0.1/server.ts" \
  > out.log 2> err.log
```

`FAKECHAT_PORT` replaces the plugin's default 8787, which a live session on the
host already held. Every launch in this section sets it. `err.log` took the one
line the server prints on startup, `fakechat: http://localhost:18787`, and
`out.log` took nothing but MCP frames.

`fakechat-page.html` is the body the server answered `GET /` with, byte for
byte, captured the same day on bun 1.3.13 and plugin version 0.0.1:

```sh
mkfifo fifo
exec 3<>fifo
FAKECHAT_PORT=18790 \
  bun "$CLAUDE_CONFIG_DIR/plugins/cache/claude-plugins-official/fakechat/0.0.1/server.ts" \
  < fifo > out.log 2> err.log &
curl -s --retry-connrefused --retry 20 --retry-delay 1 \
  -o fakechat-page.html http://127.0.0.1:18790/
kill $!
```

The fifo holds the server's stdin open, which is what the `tail -f /dev/null`
in the launch above does. A fifo does it here because it closes with the shell
and leaves no writer behind to stop. The port is again one of its own, 18790,
and nothing was left listening on it afterwards. The page is a constant in
`server.ts` and names no port, so this is a page rather than a template: it is
identical whatever port the server was started on, and the
`<title>fakechat</title>` a probe recognises the plugin by is in it as the
plugin writes it.

`fakechat-upload.http` is the request the server answered 204 to, byte for byte
as curl sent it. A throwaway Go program listened on 28787, copied the client's
bytes into the fixture through an `io.MultiWriter` and forwarded them to 18787
unparsed, then copied the response back. curl connected to that relay while
addressing the server, so the recorded `Host` header names the real port:

```sh
curl --connect-to 127.0.0.1:18787:127.0.0.1:28787 \
  --form-string id=crabswarm-1 \
  --form-string 'text=[crabswarm chat] new message from team/bob — …' \
  http://127.0.0.1:18787/upload
```

The fixture holds the full `text` field; the line above shortens it. curl chose
the boundary `------------------------jJ5AtftPT4iZsyuUkIlYeo` and wrote it into
both the `Content-Type` header and the body, so no part of the file was picked
by hand. The relay copies raw bytes, so the CRLF line endings and the trailing
closing boundary are the ones that crossed the wire.

Two deviations from the obvious recipe:

- The capture went through the relay rather than `curl --trace-ascii`. The ascii
  trace drops the CR of each CRLF and replaces non-printable bytes with dots, so
  its output cannot be turned back into the request.
- The fields went in as `--form-string` rather than `-F`. `-F` reads a leading
  `@` or `<` in a value as a file reference, and the text starts with `[`.

`fakechat-channel-notification.jsonl` is the whole of `out.log` after that one
upload: a single `notifications/claude/channel` frame, 309 bytes including its
newline. The server wrote nothing else to stdout, so no line was left out of the
fixture. `params.meta.message_id` repeats the `id` field of the upload and ties
the frame to the request beside it. The server sends the frame without any
`initialize` handshake, since nothing was written to its stdin.

`fakechat-launch.md` records how an interactive session and a background session
each register the plugin, captured the way the Claude Code screens above were:

```sh
cmdman run -t --rm -n fakechat-launch -w <scratch dir> -- \
  env -u CLAUDE_CODE_CHILD_SESSION -u CLAUDE_CODE_SESSION_ID \
      -u CLAUDE_JOB_DIR -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT \
      FAKECHAT_PORT=18788 claude --model haiku \
      --channels plugin:fakechat@claude-plugins-official
cmdman capture-screen fakechat-launch
cmdman stop fakechat-launch
```

A `GET /` on the launch port proves the plugin server read `FAKECHAT_PORT` out of
the launch environment: it answers 200 with a page titled `fakechat`. The
background half ran the same `claude` line under `--bg --name <x>` with a prompt
in front of `--channels`, and a post to its own port started a turn there too.

## Not captured

Nothing on the list is missing. The OpenCode `initialize` was captured live
rather than read out of the project's source, which is a better record than the
source would have been.
