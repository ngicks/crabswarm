# STATUS — `crabswarm chat` subcommand

**Current state:** IMPLEMENTED (2026-08-27) — steps 1–9 all landed on
feat-add-agent-chat with unit + e2e coverage; awaiting user review. C3
was confirmed empirically against the local cmdman v0.0.23 source
(see DECISION.md [automatic] entries: query surface + capture-pane
fallback via `logs --tail`).

## Checklist

- [x] Idea-level open questions resolved with the user (D1–D12)
- [x] IDEA.md gate confirmed by user (2026-08-27)
- [x] PLAN.md contracts detailed (proto, cobra surface, config keys)
- [x] Contract-level open questions resolved (C1→D13, C2→D14, C4→D15,
      C5→D16, C6→D17, C7→D18; C3 deferred to implementation-time
      confirmation by design)
- [x] Traceability gate walked (clause→step table in PLAN.md; all owned)
- [x] D1 "supersede" enacted: old agent_swarming STATUS.md marked superseded
- [x] Implementation (steps 1–9 in PLAN.md)
  - [x] 1. D17 proto rename + `ngicks.crabswarm.chat.v1` schema
  - [x] 2. D14 SQLite store; D10(a)-(c) resolution rules; D12 double-join no-op; D20(a) member state column
  - [x] 3. D11(a)-(b) cmdman-compose provider; D10(d) work_dir⇒room mapping
  - [x] 4. D3 service on existing daemon; D8(c) token interceptor; D15 Broadcast; D18 lazy reap
  - [x] 5. D7 age nonce challenge; D16 RegisterMember; D4(d) formation edits
  - [x] 6. D9 notifier interface + D19 send-keys-only adapter; D20(b)
        idle-only injection + D20(c) capture-pane guard (native adapters
        handed off to chat_channels_spike — see HANDOFF.md)
  - [x] 7. D5(a)-(b) CLI verbs; D8(b) token metadata; D15 broadcast verb
  - [x] 8. D5(c) skill + per-harness hook wiring (claude plugin, codex
        hooks.json): join, D20(a) state reporting, D9 turn-boundary
        reinforcement
  - [x] 9. e2e: delivery, collision addressing, room isolation, D12 double-join, D11(a) rejection
- [x] Post-review: D21 harness-state vocabulary aligned to
      `cmdman status set working|waiting|done` (proto enum renumbered,
      store strings, CLI words, `--done-when-empty` flag, hooks, docs)
- [x] Post-review: D22 chat store SQL migrated to sqlc
      (schema.sql + queries.sql + generated internal/chatdb)

## Done

- Repo grounding: root/serve cobra wiring, `crabswarm/server/server.go`
  (UDS + flock + AuditService), layered config, proto/buf layout.
- Superseded plan identified: `doc/plan/2026-06-27-01-agent_swarming`
  retired by D1.
- Notification-mechanism research (Claude Code / Codex / OpenCode)
  verified and recorded in notification_mechanisms.md.
- IDEA.md finalized and gate-confirmed; PLAN.md detailed with public
  surface delta and steps 1–9.

## In progress

- Nothing. Implementation complete; awaiting user review.

## Blocked / needs decision

- None blocking. Implementation-time confirmation (not a blocker): C3 —
  exact cmdman query surface for token → {work_dir, compose project},
  `send-keys` invocation, and the devenv mount layout for D13 endpoints.

## Next action

- User review of the implementation; then fold HANDOFF.md (native
  notification adapters; buf generate template unification) into the
  issue backlog if desired.
- Untested-in-environment items: web playwright e2e (browser binary
  missing here) and the Codex hook wiring (no Codex session available;
  marked best-effort in plugin/README.md).
