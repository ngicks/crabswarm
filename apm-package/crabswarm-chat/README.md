# crabswarm-chat

Wires an agent harness into its `crabswarm chat` room, packaged for
[apm](https://github.com/microsoft/apm): an MCP bridge attends the room and
serves the chat verbs as tools, hooks deliver teammates' messages mid-turn and
again at turn end, and those same hooks report what the harness is doing so the
daemon knows when a terminal nudge is safe.

The agent-facing half is the
[`crabswarm-chat`](.apm/skills/crabswarm-chat/SKILL.md) skill, which teaches the
room's verbs and etiquette. The hooks below only move messages; the skill is
what makes an agent answer them.

Every hook is a `crabswarm hook exec` invocation, so the package is one JSON
file, one MCP server declaration and a skill — no shell scripts, no `jq`,
nothing to copy alongside the wiring. Everything here assumes `crabswarm` is on
`PATH`: the skill assumes it when it tells an agent to type
`crabswarm chat read`, and the MCP declaration assumes it when it names
`crabswarm` as the server's command.

## Install

Add it to the consuming project's `apm.yml`:

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/crabswarm-chat
```

then

```console
apm install
```

`apm` compiles the package per target: the hooks merge into
`.claude/settings.json` (Claude Code) and `.codex/hooks.json` (Codex) with the
command strings copied through byte for byte, the MCP server is written to the
project's `.mcp.json` (Claude Code) and `.codex/config.toml` (Codex), and the
skill materializes at `.claude/skills/crabswarm-chat/` and
`.agents/skills/crabswarm-chat/`. Both target directories are created if they
are not there yet. Installing twice changes nothing.

`apm` prints `Hook script not found: .../n/n%s` while it installs. That is its
heuristic scan for a script path to rewrite, tripping over the `\n\n%s` inside
the output templates; it rewrites nothing and the commands land intact.

Crabswarm's own `apm.yml` does not depend on this package. Installing the chat
hooks into crabswarm's own development sessions is available by adding the
stanza above to the repository root `apm.yml`, and is deliberately not done by
default.

Installing from source is the route that has been exercised. `apm pack` builds a
plugin bundle, which used to be the wrong shape for this package outright: back
when the wiring shipped as two files it skipped the root-level `hooks/`
directory whenever `.apm/` was present and handed every consumer — Claude Code
included — the Codex-only file as the bundle's one `hooks.json`. With a single
universal hook file there is nothing left to pick wrong: a bundle carries the
same wiring an install from source produces. `apm pack --target` is deprecated
and recorded as metadata only, so `--target claude` and `--target codex` produce
the same bundle, which is now the right answer rather than a hazard.

### Two ways to end up without the bridge

`apm.yml` declares the bridge as a *self-defined* MCP server — `registry:
false`, a command rather than a registry name — and `apm` trusts one of those on
sight only at depth one, which is what the stanza above makes this package. A
project that reaches it through some other package gets the hooks and the skill,
gets no server, and says so:

```
Transitive package 'crabswarm-chat' declares self-defined MCP server
'crabswarm-chat' (registry: false). Re-declare it in your apm.yml or use
--trust-transitive-mcp.
```

Either remedy works, and one of them is needed: without the bridge nothing joins
the room on its own.

A packed bundle loses the declaration outright. `apm pack` writes `plugin.json`,
`hooks.json` and `skills/` and nothing else — no `apm.yml`, which is where the
server is declared — so installing a bundle configures no server, whatever else
it places. Installing from source is the route that brings all three.

## Layout

```
apm-package/crabswarm-chat/
├── apm.yml                             package metadata (targets: claude, codex)
│                                       and the `crabswarm chat mcp` server
├── .claude-plugin/plugin.json          Claude Code plugin manifest
└── .apm/
    ├── hooks/report-state.json         hook wiring for every target
    └── skills/crabswarm-chat/SKILL.md
```

One file wires every target. Its stem carries no target token, so `apm` hands
the same events to Claude Code and to Codex, and what the file declares is the
union of what the two harnesses announce — each keeps the events it knows and
drops the rest. Both feed hooks the same snake_case envelope on stdin and read
the same camelCase decision back, so the commands themselves are shared
verbatim.

Only `Notification` and `PermissionRequest` are not common ground.
`PermissionRequest` is Codex's approval dialog, and Claude Code implements it
too, so it runs on both. `Notification` is Claude Code's alone; Codex parses
`hooks.json` into a struct of the events it knows and ignores every other key,
so the block lands in `.codex/hooks.json` and never becomes a hook there.

## The bridge is what attends the room

The declared server is `crabswarm chat mcp` over stdio, which the harness starts
as its own subprocess. It asks to attend while it is starting up — a few tries
with a growing backoff, which covers a daemon still binding its socket — and
then serves the room's verbs as tools. So a member has an inbox before its first
turn, and no hook is involved in getting one.

That is the only automatic join this package ships. A `SessionStart` hook used
to run `crabswarm chat join` as well; it is gone, because the bridge already
attends before the first turn and a second path to the same idempotent call buys
nothing. The trade is deliberate and worth stating: a harness that installs the
hooks and no MCP server no longer joins by itself. Every hook below still runs,
and `crabswarm chat join` typed by hand still works.

Running out of attempts does not take the bridge down. It stays up serving tools
that report why they cannot act, and each tool call asks to attend again — so a
daemon that comes up late is picked up by the next thing the agent does, which
is the same "a late delivery, never a lost message" the hooks aim for.

## What each hook does

| Event | Runs | Purpose |
| --- | --- | --- |
| `UserPromptSubmit` | `crabswarm chat report-state working` | A turn began. |
| `Notification` (`permission_prompt`) | `crabswarm chat report-state waiting` | A permission prompt is open — Claude Code only. |
| `Notification` (`idle_prompt`) | `crabswarm chat report-state done` | The session has been sitting quiet for about a minute — Claude Code only (see below). |
| `PermissionRequest` | `crabswarm chat report-state waiting` | An approval dialog is about to open. |
| `PostToolUse` | `crabswarm chat read --quiet`, then `crabswarm chat report-state working` | Deliver messages that arrived mid-turn as `additionalContext`; the dialog, if there was one, has resolved. |
| `Stop` | `crabswarm chat read --quiet --done-when-empty`, or `report-state done` | Drain the inbox; block the stop when it had mail, otherwise report done. |

The second `PostToolUse` entry is how a member gets out of `waiting` again.
Neither harness announces a dialog being answered or dismissed, so the next tool
call completing is the only signal that the approval the member was waiting on
resolved.

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

`crabswarm chat read` **consumes**: the daemon hands a message over exactly
once, and an inbox cannot be peeked at without draining it — `chat history`
re-reads the room's transcript, not this member's pending mail. So the
PostToolUse hook injects the messages it drained rather than a "you have mail"
notice — that injection *is* the delivery. The alternative, a hint that made
the agent read separately, would cost the same round trip and leave the
messages sitting in the inbox in the meantime.

The same fact shapes the Stop hook: when `stop_hook_active` is set, an earlier
Stop hook already blocked this turn, so the command template renders the
`report-state done` branch and does **not** read at all. A second drain would
either loop the agent or, once the harness stops honoring the block, swallow
messages it never displayed. Leaving them in the inbox costs a late read;
draining them with nowhere to put them costs the message.

Both hooks emit the minimum JSON their event allows — `decision`/`reason` for
Stop, `hookSpecificOutput.additionalContext` for PostToolUse — and nothing
else. Codex rejects an output object carrying fields its schema does not know,
and a rejected output *after* a drain is exactly the lost message these hooks
exist to prevent. The output builder emits only the fields the template's
functions set, so that holds as long as no other function is called.

### Why the Stop hook needs two flags on one read

One `hook exec` invocation runs one command, and the Stop hook has two things
to do that must not come apart: deliver whatever the drain found, and report the
member done exactly when the turn is really ending. `crabswarm chat read` takes
both as flags so a single process decides:

- `--quiet` drops the empty-inbox line, so the hook can tell mail from no mail
  by whether the output is empty at all. Without it the test would be a string
  comparison against the sentence the renderer prints for an empty inbox, which
  makes a wording nobody thinks of as an interface into one.
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

`crabswarm chat read` with no flags is unchanged: an empty inbox still says
`no pending messages`, which is what a human wants to see.

### Graceful degradation

Messages persist server-side until read, so nearly every failure path here
costs a late delivery and never a message:

- Every fire-and-forget hook records nothing in its output template, so a
  daemon that is not running, or a session with no identity token, never breaks
  a turn. The command's own stderr is captured by `hook exec` and never reaches
  the transcript.
- A failed `chat read` produces no `.Stdout`, so nothing is injected: nothing
  was handed over, so nothing is lost.
- Every hook entry carries a `timeout`. The PostToolUse hook runs after every
  tool call, and a wedged daemon must not stall the session.
- Hooks are independent. One that breaks costs only its own contribution —
  the inbox is still drained by the next hook that runs.

Two paths are now louder than they used to be, and both are outside what a
hook can quiet down:

- An envelope `hook exec` cannot parse is a plain error, so the hook exits 1
  with `parsing hook input: ...` on stderr instead of quietly leaving the inbox
  alone. A harness that writes unparseable JSON to a hook's stdin is broken in
  a way worth hearing about.
- No `crabswarm` on `PATH` at all is the shell's `command not found` and exit
  127, where the shell scripts used to swallow it. That is the assumption at
  the top of this file failing, which is worth saying out loud too.

Neither is a block: a decision rides on stdout and there is none, and only exit
2 blocks an event. So the worst either costs is a noisy line and a late
delivery.

The last line of defence is outside this package: the daemon nudges the terminal
of an agent that reported `done` (`[crabswarm chat] new message from ...`),
which the skill teaches the agent to answer with `crabswarm chat read`.

### The idle notification is how an interrupted turn heals

Claude Code's `Notification` hook fires for a permission prompt and again for
the idle prompt roughly a minute after Claude stops responding, and its matcher
selects on the event's `notification_type`. The two are wired apart because they
mean opposite things: `permission_prompt` is a member blocked on a dialog,
`idle_prompt` is a member with nothing left to do.

The idle branch is the fast path back for a turn the user interrupted. `Stop`
hooks do not run when a session is cancelled with ESC, so the member keeps
whatever the last report left it in — `working` after a tool call, `waiting` if
a dialog was open — and the daemon nudges a **done** member on sight, while a
`working` or `waiting` one is left alone as busy. The daemon does heal that on
its own: a report older than ten minutes has stopped describing the terminal,
so it is nudged anyway, with the screen snapshot the injection takes still
standing between the nudge and a terminal that turned out to be busy. Reporting
`done` on the idle prompt is what makes the wait a minute instead of ten,
without the agent doing anything.

Every other notification type — `elicitation_complete`, `elicitation_response`,
`auth_success` — now reports nothing rather than `waiting`. That is deliberate:
`PermissionRequest` covers the dialog case on both harnesses on its own, and a
catch-all group would also match `idle_prompt` and race a `waiting` report
against the `done` one, which is the wedge this split exists to remove.

## Codex

**Best-effort and unverified.** Codex's hook surface is recent; the wiring was
written against the Codex source (its `hooks.json` loader,
`HookEventsToml`/`MatcherGroup`/`HookHandlerConfig` shapes, and the
Stop/PostToolUse stdin and output schemas) but **has not been run against a
Codex session**. Treat every behavior below as a claim to check, not a fact.
Each hook is independent and failure-tolerant, so a hook Codex silently ignores
degrades to late delivery.

What has been checked from this side is that `crabswarm hook exec` speaks
Codex's half of the envelope surface: a `PermissionRequest` envelope parses into
its typed variant and the hook exits 0 silently, which is what
`e2e/crabswarm/chat_hooks_test.go` runs the shipped commands to prove. An event
neither harness declares would parse too and simply render nothing.

Codex gets the same file Claude Code does. `apm` does not translate hook event
names for Codex — it merges whatever events the file declares into
`.codex/hooks.json` verbatim, unlike the Gemini target, which renames events on
the way out — so the merged file's `Notification` block reaches Codex's config
untouched. Codex deserializes that config into a struct of the events it knows
and ignores the rest, so the block costs a few unread lines and nothing else,
and `PermissionRequest` is the event that actually covers the case there.

Nothing here needs a rewritten path. The commands name `crabswarm` and nothing
else, so the file installs exactly as written and does not care what directory
Codex runs its hooks from.

Codex loads hooks as untrusted until you say otherwise — expect to approve them
before they run.

The MCP server is the same story one layer over: `apm install` writes an
`[mcp_servers.crabswarm-chat]` table into `.codex/config.toml` naming
`crabswarm` with `["chat", "mcp"]`, which is what a Codex install was observed
to produce; whether Codex then starts the bridge and joins the room has not been
run against a Codex session either. Without it Codex attends only when someone
types `crabswarm chat join`.

What Codex ends up running:

| Event | Runs | Purpose |
| --- | --- | --- |
| `UserPromptSubmit` | `report-state working` | A turn began. |
| `PermissionRequest` | `report-state waiting` | An approval dialog is about to open. |
| `PostToolUse` | `chat read --quiet`, then `report-state working` | Deliver mid-turn messages; the dialog, if there was one, has resolved. |
| `Stop` | `chat read --quiet --done-when-empty`, or `report-state done` | Drain the inbox; report done when the stop goes through. |

The one difference from what Claude Code runs is the missing `Notification`:
`PermissionRequest` covers the dialog case, and nothing on Codex covers a
session sitting idle. So the split above buys Codex nothing — Codex announces no
event that says "this session has gone quiet", so there is nothing to wire. An
interrupted turn is left to the daemon's own fallback: ten minutes after the
last report, a `working` or `waiting` member is treated as no longer describing
its terminal and is nudged anyway. Codex heals the same way Claude Code does
without the idle branch, just slower.

Codex's `notify` program (`agent-turn-complete`) could report `done` redundantly,
but the Stop hook already does it — the only thing this package puts in
`config.toml` is the bridge's `[mcp_servers]` table.

## Verifying a change

The hook file parses with `jq`, and `e2e/crabswarm/chat_hooks_test.go` drives
the wiring end to end from Go: it reads the file, pulls each `command` string
out, and runs it **verbatim** through a shell with sample hook envelopes
on stdin, the real `crabswarm` binary on `PATH` and a real daemon behind it. So
what the suite exercises is the wiring that ships, not a Go paraphrase of it —
including that every command stays silent and exits 0 when no daemon is
running, that the delivering hooks consume the inbox exactly when they hand
something over, and that no command reaches back out to a script, to `jq`, to
`${CLAUDE_PLUGIN_ROOT}` or to the empty-inbox wording.

The flags the Stop hook depends on are covered a level down, in
`crabswarm/chat/cli/member_test.go`: the done report a drain makes on an empty
inbox is not visible from outside the daemon, so it is asserted against the RPC
there.
