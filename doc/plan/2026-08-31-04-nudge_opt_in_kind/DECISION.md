# Decisions — nudge opt-in kind for members

## D1: deferred by user (user decision, 2026-08-31)

The user deferred this item when triaging the chat feedback: "ignore that
for [now]. admin will never join as member. We may be coming back to later,
so it should be recorded as an issue item." Also from the same message:
"`--agent` means notification. But without it, there shouldn't be
notification."

- Choice: do not implement now; keep the plan drafted so the `--agent`
  semantics the user specified are not lost.
- Consequence: the human-in-a-room use case is served by the admin plane
  (`chat admin` plan) in the meantime.

## D2: represent opt-in via the existing MemberKind (automatic decision)

- Choice: `chat join --agent` → `KindAgent`; absent → `KindHuman`. The kind
  constants' doc comments change from provenance ("provider-originated" /
  "registered through an admin RPC") to capability ("nudgeable harness" /
  "inbox-only").
- Rationale: every guard that decides whether to type already branches on
  exactly this kind (`internal/cmdman/cmdman.go:103`, `status.go:88`,
  `service.go:185`); reusing it keeps the change confined to `Service.Join`
  plus the proto/CLI surface.
- Rejected: a separate `notify`/`nudge` boolean column (duplicates the kind
  gate, two sources of truth); a third `MemberKind` value like `shell`
  (agent/human already encodes precisely nudgeable/not; a third value forces
  every `Kind != KindAgent` site to re-decide).

## D3: hooks pass --agent; the daemon does not guess (automatic decision)

- Choice: the apm-package SessionStart hook command becomes
  `crabswarm chat join --agent`; the daemon never infers harness-ness from
  the process or resolver.
- Rationale: the joiner is the only party that knows what is running in the
  terminal; inference from cmdman metadata is exactly the heuristic that
  caused the original bug (every cmdman command assumed to be an agent).
- Rejected: defaulting to agent and adding `--no-nudge` (keeps the unsafe
  default; the user explicitly specified opt-in semantics — see D1 quote).

## D4 — left deferred during the 2026-09-02 implement-all run (automatic decision)

The autonomous run that implemented every other outstanding plan
("implement all unimplemented plans except the channels spike") skipped
this one: D1 records the user explicitly deferring it ("ignore that for
now ... We may be coming back to later"), and a blanket implement-all
directive was not read as silently re-scheduling a specifically deferred
plan whose change alters runtime nudge behavior for every joining agent.
The user can override by re-scheduling; the plan and the issue-backlog
entry are unchanged.
