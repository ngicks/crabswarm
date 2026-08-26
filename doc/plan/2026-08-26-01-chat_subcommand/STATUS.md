# STATUS — `crabswarm chat` subcommand

**Current state:** Plan FINALIZED (2026-08-27) — idea gate confirmed,
D1–D18 decided, traceability gate passed. Ready to implement; no code
written. One deliberate implementation-time confirmation remains: C3
(cmdman query CLI surface + devenv mount layout; external to this repo).

## Checklist

- [x] Idea-level open questions resolved with the user (D1–D12)
- [x] IDEA.md gate confirmed by user (2026-08-27)
- [x] PLAN.md contracts detailed (proto, cobra surface, config keys)
- [x] Contract-level open questions resolved (C1→D13, C2→D14, C4→D15,
      C5→D16, C6→D17, C7→D18; C3 deferred to implementation-time
      confirmation by design)
- [x] Traceability gate walked (clause→step table in PLAN.md; all owned)
- [x] D1 "supersede" enacted: old agent_swarming STATUS.md marked superseded
- [ ] Implementation (steps 1–9 in PLAN.md; not started)
  - [ ] 1. D17 proto rename + `ngicks.crabswarm.chat.v1` schema
  - [ ] 2. D14 SQLite store; D10(a)-(c) resolution rules; D12 double-join no-op; D20(a) member state column
  - [ ] 3. D11(a)-(b) cmdman-compose provider; D10(d) work_dir⇒room mapping
  - [ ] 4. D3 service on existing daemon; D8(c) token interceptor; D15 Broadcast; D18 lazy reap
  - [ ] 5. D7 age nonce challenge; D16 RegisterMember; D4(d) formation edits
  - [ ] 6. D9 notifier interface + D19 send-keys-only adapter; D20(b)
        idle-only injection + D20(c) capture-pane guard (native adapters
        handed off to chat_channels_spike — see HANDOFF.md)
  - [ ] 7. D5(a)-(b) CLI verbs; D8(b) token metadata; D15 broadcast verb
  - [ ] 8. D5(c) skill + per-harness hook wiring (claude plugin, codex
        hooks.json): join, D20(a) state reporting, D9 turn-boundary
        reinforcement
  - [ ] 9. e2e: delivery, collision addressing, room isolation, D12 double-join, D11(a) rejection

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

- Nothing. Planning complete.

## Blocked / needs decision

- None blocking. Implementation-time confirmation (not a blocker): C3 —
  exact cmdman query surface for token → {work_dir, compose project},
  `send-keys` invocation, and the devenv mount layout for D13 endpoints.

## Next action

- Begin Step 1 (D17 proto rename + chat schema) when the user is ready to
  implement.
