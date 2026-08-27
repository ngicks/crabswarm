# crabswarm-chat

Wires an agent harness into its `crabswarm chat` room, packaged for
[apm](https://github.com/microsoft/apm): it joins on session start, delivers
teammates' messages mid-turn and again at turn end, and reports what the harness
is doing so the daemon knows when a terminal nudge is safe.

The agent-facing half is the
[`crabswarm-chat`](.apm/skills/crabswarm-chat/SKILL.md) skill, which teaches the
room's verbs and etiquette. The hooks below only move messages; the skill is
what makes an agent answer them.

Everything here assumes `crabswarm` is on `PATH` — the same assumption the
skill makes when it tells an agent to type `crabswarm chat read`.

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
`.claude/settings.json` (Claude Code) and `.codex/hooks.json` (Codex), the two
shell scripts are copied to `<target>/hooks/crabswarm-chat/scripts/` with the
hook commands rewritten to point at them, and the skill materializes at
`.claude/skills/crabswarm-chat/` and `.agents/skills/crabswarm-chat/`. Codex
only receives hooks when a `.codex/` directory already exists in the project.

Crabswarm's own `apm.yml` does not depend on this package. Installing the chat
hooks into crabswarm's own development sessions is available by adding the
stanza above to the repository root `apm.yml`, and is deliberately not done by
default.

Installing from source is the only supported route. `apm pack` builds a
plugin bundle out of `plugin.json` plus the plugin-native directories, and
`scripts/` is not one of them — the two shell scripts every hook here runs
would be missing from the bundle, and the per-target hook files would be merged
into one. A bundle is the wrong shape for this package; the dependency stanza
above is the right one.

## Layout

```
apm-package/crabswarm-chat/
├── apm.yml                             package metadata (targets: claude, codex)
├── .claude-plugin/plugin.json          Claude Code plugin manifest
├── hooks/hooks.json                    hook wiring for every target but Codex
├── .apm/
│   ├── hooks/hooks-codex.json          hook wiring for Codex (see below)
│   └── skills/crabswarm-chat/SKILL.md
└── scripts/
    ├── chat-stop-drain.sh              Stop: drain the inbox, block the stop if it had mail
    └── chat-inbox-hint.sh              PostToolUse: deliver what arrived mid-turn
```

The two scripts are shared verbatim by both harnesses: Codex feeds hooks the
same snake_case envelope on stdin and reads the same camelCase decision back.

The package is also shaped like a Claude Code plugin, so a checkout should load
directly with `claude --plugin-dir ./apm-package/crabswarm-chat` — that path
reads `hooks/hooks.json` and `.claude-plugin/plugin.json` and ignores `.apm/`.
`apm install` is the route that has actually been exercised, though; the
`--plugin-dir` one is untested, and `hooks/hooks.json` carries a top-level
`"version": 1` that only `apm` is known to tolerate.
The hook commands are written as `"${CLAUDE_PLUGIN_ROOT}/scripts/…"` for exactly
this reason: Claude Code expands the variable itself, and `apm` rewrites the
same reference into a target-local path for every target it deploys to.

## What each hook does

| Event | Command | Purpose |
| --- | --- | --- |
| `SessionStart` | `crabswarm chat join >/dev/null 2>&1 \|\| true` | Attend the room. Idempotent, so the duplicate SessionStart a resumed session fires is harmless. |
| `UserPromptSubmit` | `crabswarm chat report-state running >/dev/null 2>&1 \|\| true` | A turn began. |
| `Notification` | `crabswarm chat report-state waiting_input >/dev/null 2>&1 \|\| true` | A prompt is open (see the caveat below). |
| `PostToolUse` | `chat-inbox-hint.sh` | Deliver messages that arrived mid-turn as `additionalContext`. |
| `Stop` | `chat-stop-drain.sh` | Drain the inbox; block the stop when it had mail, otherwise report idle. |

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

The last line of defence is outside this package: the daemon nudges an idle
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
`.apm/hooks/hooks-codex.json` was written against the Codex source (its
`hooks.json` loader, `HookEventsToml`/`MatcherGroup`/`HookHandlerConfig` shapes,
and the Stop/PostToolUse stdin and output schemas) but **has not been run
against a Codex session**. Treat every behavior below as a claim to check, not a
fact. Each hook is independent and failure-tolerant, so a hook Codex silently
ignores degrades to late delivery.

It ships as a second hook file rather than as the same one because `apm` does
not translate hook event names for Codex — it merges whatever events the file
declares into `.codex/hooks.json` verbatim, unlike the Gemini target, which
renames events on the way out. `Notification` has no Codex equivalent, so
shipping the Claude file to Codex would put an event Codex does not know into
its config; `PermissionRequest` is the event that actually covers the case.

`apm` picks the right file by filename: a hook file whose stem ends in a target
token (`hooks-codex`) is used only for that target, and a target with such a
file does not also receive the untagged `hooks.json`. Filename routing is
deprecated upstream and logs a warning suggesting per-dependency `targets:` in
the consumer's `apm.yml`, but that setting selects targets for a whole
dependency and cannot pick between two hook files inside one package, so this
remains the only in-package mechanism. If it is removed upstream, both files
would become universal and Claude would receive the Codex wiring too — merge
them by hand at that point.

Script paths need no editing any more. `${CLAUDE_PLUGIN_ROOT}` is rewritten for
every target, so the Codex file gets `.codex/hooks/crabswarm-chat/scripts/…`
from `apm install` just as the Claude file gets its own copy. That path comes
out *relative*, though — only the Claude target gets an absolute
`${CLAUDE_PROJECT_DIR}`-anchored one — so it resolves only when Codex runs its
hooks from the project root. If it runs them from elsewhere, the scripts will
not be found; rewrite the two commands to absolute paths in that case.

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

## Verifying a change

The JSON files parse with `jq`, the scripts are POSIX `sh` (`sh -n`), and both
scripts are exercised end to end from Go by `e2e/crabswarm/chat_hooks_test.go`,
which pipes sample hook envelopes through them against a stub `crabswarm` on
`PATH`.
