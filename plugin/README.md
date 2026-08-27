# crabswarm plugin

Wires an agent harness into its `crabswarm chat` room: it joins on session
start, delivers teammates' messages mid-turn and again at turn end, and reports
what the harness is doing so the daemon knows when a terminal nudge is safe.

The agent-facing half is the [`crabswarm-chat`](skills/crabswarm-chat/SKILL.md)
skill, which teaches the room's verbs and etiquette. The hooks below only move
messages; the skill is what makes an agent answer them.

Everything here assumes `crabswarm` is on `PATH` — the same assumption the
skill makes when it tells an agent to type `crabswarm chat read`.

## Layout

```
plugin/
├── .claude-plugin/plugin.json    Claude Code plugin manifest
├── hooks/hooks.json              Claude Code hook wiring
├── codex/hooks.json              Codex hook wiring (best-effort, see below)
├── scripts/
│   ├── chat-stop-drain.sh        Stop: drain the inbox, block the stop if it had mail
│   └── chat-inbox-hint.sh        PostToolUse: deliver what arrived mid-turn
└── skills/crabswarm-chat/SKILL.md
```

The two scripts are shared verbatim by both harnesses: Codex feeds hooks the
same snake_case envelope on stdin and reads the same camelCase decision back.

## Install (Claude Code)

Load the directory straight from a checkout:

```console
claude --plugin-dir ./plugin
```

For a persistent install, publish it through a marketplace catalog — a
`.claude-plugin/marketplace.json` listing `./plugin` as a plugin source, added
with `/plugin marketplace add <dir>` and installed with `/plugin install`. No
catalog ships here; `--plugin-dir` is the supported path today.

## What each hook does

| Event | Command | Purpose |
| --- | --- | --- |
| `SessionStart` | `crabswarm chat join >/dev/null 2>&1 \|\| true` | Attend the room. Idempotent, so the duplicate SessionStart a resumed session fires is harmless. |
| `UserPromptSubmit` | `crabswarm chat report-state running >/dev/null 2>&1 \|\| true` | A turn began. |
| `Notification` | `crabswarm chat report-state waiting_input >/dev/null 2>&1 \|\| true` | A prompt is open (see the caveat below). |
| `PostToolUse` | `"${CLAUDE_PLUGIN_ROOT}"/scripts/chat-inbox-hint.sh` | Deliver messages that arrived mid-turn as `additionalContext`. |
| `Stop` | `"${CLAUDE_PLUGIN_ROOT}"/scripts/chat-stop-drain.sh` | Drain the inbox; block the stop when it had mail, otherwise report idle. |

### Mid-turn delivery injects the messages, not a hint

`crabswarm chat read` **consumes**: the daemon hands a message over exactly
once, and there is no peek. So the PostToolUse hook injects the messages it
drained rather than a "you have mail" notice — that injection *is* the
delivery. The alternative, a hint that made the agent read separately, would
cost the same round trip and leave the messages sitting in the inbox in the
meantime.

The same fact shapes the Stop hook: when `stop_hook_active` is set, an earlier
Stop hook already blocked this turn, so the script does **not** read at all. A
second drain would either loop the agent or, once the harness stops honoring
the block, swallow messages it never displayed. Leaving them in the inbox costs
a late read; draining them with nowhere to put them costs the message.

Both scripts emit the minimum JSON their event allows — `decision`/`reason` for
Stop, `hookSpecificOutput.additionalContext` for PostToolUse — and nothing
else. Codex rejects an output object carrying fields its schema does not know,
and a rejected output *after* a drain is exactly the lost message these hooks
exist to prevent.

### Graceful degradation

Messages persist server-side until read, so every failure path here costs a
late delivery and never a message:

- Every fire-and-forget hook ends in `|| true` and discards its output. A
  daemon that is not running, or a session with no identity token, never
  breaks session start or a turn.
- Both scripts check for `jq` **before** reading. If it is missing they cannot
  encode a message back to the harness, so they leave the inbox untouched.
