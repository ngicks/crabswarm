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

Seen with Codex 0.156.1, not recorded in the pair:

- A TUI's thread is loaded and answers `threadSource: "user"` from the moment
  the TUI attaches, before any turn. `thread/resume` on it fails until the
  first turn has run, with
  `[-32600] no rollout found for thread id <id>`: resuming reads the rollout
  file the first turn writes. `turn/start` on that thread works and the TUI
  shows the turn, and `thread/resume` answers once it has run.
- `codex app-server --listen unix://PATH` creates PATH as a symlink into
  `/tmp/codex-daemon-<uid>/`, where the socket itself lives. Dialing PATH
  works as before.
- A helper thread appears after a turn as before, answering
  `threadSource: "thread_title"`, `ephemeral: true` and `path: null`; its
  `originator` names the client that last spoke to the app server, so the
  field says nothing about whose thread it is.

### Several TUIs on one app server

Sixteen files record one app server shared by several remote TUIs, captured
2026-09-26 with Codex 0.156.1, cmdman 0.0.26 and crabswarm v0.1.0. They show
what a client needs to address one thread's MCP server out of many.

The app server ran with its `crabswarm-mcp` entry pointed at a wrapper that
logs each spawned server's environment and both halves of its stdio before
running `crabswarm mcp`:

```sh
cmdman run --rm -n cxs-server -w <scratch dir> -- \
  codex app-server --listen unix:///tmp/cxs.sock \
  -c 'mcp_servers.crabswarm-mcp.command="<scratch dir>/mcpwrap.sh"'
cmdman run -t --rm -n cxs-a -w <scratch dir> -- codex --remote unix:///tmp/cxs.sock
cmdman run -t --rm -n cxs-b -w <scratch dir> -- codex --remote unix:///tmp/cxs.sock
```

```sh
#!/bin/sh
d=<log dir>/$$
mkdir -p "$d"
env | sort > "$d/env"
tee -a "$d/in.ndjson" | crabswarm mcp 2>> "$d/stderr" | tee -a "$d/out.ndjson"
```

Each TUI was given the prompt `Call the crabswarm-mcp chat_members tool exactly
once, then reply with the single word done.` in two `send-keys` calls as above.

`codex-mcp-thread-a.stdin.ndjson` and `codex-mcp-thread-b.stdin.ndjson` are the
whole stdin of the two servers the app server spawned, one per TUI thread.
`codex-mcp-thread-b.stdout.ndjson` is the whole stdout of the second one.

`codex-app-server-tool-call.client.ndjson` and
`codex-app-server-tool-call.server.ndjson` are a recorder's `initialize`,
`initialized` and one `mcpServer/tool/call` naming thread A's id, server
`crabswarm-mcp` and tool `chat_members`. The last line of the thread A stdin
fixture is the frame that call produced.

`codex-app-server-thread-config.client.ndjson` and
`codex-app-server-thread-config.server.ndjson` are a recorder's
`thread/start` carrying
`config: {"mcp_servers.crabswarm-mcp.env.PROBE_TAG": "H"}`, with every frame the
server sent until the thread's MCP servers reported `ready`.

`codex-tui-thread-start.client.ndjson` and
`codex-tui-thread-start.server.ndjson` are the `thread/start` a remote TUI sent
and the answer it got. A throwaway WebSocket relay sat between the TUI and the
app server and logged each direction; the two lines were picked out of those
logs by the request's `id`. The rest of the TUI's traffic stays out: its
`thread/list` answers cover every Codex session on the host.

`codex-tui-upgrade-unix.txt` and `codex-tui-upgrade-bearer.txt` are what a
second version of that relay saw of one TUI's WebSocket upgrade: the request
headers as Go's `http.Header.Write` prints them, then, for the unix socket
listener only, the peer credentials read with `SO_PEERCRED` and the
`CMDMAN_CMD_ID` read from that pid's `/proc/<pid>/environ`. The headers are not
the bytes on the wire; the relay printed them from the parsed request.

```sh
cmdman run -t --rm -n cxs-bare -w <scratch dir> -- codex --remote unix:///tmp/cxs-r.sock
cmdman run -t -n cxs-tok -w <scratch dir> -- \
  codex --remote ws://127.0.0.1:47123 --remote-auth-token-env CMDMAN_CMD_ID
```

- Over the unix socket the TUI sends no credential of its own. The peer pid is
  the TUI's, and its environment names the TUI's cmdman command.
