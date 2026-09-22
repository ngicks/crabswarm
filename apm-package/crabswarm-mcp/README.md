# crabswarm-mcp

Wires an agent harness into its `crabswarm chat` room, packaged for
[apm](https://github.com/microsoft/apm): the crabswarm MCP server attends the
room, serves the chat verbs as tools and hands a mention to the harness it is
serving, and hooks deliver teammates' messages mid-turn and again at turn end.
Where a harness has a feed of its own, that feed is also where the member's
state comes from.

The agent-facing half is the
[`crabswarm-mcp`](.apm/skills/crabswarm-mcp/SKILL.md) skill, which teaches the
tools and the etiquette. The hooks below only move messages; the skill is what
makes an agent answer them.

Every hook is a `crabswarm hook exec` invocation, so the package is hook JSON,
an MCP server declaration and a skill — no shell scripts, no `jq`, nothing to
copy alongside the wiring. Everything here assumes `crabswarm` is on `PATH`:
the MCP declaration assumes it when it names `crabswarm` as the server's
command, and every hook command assumes it the same way.

On Claude Code the whole package is a skills-directory plugin: apm copies the
skill directory, and Claude Code reads the hooks and the MCP server out of that
directory on every session. Nothing is merged into `settings.json`. On OpenCode
the same directory carries a plugin file that OpenCode runs in-process; one
line in `opencode.json` names it once, and apm updates the file from then on.
Codex still gets its hooks merged into `hooks.json` and its server written to
`config.toml`, because Codex plugins carry no hooks.

## Install

Add it to the consuming project's `apm.yml`:

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/crabswarm-mcp
```

then

```console
apm install
```

or `apm install -g` to wire every session on the host.

`apm` deploys the package per target:

- **Claude Code** gets the skill directory at `.claude/skills/crabswarm-mcp/`
  (project scope) or `~/.claude/skills/crabswarm-mcp/` (user scope, under
  `CLAUDE_CONFIG_DIR` when that is set), with every file in it. The directory
  carries `.claude-plugin/plugin.json`, so Claude Code loads it as the plugin
  `crabswarm-mcp@skills-dir` and reads `hooks/hooks.json` and `.mcp.json` from
  there. `claude plugin list` shows it as loaded. No hook entry and no server
  entry is merged into `settings.json`, and a hook a later version stops
  declaring disappears with its file on the next `apm install`. apm also
  renders the server declared in `apm.yml` into Claude Code's user-scope
  config; that entry and the plugin's start the same server, and Claude Code
  keeps the higher-precedence one.
- **Codex** gets `.apm/hooks/codex-hooks.json` merged into `.codex/hooks.json`
  with the command strings copied through byte for byte, the server written to
  `.codex/config.toml`, and the skill at `.agents/skills/crabswarm-mcp/`.
- **OpenCode** gets the skill directory at `~/.config/opencode/skills/crabswarm-mcp/`
  at user scope. At project scope apm deploys no OpenCode skill directory;
  OpenCode reads the project's `.agents/skills/crabswarm-mcp/` instead, and
  apm writes the server into the project's `opencode.json`. apm places no
  plugin entry at either scope, so add the deployed file to `opencode.json`
  once, relative to that config file:

  ```json
  { "plugin": ["./skills/crabswarm-mcp/opencode.ts"] }
  ```

  or, in a project's `opencode.json`,
  `"plugin": ["./.agents/skills/crabswarm-mcp/opencode.ts"]`. See
  [OpenCode](#opencode) below for what the plugin does.

Both target directories are created if they are not there yet. Installing twice
changes nothing.

`apm` prints `Hook script not found: .../n/n%s` while it installs. That is its
heuristic scan for a script path to rewrite, tripping over the `\n\n%s` inside
the Codex file's output templates; it rewrites nothing and the commands land
intact.

The `codex-` stem is what keeps the merged file away from Claude Code. apm
calls stem routing deprecated in favor of `targets:` on the consuming
dependency, but that setting would keep the skill and the server away from
Claude Code too, so the stem stays. If a future apm stops honoring the stem,
Claude Code runs every hook twice: once from the plugin and once from
`settings.json`.

Crabswarm's own `apm.yml` does not depend on this package. Installing the chat
hooks into crabswarm's own development sessions is available by adding the
stanza above to the repository root `apm.yml`, and is deliberately not done by
default.

Installing from source is the route that has been exercised. `apm pack` builds
a bundle holding the skill directory with its plugin files, the Codex hook file
renamed to a plain `hooks.json`, and no `apm.yml`. Installing such a bundle has
not been tried; the renamed hook file has lost its `codex-` stem, so a bundle
install would merge the hooks into Claude Code's `settings.json` beside the
plugin's copy.

### Two ways to end up without the server

`apm.yml` declares it as a *self-defined* MCP server — `registry: false`, a
command rather than a registry name — and `apm` trusts one of those on sight
only at depth one, which is what the stanza above makes this package. A project
that reaches it through some other package gets the hooks and the skill, gets no
server, and says so:

```
Transitive package 'crabswarm-mcp' declares self-defined MCP server
'crabswarm-mcp' (registry: false). Re-declare it in your apm.yml or use
--trust-transitive-mcp.
```

Either remedy works, and one of them is needed: without the server nothing
attends the room on its own.

Codex is the harness these two paths leave without one. On Claude Code the
plugin's own `.mcp.json` declares the server as well, so a Claude Code consumer
that received the skill directory has it even when `apm.yml` was never read.

## Layout

```
apm-package/crabswarm-mcp/
├── apm.yml                             package metadata (targets: claude, codex,
│                                       opencode) and the `crabswarm mcp` server
└── .apm/
    ├── hooks/codex-hooks.json          Codex: merged into hooks.json by apm
    └── skills/crabswarm-mcp/           Claude Code: a skills-directory plugin
        ├── SKILL.md
        ├── .claude-plugin/plugin.json
        ├── hooks/hooks.json            delivery, and Claude Code's state reports
        ├── .mcp.json                   the same server as apm.yml declares
        └── opencode.ts                 OpenCode: the same wiring as a plugin
```

The two hook files share the `PostToolUse` read byte for byte, and
`e2e/crabswarm` keeps them so; the `Stop` drain carries the same delivery
wording without the `report-state done` branch Claude Code's needs. They do not
declare the same events either: Codex says what it is doing on the app server
the MCP server is already listening to, so its file wires the delivery and
nothing else, while the plugin's file also carries the state reports Claude Code
has no other way to make. Both harnesses
feed hooks the same snake_case envelope on stdin and read the same camelCase
decision back, which is what lets the commands they do share be shared verbatim.
The file under `.apm/hooks/` is written separately rather than shared because
apm cannot hand one file to Codex's merge and keep it out of Claude Code's: a
file there with no target in its stem is merged into `settings.json` beside the
plugin, and every hook runs twice.

`PermissionRequest` is Codex's approval dialog, and Claude Code implements it
too; only Claude Code's file wires it, since a Codex state report would race the
app server with a slower, coarser answer. `Notification` is Claude Code's alone.

## The MCP server is what attends the room

The declared server is `crabswarm mcp` over stdio. The harness starts it as its
own subprocess. It asks to attend as it starts and then serves the room's verbs
as tools, so the room has the member before its first turn. Chat is the first
family of tools it carries; the command is named for the server rather than for
them, so a family added later needs no change on a consumer's machine.

Attending is not a verb. One open stream is the whole of what attendance is, so
the server *is* the session's place in the room: there is nothing for a hook or
a command line to declare, and nothing to leave with. Stopping the harness ends
it. A harness that installs the hooks and no MCP server therefore never attends
at all — every hook below still runs, and the daemon refuses each of them,
having no member to read for or to report about. An install made before this
server shipped may still carry a join hook of its own; *Stale hooks from older
installs* below says how to find it.

It always attends as an agent, under the name the daemon derives from the
identity token: it is started by a harness and serves nothing else. It also says
which harness that is, read off the name the client gives itself in the MCP
handshake, and how a mention reaches this member — *How a mention reaches the
agent* below is that half.

It keeps attending for the whole session and never gives up. The first retries
come a fifth of a second apart and the wait grows to two seconds. A server whose
daemon is not up yet attends as soon as the daemon binds its socket. A stream
that ends is opened again the same way — a daemon that went away and came back,
a daemon restarted on a fresh database — and an attendance that stood for a
while starts its retries from the short wait rather than inheriting the one a
failing attempt had climbed to. Neither case needs a tool call to prompt it.
Until an attendance lands the server still serves its tools, and each of them
reports why it cannot act.

An attendance can wait on the channel as well as on the daemon. Where a harness
offers a channel the server can ask about beforehand, the server asks before
every attempt and holds the attendance back until the channel answers, rather
than attending with a delivery route that does not work. A member gated that way
is simply absent from `crabswarm chat members`, which is how a launch that went
wrong shows itself.

The chat tools are the member verbs: `chat_send(to, message)`,
`chat_read(cursor, range, to, since, until)` and `chat_members`, each answering
with the text the matching `crabswarm chat` verb prints. The room's attendance
is offered as a resource beside them, and the server announces it as changed
when the roster moves — including after an attendance it had to open again,
since whatever happened while nothing was attending went unannounced.

A person on the host attends another way: `crabswarm chat admin register` puts
them in the room and prints the token they act with from then on, and a plain
shell holding that token in `$CRABSWARM_CHAT_TOKEN` is a member. No stream holds
that attendance, so it lasts until the daemon restarts, and the operator
registers again after one.

### How a mention reaches the agent

A mention wakes its agent one of two ways, and which one is settled per member
at attendance. The server delivers it **natively** when it recognises its
harness *and* the environment carries the variable that harness's channel needs;
otherwise the daemon types a line into the agent's **terminal** through cmdman,
the way every agent was reached before a harness could take a delivery. Both
carry the same notice, which says who wrote and names `chat_read` and
`chat_send`, both from this server — tools rather than a command line, since
every harness the room reaches is served by this server, and some of them decline
to run a command nobody asked them to run. Naming the answer is what keeps it in
the room: a channel arrives with instructions of its own, and one that tells the
model to answer with its own reply tool would carry the answer somewhere nobody
in the room reads. The message itself is never in the notice: it is
sender-controlled content, and a line pushed at a session is the last place to
repeat it.

`crabswarm chat members` prints which one each member got. A line is
`<address> <kind> <state> <harness> <nudge>`, and the last two columns are this:

```
backend/alice  agent  working  claude-code  native
backend/dave   agent  done     other        terminal
frontend/bob   human  done     -            -
```

An agent this server attends for always names a harness; `other` is one whose
client the server did not recognise. A `-` is a column the member declared
nothing for: a person, who runs no harness and is never nudged, or an agent
attending through something other than this server.

The variables are set by whoever launches the session, in the environment the
harness starts the server in, beside the flag that makes the channel exist.
Nothing in an MCP session reports that the flag was passed, so a variable is the
only honest answer the server has:

- **Claude Code** — `CRABSWARM_CLAUDE_CHANNEL=1` and `FAKECHAT_PORT`, with the
  session hosting the official `fakechat` plugin as a channel. Install the
  plugin once, from a Claude Code session:

  ```
  /plugin install fakechat@claude-plugins-official
  ```

  then launch as

  ```console
  FAKECHAT_PORT=<port> CRABSWARM_CLAUDE_CHANNEL=1 claude --channels plugin:fakechat@claude-plugins-official
  ```

  The plugin is the channel and this server is only the sender. Claude Code runs
  the plugin's bun script beside the session, the script listens on
  `127.0.0.1:$FAKECHAT_PORT`, and each notice this server posts to that server's
  `POST /upload` reaches the model as a
  `<channel source="fakechat" chat_id="web" message_id="crabswarm-...">` event
  that starts a turn.

  `CRABSWARM_CLAUDE_CHANNEL=1` is the launcher's opt-in. `FAKECHAT_PORT` is
  fakechat's own variable: the plugin reads it out of the same launch
  environment, which is what puts both ends on one port. Unset and empty mean
  8787 on this side — the plugin's default, and what the `.mcp.json` entry's
  `${FAKECHAT_PORT}` expands to when the launcher named nothing. So a launcher
  either leaves the variable alone or gives it a real port, and never exports an
  empty one: the plugin computes `Number(process.env.FAKECHAT_PORT ?? 8787)`, so
  `""` is port 0 and a listener on whatever port the kernel chose, which nothing
  can address.

  One fakechat per session and one port per session. A second plugin server on a
  port already held exits with `EADDRINUSE` and leaves that session without a
  channel at all, so handing out distinct ports is the launcher's job.

  The launch itself is quiet: no confirmation dialog, the session opens straight
  onto its prompt, and one banner line stays up for the whole session.

  ```
  ▎ Channels (experimental) messages from plugin:fakechat@claude-plugins-official
  ▎ inject directly in this session · restart without --channels to stop
  ```

  `--channels` is variadic, so every argument after it is read as another channel
  entry: a positional prompt has to come *before* the flag. `claude --bg` hosts
  this channel as well as an interactive session does, with the prompt in front
  of the flag the same way. The plugin's start script runs `bun install` on every
  launch, so a first launch wants `bun` on `PATH` and a network.

  This server probes `GET /` on the port before every attendance, and takes the
  plugin's own page as the answer that the channel is there. While nothing
  answers — the plugin failed to start, the port was taken, the flag was
  forgotten after the variable was set — the member does not attend at all: the
  server logs why, retries on the attendance backoff above, and attends as
  `native` the moment fakechat answers. Until then the member is missing from
  `crabswarm chat members` and every chat tool says why it cannot act. A session
  launched with no `CRABSWARM_CLAUDE_CHANNEL=1` is a different case: nothing is
  probed, and it attends as a terminal member the daemon types at.

  fakechat's own server instructions tell the model to answer a channel event
  with fakechat's `reply` tool, which reaches a browser tab nobody in the room is
  watching. Two things keep the answer in the room instead: the notice names
  `chat_send`, and this server's instructions say never to answer a
  `[crabswarm chat]` event with that reply tool.
- **Codex** — `CRABSWARM_CODEX_APP_SERVER=unix:///path.sock`, with the session
  hosted by an app server the TUI is attached to:

  ```console
  codex app-server --listen unix:///path.sock
  codex --remote unix:///path.sock
  ```

  The socket path has to stay short; a long one fails with
  `path must be shorter than SUN_LEN`.
- **OpenCode** — `CRABSWARM_OPENCODE_RELAY`, which the plugin sets itself. It
  opens a loopback listener and puts its address into the environment of the
  server it declares, so an OpenCode running the plugin needs nothing from the
  launcher. See [OpenCode](#opencode).

A session launched without its variable still attends, still serves its tools
and still receives its mentions; only the delivery changes. That fallback has
one limit worth knowing: typed keys cannot see what is already in the harness's
composer, so a notice typed into a session holding an unsent draft may submit
the draft along with it. Native delivery never touches the composer — it pushes a
turn or a prompt of its own — which is the reason the channels exist at all.

A session launched *with* its variable and no working channel is the one case
that does not attend. Claiming a channel and then failing every delivery on it is
worse than never claiming one, so the Claude Code bullet's probe holds the
attendance back instead, and the missing member is the report.

### Where a member's state comes from

The daemon needs to know whether an agent is mid-turn, because a terminal nudge
is only safe while the terminal is waiting for one. Each harness answers that
differently:

| Harness | State comes from | Hooks that report it |
| --- | --- | --- |
| Claude Code | Its hooks as the fast path, corrected by the daemon's screen poller, which reads every attending Claude Code terminal and records what it shows. | `UserPromptSubmit`, `Notification`, `PermissionRequest`, `PostToolUse`, `Stop` |
| Codex | The app server's own feed: thread status changes (idle, active, and the active flags that mean it is waiting on the person) and turn completion, which reads as done however the turn ended. | none |
| OpenCode | The plugin's in-process events, which it reports as it handles them. | none — OpenCode has no hook file |

Claude Code is the one that needs both halves. It offers no state API for an
interactive session, so hooks are the only thing that reports one — and a hook
goes missing the moment a turn is interrupted with Esc, which fires no `Stop`
hook and leaves the member marked working with nothing left to correct it. The
screen still says which of working, waiting on a dialog and back at the prompt
the session is in, so the daemon reads it and lets that reading override the
last report. `chat.screen_poll_interval` in the crabswarm config
(`CRABSWARM_CHAT_SCREEN_POLL_INTERVAL` in the environment, a duration string
there and nanoseconds in the file) sets how often: zero means the default of 3s,
and a negative value turns the polling off and leaves the hooks alone with it.

### The room outlives the sessions in it

A room is made by the first attendance or message in it and stays once every
session in it has ended. The conversation is the room's log, and each role's
read position is kept beside it, so a member coming back under the same role
picks up where that role left off rather than at the room's first message.
`crabswarm chat admin list` shows every room that way, attended or not, and
`crabswarm chat admin log <room>` prints what one of them said without moving
anybody's position.

Nothing prunes a room but the host. `chat.history_limit` in the crabswarm config
caps how many messages each room keeps, pruned as new ones arrive — 0 means the
default of 1000, a negative value caps nothing — and
`crabswarm chat admin delete-room <room>` takes a room away with its messages
and its read positions, refused while anybody is attending it.

An older crabswarm's `chat.db` is not upgraded into this layout, and nothing
says so out loud. The daemon creates only the tables it finds missing, so it
starts, adds the ones this layout introduced, and leaves the old `messages`
table beside them with its old columns — a store every send and every read then
fails against. Remove the old file — `crabswarm config` prints its path —
before the first start.

### Stale hooks from older installs

apm's merge never removes an entry a package stopped declaring. Three upgrades
leave one behind, and each needs a manual cleanup.

An older version of this package installed a `SessionStart` hook running
`crabswarm chat join`. That hook survives an upgrade and runs on every session
start. There is no `join` verb any more — attendance is the MCP server's open
stream — so it fails; the hook discards that failure and exits 0, so nothing
reports it. Search for the string `crabswarm chat join` in:

- `~/.codex/hooks.json` and a project's `.codex/hooks.json`
- `~/.claude/settings.json` and a project's `.claude/settings.json`, under
  `hooks`

Older versions also merged every hook of this package into Claude Code's
`settings.json`. Claude Code now gets them from the plugin, and apm no longer
visits any Claude Code event for this package, so every merged entry stays
where it is and each hook runs twice: once from the plugin, once from
`settings.json`. Delete every entry whose command contains `crabswarm chat`
from `settings.json` at both scopes, and from the `apm-hooks.json` apm keeps
beside each of them. `settings.json` stays untouched by this package from then
on.

This package was `crabswarm-chat` until it grew past chat, and it declared an
MCP server of that name running `crabswarm chat mcp`. That subcommand is gone,
and the old entry outlives the upgrade wherever it was written: Claude Code's
user-scope config, the `[mcp_servers.crabswarm-chat]` table in
`.codex/config.toml`, and `opencode.json`. Delete it — the harness would
otherwise start two servers on one identity token, and the second of them never
attends: the room already has that member, so it spends the session retrying and
warning about a refusal nothing can clear.

### What the harness forwards to the server

A harness may spawn a stdio MCP server with a fixed environment whitelist and
pass nothing else through. Codex does, so the declaration names every variable
the server needs:

```yaml
env_vars:
- CMDMAN_CMD_ID
- CRABSWARM_CHAT_TOKEN
- CRABSWARM_CLAUDE_CHANNEL
- CRABSWARM_CODEX_APP_SERVER
- FAKECHAT_PORT
- XDG_RUNTIME_DIR
```

The first two carry the identity token in its two spellings: the cmdman command
id an agent inherits, and the token `crabswarm chat admin register` prints for a
member registered by hand. A server that resolves neither still serves, and
every tool answers that it has no identity.

The three in the middle are the channel variables *How a mention reaches the
agent* describes. None of them is needed for the server to work, so a missing one
costs a keystroke nudge rather than a refusal — but a variable the launcher set
and the harness did not forward is the same as one nobody set, which is why they
are listed here beside the rest.

`FAKECHAT_PORT` is the exception worth reading twice, because forwarding it wrong
costs more than a keystroke nudge. It belongs to the fakechat plugin, not to this
package: the plugin reads it in the launch environment and this server reads the
forwarded copy, and both ends have to land on the same number. A harness that
drops it leaves this server posting at 8787 while the plugin listens elsewhere.
With 8787 idle the probe catches that as a channel that is not there, and the
member does not attend rather than attending and losing its mentions. With
another session's fakechat sitting on 8787 it does not: a probe recognises the
plugin by the page it serves and cannot tell one session's plugin from another's,
so the notices would land in that session instead. Forwarding the variable is
what makes the port a fact rather than a guess.

`XDG_RUNTIME_DIR` decides where the server looks for the daemon. The socket path
is derived from that variable, and a daemon started from a login shell listens
under it. A server spawned without the variable probes `/run/user/<uid>` and
takes `/run/user/<uid>/crabswarm/default.sock` when that directory is there,
otherwise `/tmp/crabswarm/default.sock`. On an ordinary Linux login the probe
lands on the path the daemon chose. A daemon started with some other
`XDG_RUNTIME_DIR`, in a container or under a test harness, still listens where
the probe never reaches. Forwarding the variable is what covers that case.
Pinning `sock` in `~/.config/crabswarm/config.json` settles the same question
for a harness that forwards nothing.

Codex reads `env_vars` from the server's own `[mcp_servers.crabswarm-mcp]`
table. Claude Code has no such key; the plugin's `.mcp.json` names the same
variables as `env` entries of the form `"${CMDMAN_CMD_ID}"`, which Claude Code
expands from its own environment when it starts the server. All of them but
`CRABSWARM_CODEX_APP_SERVER`: Claude Code never hosts a Codex app server, so the
entry would expand to nothing on every machine, and Codex reads its own list out
of `config.toml` rather than out of `.mcp.json`. `e2e/crabswarm` derives what it
expects in `.mcp.json` from `apm.yml` minus that one, so a variable added to
`apm.yml` alone still fails.

## What each hook does on Claude Code

| Event | Runs | Purpose |
| --- | --- | --- |
| `UserPromptSubmit` | `crabswarm chat report-state working` | A turn began. |
| `Notification` (`permission_prompt`) | `crabswarm chat report-state waiting` | A permission prompt is open. |
| `Notification` (`idle_prompt`) | `crabswarm chat report-state done` | The session has been sitting quiet for about a minute (see below). |
| `PermissionRequest` | `crabswarm chat report-state waiting` | An approval dialog is about to open. |
| `PostToolUse` | `crabswarm chat read --quiet`, then `crabswarm chat report-state working` | Deliver messages that arrived mid-turn as `additionalContext`; the dialog, if there was one, has resolved. |
| `Stop` | `crabswarm chat read --quiet --done-when-empty`, or `report-state done` | Read what arrived; block the stop when the read handed something over, otherwise report done. |

Every state report here is Claude Code's alone — the other two harnesses say
what they are doing on a feed of their own, and a hook reporting alongside one
would race it. What the daemon does with these reports, and what corrects them
when one goes missing, is *Where a member's state comes from* above.

The second `PostToolUse` entry is how a member gets out of `waiting` again.
Claude Code announces neither a dialog being answered nor one dismissed, so the
next tool call completing is the only signal that the approval the member was
waiting on resolved.

### How `hook exec` shapes the decision

`crabswarm hook exec` takes two Go templates: a command template it renders and
executes, and an output template that turns the result into the hook's JSON.
The output template speaks only through functions — `context`, `blockDecision` —
each of which records a field and renders as the empty string.

That gives this package its two idioms.

**A template that records nothing is a plain allow**, whatever the command did.
The fire-and-forget hooks pass `'{{/* records nothing ... */}}'` as their output
template, so a `report-state` against a daemon nobody is running writes no JSON,
prints nothing and exits 0.

The comment matters: an *empty* second argument is indistinguishable from an
omitted one, and an omitted output template selects `hook exec`'s built-in
behavior — block the event with the captured output as the reason. Passing `''`
would turn every unreachable daemon into a blocked turn. A template that is
non-empty but renders nothing is what says "run this, and never mind how it
went".

**The delivering hooks branch on `.Stdout`, not `.Output`.** `.Output` is the
combined capture, so a daemon that is not running would put the CLI's
`chat daemon unreachable ... hint: start it by running crabswarm serve` into
the agent's context as if a teammate had sent it. `{{if and .Success .Stdout}}`
fires only when a read succeeded and printed something.

### Reading is the delivery, not a notification

`crabswarm chat read` **moves the member's read position** to the newest message
it showed, so what one read hands over the next one does not offer again. That
is why the PostToolUse hook injects the messages it read rather than a "you have
mail" notice — the injection *is* the delivery. The alternative, a hint that
made the agent read separately, would cost the same round trip and leave the
messages unseen in the meantime.

The same fact shapes the Stop hook: when `stop_hook_active` is set, an earlier
Stop hook already blocked this turn, so the command template renders the
`report-state done` branch and does **not** read at all. A second read would
either loop the agent or, once the harness stops honoring the block, carry the
position past messages it never displayed. Those messages are still in the room
— the log outlives every read, and `--cursor tail` reprints them — but nothing
would be left to tell the agent to look.

Both hooks emit the minimum JSON their event allows — `decision`/`reason` for
Stop, `hookSpecificOutput.additionalContext` for PostToolUse — and nothing
else. Codex rejects an output object carrying fields its schema does not know,
and a rejected output *after* a read has already moved the position is exactly
the undelivered message these hooks exist to prevent. The output builder emits
only the fields the template's functions set, so that holds as long as no other
function is called.

### Why the Stop hook needs two flags on one read

One `hook exec` invocation runs one command, and the Stop hook has two things
to do that must not come apart: deliver whatever the read found, and report the
member done exactly when the turn is really ending. `crabswarm chat read` takes
both as flags so a single process decides:

- `--quiet` drops the line an empty read prints, so the hook can tell messages
  from none by whether the output is empty at all. Without it the test would be
  a string comparison against that sentence, which makes a wording nobody thinks
  of as an interface into one.
- `--done-when-empty` reports the member done when the read handed nothing
  over. Done is what re-arms the daemon's terminal nudge, and it is wrong
  exactly when the hook is about to block — the turn continues, so the member
  is still working.

Wiring the done report as a second hook entry on `Stop` would race: hooks on
one event run concurrently, so the report could land while the delivering path
is still deciding, and mark a continuing turn done. Keeping both on the read
also keeps them honest about failure — a read that could not reach the daemon
reports nothing, because the daemon that would hear the report is the one that
did not answer.

`crabswarm chat read` with no flags is unchanged: a read that found nothing
still says `no pending messages`, which is what a human wants to see.

### Graceful degradation

Messages are the room's log and outlive every failure here, so nearly every
path costs a late delivery and never a message:

- Every fire-and-forget hook records nothing in its output template, so a
  daemon that is not running, or a session with no identity token, never breaks
  a turn. The command's own stderr is captured by `hook exec` and never reaches
  the transcript.
- A failed `chat read` produces no `.Stdout`, so nothing is injected: nothing
  was handed over, and the read position did not move.
- Every hook entry carries a `timeout`. The PostToolUse hook runs after every
  tool call, and a wedged daemon must not stall the session.
- Hooks are independent. One that breaks costs only its own contribution —
  the next hook that runs reads what it did not.

Two paths are now louder than they used to be, and both are outside what a
hook can quiet down:

- An envelope `hook exec` cannot parse is a plain error, so the hook exits 1
  with `parsing hook input: ...` on stderr instead of quietly reading nothing.
  A harness that writes unparseable JSON to a hook's stdin is broken in a way
  worth hearing about.
- No `crabswarm` on `PATH` at all is the shell's `command not found` and exit
  127, where the shell scripts used to swallow it. That is the assumption at
  the top of this file failing, which is worth saying out loud too.

Neither is a block: a decision rides on stdout and there is none, and only exit
2 blocks an event. So the worst either costs is a noisy line and a late
delivery.

The last line of defence is outside this package: a mention reaches an agent
that has stopped working whether or not a hook ever ran, either through its
harness's own channel or as a line typed into its terminal
(`[crabswarm chat] new message from ... — read it with the chat_read tool and respond with chat_send, both from crabswarm-mcp`).

### The idle notification is how an interrupted turn heals

Claude Code's `Notification` hook fires for a permission prompt and again for
the idle prompt roughly a minute after Claude stops responding, and its matcher
selects on the event's `notification_type`. The two are wired apart because they
mean opposite things: `permission_prompt` is a member blocked on a dialog,
`idle_prompt` is a member with nothing left to do.

The idle branch is one of the two ways back for a turn the user interrupted.
`Stop` hooks do not run when a session is cancelled with ESC, so the member
keeps whatever the last report left it in — `working` after a tool call,
`waiting` if a dialog was open — and the daemon nudges a **done** member on
sight, while a `working` or `waiting` one is left alone as busy. Reporting
`done` on the idle prompt clears that about a minute later, without the agent
doing anything.

The other way back is the daemon's screen poller, which reads the terminal
directly and does not wait for any hook at all; a host that has turned the
polling off with a negative `chat.screen_poll_interval` is left with this branch
alone. Failing both, a report older than ten minutes has stopped describing the
terminal and the member is nudged anyway, with the screen snapshot the injection
takes still standing between the nudge and a terminal that turned out to be
busy.

Every other notification type — `elicitation_complete`, `elicitation_response`,
`auth_success` — reports nothing rather than `waiting`. That is deliberate:
`PermissionRequest` covers the dialog case on its own, and a catch-all group
would also match `idle_prompt` and race a `waiting` report against the `done`
one, which is the wedge this split exists to remove.

## Codex

**The hook half is best-effort and unverified.** Codex's hook surface is recent;
the wiring was written against the Codex source (its `hooks.json` loader,
`HookEventsToml`/`MatcherGroup`/`HookHandlerConfig` shapes, and the
Stop/PostToolUse stdin and output schemas) but **has not been run against a
Codex session**. Treat every claim about the two hooks below as one to check.
Each hook is independent and failure-tolerant, so a hook Codex silently ignores
degrades to late delivery.

The app-server half stands on firmer ground: the exchange this package's client
speaks was recorded off a real `codex app-server` and a real TUI, and
`e2e/crabswarm/testdata/harness` keeps those frames.

What has been checked from this side is that `crabswarm hook exec` speaks
Codex's half of the envelope surface: a `PermissionRequest` envelope parses into
its typed variant and the hook exits 0 silently, which is what
`e2e/crabswarm/chat_hooks_test.go` runs the shipped commands to prove. An event
neither harness declares would parse too and simply render nothing.

Codex gets `codex-hooks.json`, which wires two events and no more: the
`PostToolUse` read and the `Stop` drain. Everything else the plugin's
`hooks.json` declares is a state report, and Codex says what it is doing on the
app server this package's MCP server is already listening to — a hook reporting
beside that feed would race it with a slower, coarser answer. `apm` does not
translate hook event names for Codex; it merges whatever events the file
declares into `.codex/hooks.json` verbatim, unlike the Gemini target, which
renames events on the way out.

Nothing here needs a rewritten path. The commands name `crabswarm` and nothing
else, so the file installs exactly as written and does not care what directory
Codex runs its hooks from.

Codex loads hooks as untrusted until you say otherwise — expect to approve them
before they run.

The MCP server is the same story one layer over: `apm install` writes an
`[mcp_servers.crabswarm-mcp]` table into `.codex/config.toml` naming
`crabswarm` with `["mcp"]`, which is the shape a Codex install was observed
to produce; whether Codex then starts the MCP server and it attends the room has
not been run against a Codex session either. The table also carries the
`env_vars` list above. Without that list Codex hands the MCP server an
environment holding neither an identity token nor the runtime dir the socket
path comes from. Without it Codex attends nothing at all: attendance is that
open stream, and there is no command an operator could type to declare one.

What Codex ends up running:

| Event | Runs | Purpose |
| --- | --- | --- |
| `PostToolUse` | `chat read --quiet` | Deliver mid-turn messages. |
| `Stop` | `chat read --quiet --done-when-empty` | Read what arrived and block the stop when it handed something over. |

The `PostToolUse` read is the plugin's own command byte for byte, and
`chat_codex_test.go` keeps it so, since a message announced two different ways
on two harnesses is a skill teaching the wrong words. The `Stop` drain carries
that same wording with the `report-state done` branch left off — Codex's
`stop_hook_active` case reads nothing and reports nothing. It does still pass
`--done-when-empty`, which reports the member done when the read hands nothing
over; the app-server feed says the same thing a moment later.

The state Codex does not report through hooks it reports on the app server the
session is hosted by, which is the same socket a mention is delivered over.
`CRABSWARM_CODEX_APP_SERVER` is what points the MCP server at it; a Codex
started without an app server has no feed and no channel, and its member is
woken through its terminal like any other. *Where a member's state comes from*
above says what the feed carries.

Codex's `notify` program (`agent-turn-complete`) could report `done`
redundantly, and nothing here uses it — the only thing this package puts in
`config.toml` is this server's `[mcp_servers]` table.

## OpenCode

OpenCode has no hook file. It loads JavaScript or TypeScript plugins and hands
them session, permission and tool events, so `opencode.ts` in the skill
directory is this package's hook wiring for OpenCode: each handler runs one
`crabswarm chat` verb, quietly, and ignores failure the way the hooks do. The
delivery wording is the hook file's, and `e2e/crabswarm` keeps it so.

It carries one thing no hook file needs: the relay a mention arrives on. The
plugin is the only part of the install that knows which session the person is
driving, so the MCP server posts the notice to a loopback listener the plugin
opens, and the plugin prompts that session with it. A prompt of its own rather
than the composer, for the reason *How a mention reaches the agent* gives: a
notice appended to the composer and submitted would submit the person's unsent
draft with it. A plugin that cannot open the listener declares no relay, and the
member is a terminal one.

apm does not register plugins with OpenCode, so the operator names the deployed
file once in `opencode.json`, relative to that config file:

```json
{ "plugin": ["./skills/crabswarm-mcp/opencode.ts"] }
```

OpenCode installs the plugin's dependency package from npm the first time it
starts with a plugin configured, which makes that first start slower.

The plugin declares the server itself, through the `config` hook, as a local
MCP server running `crabswarm mcp` with `CMDMAN_CMD_ID`,
`CRABSWARM_CHAT_TOKEN` and `XDG_RUNTIME_DIR` set when the session has them.
OpenCode hands a local server its whole environment anyway, so the three are
there either way; naming them keeps the declaration the same shape everywhere.
It also sets `CRABSWARM_OPENCODE_RELAY` to the loopback address it listens on,
and that is the whole of what makes an OpenCode member one the daemon does not
type at.

**A `crabswarm-mcp` server already in the config wins**, whether the operator
wrote it by hand or apm did at project scope — and a server that won that way
never receives the relay's address, since only the declaration the plugin builds
carries it. Such a member attends, serves its tools and reports its state
normally; it is simply a terminal one. Leaving the server out of `opencode.json`
and letting the plugin declare it is what keeps the relay. apm's own entry
carries the `env_vars` key Codex reads, which OpenCode accepts and ignores.

A headless `opencode serve` starts its MCP servers when something first asks for
them; whether the TUI connects them at startup or at the first tool use has not
been watched, and it decides whether the member attends before or during its
first turn.

What the plugin runs:

| OpenCode event | Runs | Purpose |
| --- | --- | --- |
| `chat.message` | `report-state working` | A turn began. |
| `permission.ask` | `report-state waiting` | An approval dialog is open. |
| `permission.replied` | `report-state working` | The dialog resolved. |
| `tool.execute.after` | `chat read --quiet`, then `report-state working` | Deliver mid-turn messages appended to the tool's result. |
| `session.idle` | `chat read --quiet --done-when-empty` | Report done, or hand the messages found to the session as its next prompt. |

What the read found at idle becomes a new user message in the session rather
than a blocked stop, because OpenCode has no stop to block; the session takes
one more turn over those messages and reports done when the next idle read finds
nothing. The read moves the position before the prompt is sent, so a prompt
OpenCode refuses costs that delivery — the same trade the Stop read makes on the
other harnesses. Only the session a person drives reports: a subagent's session
carries a parent, and its events are ignored.

OpenCode reads skills from `~/.config/opencode/skills`, `~/.agents/skills` and
`~/.claude/skills`, and apm deploys this package's skill to each of those it
targets, so the skill can be listed more than once there.

## Verifying a change

The hook files parse with `jq`, and `e2e/crabswarm/chat_hooks_test.go` drives
the wiring end to end from Go: it reads the plugin's `hooks/hooks.json`, pulls
each `command` string out, and runs it **verbatim** through a shell with sample
hook envelopes on stdin, the real `crabswarm` binary on `PATH` and a real daemon
behind it. `chat_codex_test.go` pins the Codex file to the delivering commands
and to nothing that reports state, and `mcp_package_test.go` pins the plugin's
`.mcp.json` to the same server `apm.yml` declares and runs that command line
against the built binary. `chat_opencode_test.go` runs the `opencode` on `PATH`,
when there is one, against a daemon and a mock model, and watches the plugin
attend, report and deliver; without `opencode` the case is skipped. So
what the suite exercises is the wiring that ships, not a Go paraphrase of it —
including that every command stays silent and exits 0 when no daemon is
running, that the delivering hooks move the read position exactly when they hand
something over, and that no command reaches back out to a script, to `jq`, to
`${CLAUDE_PLUGIN_ROOT}` or to the wording an empty read prints.

The Claude Code channel is covered the same way, in
`e2e/crabswarm/chat_fakechat_test.go`, against a Go fake of fakechat's inbound
surface rather than against the real plugin, which wants a Claude Code session
and a network to install.

The flags the Stop hook depends on are covered a level down, in
`crabswarm/chat/cli/member_test.go`: the done report a read makes when it hands
nothing over is not visible from outside the daemon, so it is asserted against
the RPC there.
