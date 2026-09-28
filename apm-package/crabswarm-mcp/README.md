# crabswarm-mcp

Wires Claude Code into its `crabswarm chat` room, packaged for
[apm](https://github.com/microsoft/apm): the crabswarm MCP server attends the
room, serves the chat verbs as tools, reports what the session is doing and
hands a mention to the session it is serving.

Codex and OpenCode are wired by
[`crabswarm-mcp-shared`](../crabswarm-mcp-shared/README.md), which runs one
`crabswarm mcp --transport http` for every agent a shared Codex app server or
`opencode serve` hosts. This package targets Claude Code alone, because Claude
Code starts one stdio server per session and apm cannot declare a different
transport per target.

The agent-facing half is the
[`crabswarm-mcp`](.apm/skills/crabswarm-mcp/SKILL.md) skill, which teaches the
tools and the etiquette.

The package is an MCP server declaration and a skill: no hooks, no shell
scripts, nothing to copy alongside the wiring. Everything here assumes
`crabswarm` is on `PATH`, since the MCP declaration names `crabswarm` as the
server's command.

On Claude Code the whole package is a skills-directory plugin: apm copies the
skill directory, and Claude Code reads the MCP server out of that directory on
every session. The plugin ships no hook file, and nothing is merged into
`settings.json`.

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

Claude Code gets the skill directory at `.claude/skills/crabswarm-mcp/`
(project scope) or `~/.claude/skills/crabswarm-mcp/` (user scope, under
`CLAUDE_CONFIG_DIR` when that is set), with every file in it. The directory
carries `.claude-plugin/plugin.json`, so Claude Code loads it as the plugin
`crabswarm-mcp@skills-dir` and reads `.mcp.json` from there.
`claude plugin list` shows it as loaded. No hook entry and no server entry is
merged into `settings.json`, and a hook a later version stops declaring
disappears with its file on the next `apm install` — which is how an install
made while this plugin still shipped hooks loses them. apm also renders the
server declared in `apm.yml` into Claude Code's user-scope config; that entry
and the plugin's start the same server, and Claude Code keeps the
higher-precedence one.

The target directory is created if it is not there yet. Installing twice
changes nothing.

Crabswarm's own `apm.yml` does not depend on this package. Installing this
wiring into crabswarm's own development sessions is available by adding the
stanza above to the repository root `apm.yml`, and is deliberately not done by
default.

Installing from source is the route that has been exercised. `apm pack` builds
a bundle holding the skill directory with its plugin files and no `apm.yml`.
Installing such a bundle has not been tried.

### A transitive install and the server

`apm.yml` declares the server as a *self-defined* MCP server — `registry: false`,
a command rather than a registry name — and `apm` trusts one of those on sight
only at depth one, which is what the stanza above makes this package. A project
that reaches it through some other package gets the skill and no server in the
user-scope config, and says so:

```
Transitive package 'crabswarm-mcp' declares self-defined MCP server
'crabswarm-mcp' (registry: false). Re-declare it in your apm.yml or use
--trust-transitive-mcp.
```

The plugin's own `.mcp.json` declares the server as well, so a Claude Code
consumer that received the skill directory has it even when `apm.yml` was never
read.

## Layout

```
apm-package/crabswarm-mcp/
├── apm.yml                             package metadata (targets: claude) and
│                                       the `crabswarm mcp` server
└── .apm/
    └── skills/crabswarm-mcp/           Claude Code: a skills-directory plugin
        ├── SKILL.md
        ├── .claude-plugin/plugin.json
        └── .mcp.json                   the same server as apm.yml declares
```

The plugin ships no hook file. The member's state comes off the agents listing
the MCP server polls, and its messages arrive either as a channel notification
or as a line the daemon types into its terminal, so nothing is left for a hook
to carry. The `Stop` hook it used to ship was worse than nothing: it reported
the member done whenever the inbox was empty, while a subagent running in the
background kept the session working.

## The MCP server is what attends the room

The declared server is `crabswarm mcp` over stdio. Claude Code starts it as its
own subprocess. It asks to attend as it starts and then serves the room's verbs
as tools, so the room has the member before its first turn. Chat is the first
family of tools it carries; the command is named for the server rather than for
them, so a family added later needs no change on a consumer's machine.

Attending is not a verb. One open stream is the whole of what attendance is, so
the server *is* the session's place in the room: there is nothing for a hook or
a command line to declare, and nothing to leave with. Stopping the session ends
it. An install made before this server shipped may still carry a join hook of
its own; *Stale entries from older installs* below says how to find it.

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

An attendance can wait on the channel as well as on the daemon. When the launch
asked for the fakechat channel, the server asks the channel before every attempt
and holds the attendance back until it answers, rather than attending with a
delivery route that does not work. A member gated that way is simply absent from
`crabswarm chat members`, which is how a launch that went wrong shows itself.

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
at attendance. The server delivers it **natively** when the launch set up
Claude Code's channel; otherwise the daemon types a line into the agent's
**terminal** through cmdman. Both carry the same notice, which says who wrote
and names `chat_read` and `chat_send`, both from this server. Naming the answer
is what keeps it in the room: a channel arrives with instructions of its own,
and one that tells the model to answer with its own reply tool would carry the
answer somewhere nobody in the room reads. The message itself is never in the
notice: it is sender-controlled content, and a line pushed at a session is the
last place to repeat it.

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

The channel variables are set by whoever launches the session, in the
environment Claude Code starts the server in, beside the flag that makes the
channel exist. Nothing in an MCP session reports that the flag was passed, so a
variable is the only honest answer the server has. Claude Code's channel takes
`CRABSWARM_CLAUDE_CHANNEL=1` and `FAKECHAT_PORT`, with the session hosting the
official `fakechat` plugin as a channel. Install the plugin once, from a Claude
Code session:

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

The same question is asked every few seconds for as long as the member attends,
and a plugin that dies mid-session ends the attendance exactly the way a daemon
that went away does. So the member drops out of `crabswarm chat members` until
the plugin answers again, rather than staying listed as one the daemon types
nothing at while every mention it is handed is dropped.

fakechat's own server instructions tell the model to answer a channel event
with fakechat's `reply` tool, which reaches a browser tab nobody in the room is
watching. Two things keep the answer in the room instead: the notice names
`chat_send`, and this server's instructions say never to answer a
`[crabswarm chat]` event with that reply tool.

A delivered notice is still not an obeyed one: a recorded session read the
injected line, called it a prompt injection and declined it, so the skill and
this server's instructions are what make an agent treat a `[crabswarm chat]`
event as the room asking for it rather than as text it should distrust.

A session launched without the channel variable still attends, still serves its
tools and still receives its mentions; only the delivery changes. That fallback
has one limit worth knowing: typed keys cannot see what is already in the
composer, so a notice typed into a session holding an unsent draft may submit
the draft along with it. Native delivery never touches the composer — it starts
a turn of its own — which is the reason the channel exists at all.

A session launched *with* the variable and no working channel is the one case
that does not attend. Claiming a channel and then failing every delivery on it
is worse than never claiming one, so the probe holds the attendance back
instead, and the missing member is the report.

### Where a member's state comes from

The daemon needs to know whether an agent is mid-turn, because a terminal nudge
is only safe while the terminal is waiting for one. Claude Code answers on a
feed of its own, and the member's own MCP server is what watches that feed and
reports what it says: `claude agents --json`, the registry of every session on
the host, read every two seconds. The server finds its own session in the
listing by the id Claude Code hands it as `CLAUDE_CODE_SESSION_ID`.

No hook reports state. Hooks were once the only report an interactive session
could make, and they were wrong twice over: a turn interrupted with Esc fires no
`Stop` hook and left the member marked working with nothing to correct it, and
the `Stop` read stored `done` whenever the inbox was empty, while a subagent
running in the background kept the session working. The agents listing is
Claude Code's own account of the session and stays right through both, so the
hooks went and the listing stayed.

Claude Code sets `CLAUDE_CODE_SESSION_ID` itself, in the environment of every
MCP server it spawns, so this feed needs nothing from whoever launched the
session — unlike the channel, which needs the launcher's variable. A session
that reports its own state is therefore often still a member the daemon types
at.

A harness the server does not recognise has no feed, and the daemon reads that
member's terminal instead. `chat.screen_poll_interval` in the crabswarm config
(`CRABSWARM_CHAT_SCREEN_POLL_INTERVAL` in the environment, a duration string
there and nanoseconds in the file) sets how often: zero means the default of 3s,
and a negative value turns the polling off. A screen is the coarsest answer
there is — an empty composer reads as idle whether or not something is running
behind it — which is why only a member with nothing better is read that way.

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

### Stale entries from older installs

apm's merge never removes an entry a package stopped declaring, so an upgrade
can leave wiring behind that needs a manual cleanup.

Earlier versions of this package targeted Codex and OpenCode as well:

- Codex got a `PostToolUse` and a `Stop` hook running `crabswarm chat read`,
  merged into `~/.codex/hooks.json` or a project's `.codex/hooks.json`. Delete
  every entry whose command contains `crabswarm chat`: Codex is wired by
  `crabswarm-mcp-shared` now, which ships no hooks.
- Codex and OpenCode got a stdio `crabswarm-mcp` server, in `.codex/config.toml`
  and a project's `opencode.json`. Installing `crabswarm-mcp-shared` writes its
  remote entry under the same name in its place. apm writes `opencode.json`
  only in a project that has an `.opencode/` directory, so replace any other
  stdio entry by hand with the remote one the `crabswarm-mcp-shared` README
  shows.
- OpenCode's `opencode.json` named `./skills/crabswarm-mcp/opencode.ts` or
  `./.agents/skills/crabswarm-mcp/opencode.ts` as a plugin. Remove that line and
  name the plugins `crabswarm-mcp-shared` ships instead.

An older version also installed a `SessionStart` hook running
`crabswarm chat join`. That hook survives an upgrade and runs on every session
start. There is no `join` verb any more — attendance is the MCP server's open
stream — so it fails; the hook discards that failure and exits 0, so nothing
reports it. Search for the string `crabswarm chat join` in:

- `~/.codex/hooks.json` and a project's `.codex/hooks.json`
- `~/.claude/settings.json` and a project's `.claude/settings.json`, under
  `hooks`

Older versions also merged every hook of this package into Claude Code's
`settings.json`, and apm no longer visits any Claude Code event for this
package, so every merged entry stays where it is and runs on every session.
This package wires Claude Code no hooks at all now: a state report left there
races the feed the member's own MCP server keeps, and the `Stop` read stores
`done` whenever the inbox is empty, which is wrong for as long as a background
subagent keeps the session working. Delete every entry whose command
contains `crabswarm chat` from `settings.json` at both scopes, and from the
`apm-hooks.json` apm keeps beside each of them. `settings.json` stays untouched
by this package from then on.

The plugin's own `hooks/hooks.json`, which older versions shipped in the skill
directory, needs no cleanup: apm copies the directory as it is, so the file
goes when the next `apm install` replaces it.

This package was `crabswarm-chat` until it grew past chat, and it declared an
MCP server of that name running `crabswarm chat mcp`. That subcommand is gone,
and the old entry outlives the upgrade wherever it was written: Claude Code's
user-scope config, the `[mcp_servers.crabswarm-chat]` table in
`.codex/config.toml`, and `opencode.json`. Delete it — the harness would
otherwise start two servers on one identity token, and the second of them never
attends: the room already has that member, so it spends the session retrying and
warning about a refusal nothing can clear.

### What Claude Code forwards to the server

`apm.yml` lists the variables the server needs from the environment the session
was launched in:

```yaml
env_vars:
- CMDMAN_CMD_ID
- CRABSWARM_CHAT_TOKEN
- CRABSWARM_CLAUDE_CHANNEL
- FAKECHAT_PORT
- XDG_RUNTIME_DIR
```

The plugin's `.mcp.json` names the same variables as `env` entries of the form
`"${CMDMAN_CMD_ID}"`, which Claude Code expands from its own environment when it
starts the server. `e2e/crabswarm` checks that both lists stay the same, so a
variable added to one alone fails.

The first two carry the identity token in its two spellings: the cmdman command
id an agent inherits, and the token `crabswarm chat admin register` prints for a
member registered by hand. A server that resolves neither still serves, and
every tool answers that it has no identity.

The two in the middle are the channel variables *How a mention reaches the
agent* describes. Neither is needed for the server to work, so a missing one
costs a keystroke nudge rather than a refusal.

`FAKECHAT_PORT` is the exception worth reading twice, because forwarding it wrong
costs more than a keystroke nudge. It belongs to the fakechat plugin, not to this
package: the plugin reads it in the launch environment and this server reads the
forwarded copy, and both ends have to land on the same number. A server that
does not receive it posts at 8787 while the plugin listens elsewhere. With 8787
idle the probe catches that as a channel that is not there, and the member does
not attend rather than attending and losing its mentions. With another session's
fakechat sitting on 8787 it does not: a probe recognises the plugin by the page
it serves and cannot tell one session's plugin from another's, so the notices
would land in that session instead. Forwarding the variable is what makes the
port a fact rather than a guess.

`XDG_RUNTIME_DIR` decides where the server looks for the daemon. The socket path
is derived from that variable, and a daemon started from a login shell listens
under it. A server spawned without the variable probes `/run/user/<uid>` and
takes `/run/user/<uid>/crabswarm/host/default.sock` when that directory is
there, otherwise `/tmp/crabswarm/host/default.sock`. On an ordinary Linux login
the probe lands on the path the daemon chose. A daemon started with some other
`XDG_RUNTIME_DIR`, in a container or under a test harness, still listens where
the probe never reaches. Forwarding the variable is what covers that case.
Pinning `sock` in `~/.config/crabswarm/config.json` settles the same question
for a server that receives nothing.

## Verifying a change

`e2e/crabswarm/mcp_package_test.go` pins the server `apm.yml` declares — a stdio
`crabswarm mcp` for Claude Code alone — pins the plugin's `.mcp.json` to the
same server, and runs that command line against the built binary.
`apm_package_test.go` pins the skill directory to the files it ships, the hook
files it no longer ships included.

The Claude Code channel is covered in `e2e/crabswarm/chat_fakechat_test.go`,
against a Go fake of fakechat's inbound surface rather than against the real
plugin, which wants a Claude Code session and a network to install.