- `--remote-auth-token-env` sends `Authorization: Bearer <value>`. The TUI
  refuses it on a unix socket and exits with
  ``ERROR: `--remote-auth-token-env` requires a `wss://` or loopback `ws://` remote.``

`codex-mcp-http-two-threads.jsonl` is every HTTP request a streamable-HTTP MCP
server received from one app server while two remote TUIs each made one tool
call, one JSON object per request: `method`, `path`, `headers` and the raw
`body`. The server was a throwaway Python `ThreadingHTTPServer` answering JSON
(no SSE), declared on the app server with
`-c 'mcp_servers.probe-http.url="http://127.0.0.1:47200/mcp"'` and exposing one
tool, `probe_echo`.

- Each thread opens its own MCP session: its own `initialize` and its own
  `Mcp-Session-Id` from then on.
- No header names the thread. The `user-agent` is `codex-mcp-client/0.156.1`
  on every request. `tools/call` carries `_meta.threadId` as over stdio, so a
  session maps to its thread from its first call.

A TUI can publish its own thread id where a process beside it can read it:

- `-c 'tui.terminal_title=["thread-id"]'` sets the terminal title to the
  thread id, cut by Codex to its first 29 characters plus `...` (the OSC 0
  payload read from `cmdman logs` was `01a0de70-7bb0-7462-ba8f-79cd2...`).
  cmdman keeps the title: `cmdman ls` shows it and `cmdman inspect` holds it as
  `Title`. The title changed to the new thread's id on `/new`.
- `-c 'tui.status_line=["thread-id"]'` prints the full id on the status line,
  which `cmdman capture-screen` reads.
- Both are TUI-side settings, so `[tui]` in `config.toml` works as well as
  `-c` on the TUI's own command line.
- Nothing else runs on the TUI's side. `notify` given to the TUI with `-c`
  never ran. Given to the app server, it ran once per completed turn as a
  child of the app server, with the app server's `CMDMAN_CMD_ID`, and its
  JSON argument named the thread in `thread-id`. It also ran for the helper
  thread Codex starts after a first turn.

`codex-app-server-tool-call-zero-turn.client.ndjson` and
`codex-app-server-tool-call-zero-turn.server.ndjson` are the same kind of
`mcpServer/tool/call`, made against a TUI's thread before that thread had run
any turn, naming the unlisted tool `crabswarm_bind_probe` with
`arguments: {"token": "x"}`. The call reached the thread's `crabswarm mcp`,
whose stdin got `_meta: {"threadId": …}` and the arguments unchanged; the
`-32602 unknown tool` answer is that server's.

`codex-tui-thread-lifecycle.client.ndjson` and
`codex-tui-thread-lifecycle.server.ndjson` are one remote TUI's thread
lifecycle through the relay: every `thread/start`, `thread/resume`,
`thread/fork` and `thread/unsubscribe` request the TUI sent, and the answers to
them, picked out of the relay logs by method and then by request `id`. The TUI
was driven with `send-keys`: one prompt, then `/new`, then `/resume` choosing
the first thread, then `/fork`.

- At startup the TUI sends two `thread/start` requests. The second starts a
  thread answering `threadSource: "thread_title"` and `ephemeral: true`, with
  a `config` that turns off most features and sets
  `mcp_servers: {"crabswarm-mcp": {"enabled": false}}`. The TUI unsubscribes
  from it at once.
- `/new` sends `thread/start`; `/resume` sends `thread/resume` naming the
  chosen thread; `/fork` sends `thread/fork` naming the source thread, and its
  answer is a new thread whose `forkedFromId` names the source.
- Every switch is followed by `thread/unsubscribe` for the thread the TUI
  left.

What the recordings and the probing around them show:

- The app server spawns one MCP server process per loaded thread, when the
  thread starts, before any turn. Every one of them inherits the app server's
  environment, so all of them carry the app server's `CMDMAN_CMD_ID`.
- A `tools/call` from a turn carries the thread in `_meta`: `threadId`,
  `sessionId` and `x-codex-turn-metadata.thread_id` all name it.
- `initialize`, `notifications/initialized` and `tools/list` carry no thread
  id. A server learns its thread from the first `tools/call` it receives.
- `mcpServer/tool/call` on the app server reaches the MCP server of the thread
  it names, with `_meta: {"threadId": …}`. The TUI shows nothing of it. The app
  server forwards a tool name the server never listed; the refusal
  `-32602 unknown tool` in the thread B stdout fixture came from
  `crabswarm mcp`.