- A failed `chat read` exits 0 silently: nothing was handed over, so nothing
  is lost.
- Every hook entry carries a `timeout`. The PostToolUse hook runs after every
  tool call, and a wedged daemon must not stall the session.
- Hooks are independent. One that breaks costs only its own contribution —
  the inbox is still drained by the next hook that runs.

The last line of defence is outside this plugin: the daemon nudges an idle
agent's terminal (`[crabswarm chat] new message from ...`), which the skill
teaches the agent to answer with `crabswarm chat read`.

### Caveat: `Notification` also fires when the session goes idle

Claude Code's `Notification` hook fires both for a permission prompt and for
the idle prompt some time after a turn ends. This wiring maps both to
`waiting_input`, which means an idle-prompt notification overwrites the `idle`
the Stop hook just reported — and the daemon only nudges **idle** members. An
agent left alone long enough therefore stops receiving terminal nudges until
its next turn; its messages still arrive, just at the next turn boundary
instead of immediately.

The refinement is to branch on the event's `notification_type` — a permission
prompt is `waiting_input`, an idle prompt is `idle`, which is precisely where a
nudge is meant to land. That needs the exact `notification_type` values
confirmed against the harness before it is worth shipping, so the simple
mapping stands for now.

## Codex

**Best-effort and unverified.** Codex's hook surface is recent; the wiring in
`codex/hooks.json` was written against the Codex source (its `hooks.json`
loader, `HookEventsToml`/`MatcherGroup`/`HookHandlerConfig` shapes, and the
Stop/PostToolUse stdin and output schemas) but **has not been run against a
Codex session**. Treat every behavior below as a claim to check, not a fact.
Each hook is independent and failure-tolerant, so a hook Codex silently ignores
degrades to late delivery.

Install by copying the file to whichever `.codex/` directory should own it:

```console
# this project only
cp plugin/codex/hooks.json <project>/.codex/hooks.json

# every project
cp plugin/codex/hooks.json ~/.codex/hooks.json
```

Then **edit the two script paths**: unlike Claude Code, Codex has no
`${CLAUDE_PLUGIN_ROOT}`, so the file ships `/path/to/crabswarm/plugin/scripts/`
placeholders. Point them at this checkout, or copy the two scripts somewhere
stable and point them there.

Codex loads hooks as untrusted until you say otherwise — expect to approve them
before they run.

The wiring:

| Event | Command | Purpose |
| --- | --- | --- |
| `SessionStart` | `crabswarm chat join` | Attend the room. |
| `UserPromptSubmit` | `report-state running` | A turn began. |
| `PermissionRequest` | `report-state waiting_input` | An approval dialog is about to open. |
| `PostToolUse` | `chat-inbox-hint.sh`, then `report-state running` | Deliver mid-turn messages; the dialog, if there was one, has resolved. |
| `Stop` | `chat-stop-drain.sh` | Drain the inbox; report idle when the stop goes through. |

Two differences from the Claude Code wiring, both deliberate:

- Codex reports `running` again from `PostToolUse`. It has no event for a
  dialog being answered or dismissed, so a tool call completing is the only
  signal that a `PermissionRequest` resolved. Claude Code needs no such
  workaround.
- Codex has no `Notification` equivalent; `PermissionRequest` covers the
  dialog case, and nothing covers a session sitting idle.

Codex's `notify` program (`agent-turn-complete`) could report idle redundantly,
but the Stop hook already does it — no `config.toml` change ships here.

The hooks move messages; they do not teach an agent to answer them. Make
[`skills/crabswarm-chat/SKILL.md`](skills/crabswarm-chat/SKILL.md) reachable
from the Codex side too — copy it into a skills directory Codex reads, or point
at it from the project's `AGENTS.md`.

## Verifying a change

The JSON files parse with `jq`, the scripts are POSIX `sh` (`sh -n`), and both
scripts can be exercised end to end by putting a stub `crabswarm` on `PATH` and
piping a sample hook envelope through them.