- `thread/start` takes `config` overrides per thread, and an
  `mcp_servers.<name>.env.<VAR>` key sets that variable for that thread's
  server alone. The dotted spelling layers over the app server's own `-c`
  overrides. The nested spelling
  `{"mcp_servers": {"crabswarm-mcp": {"env": {…}}}}` also set the variable, but
  the spawned server ran the `config.toml` command and lost the app server's
  `-c` override.
- The remote TUI forwards none of its own `-c` overrides as `config`. Its
  `thread/start` config holds `model_reasoning_effort` and `web_search` only,
  and `cwd` is null unless `-C` is given, so the thread runs in the app
  server's working directory, whatever directory the TUI started in.
- `mcpServer/startupStatus/updated` names the thread and the server, and goes
  from `starting` to `ready` before the `thread/start` answer.
- `/new` in a TUI starts a new thread and a new MCP server; the old thread's
  server keeps running until that thread unloads.
- A stopped TUI's thread unloads about a minute later:
  `thread/status/changed` to `notLoaded`, then `thread/closed`. Its server
  process was gone within about three minutes of the stop.

## OpenCode server with attached TUIs

Four files record one `opencode serve` with TUIs attached to it, captured
2026-09-27 with OpenCode 1.18.32 (run by absolute path under the mise install
root) and cmdman 0.0.26. The model was the free `opencode/big-pickle`, so no
provider credentials were involved.

The scratch directory held an `opencode.json` declaring the logging wrapper from
the Codex section as the local MCP server `probe`, and a server plugin
`probe-plugin.ts`. It also held a `tui.json` naming a TUI plugin
`probe-tui.ts`. Each plugin appended one JSON line per call to its own log, with
`process.pid` and `CMDMAN_CMD_ID`.

```sh
cmdman run --rm -n ocs-server -w <scratch dir> -- opencode serve --port 47400
cmdman run -t --rm -n ocs-t1 -w <scratch dir> -- opencode attach http://127.0.0.1:47400
cmdman run -t --rm -n ocs-t2 -w <scratch dir> -- opencode attach http://127.0.0.1:47400
cmdman run -t --rm -n ocs-t3 -w <scratch dir> -- opencode attach http://127.0.0.1:47400
```

`ocs-t1` and `ocs-t2` each got `Call the probe_chat_members tool exactly once,
then reply with the single word done.` and `ocs-t3` got
`Reply with the single word ok.`, typed with `send-keys` as above. `tui.json`
was written after `ocs-t1` and `ocs-t2` had attached, so only `ocs-t3` loaded
the TUI plugin.

- `opencode-mcp-shared.stdin.ndjson` is the whole stdin of the one MCP server
  the OpenCode server spawned. Both TUIs' sessions called `chat_members` on it.
- `opencode-server-plugin.jsonl` is the whole log of the server plugin.
- `opencode-tui-plugin.jsonl` is the whole log of the TUI plugin in `ocs-t3`:
  the keys of the API object it was handed, every five seconds the value of
  `api.route.current`, and each `session.status` event it received.

What they show:

- The server spawns one MCP server for all attached TUIs, not one per session.
  It inherits the server's environment.
- A `tools/call` from OpenCode carries no session id: its `_meta` holds only
  `progressToken`. The two sessions' calls are told apart only by their
  JSON-RPC ids.
- The server plugin runs in the server process, with the server's
  `CMDMAN_CMD_ID`, and sees every session: `tool.execute.before` names the
  calling session in `sessionID`.
- A server plugin can put the session into an MCP tool call.
  `opencode-mcp-session-arg.stdin.ndjson` is the whole stdin of the MCP server
  in a second run, where the server plugin's `tool.execute.before` set
  `output.args._crabswarm_session = input.sessionID` for tools named
  `probe_*`. Only `ocs-t1` was attached and given the same prompt. Both calls
  the model made arrived with `arguments: {"_crabswarm_session": "ses_…"}`,
  though the tool's schema declares no properties.
- The TUI plugin runs in the attached TUI's process, with that TUI's own
  `CMDMAN_CMD_ID`. `api.route.current` reads `{"name":"home"}` before the first
  prompt and `{"name":"session","params":{"sessionID":"ses_…"}}` after it. The
  plugin also receives `session.status` events through `api.event.on`.

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
