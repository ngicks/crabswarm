# DECISION — native notify spike

Stubs seeded from IDEA.md open questions; inherited decisions quoted from
the parent plan `doc/plan/2026-08-26-01-chat_subcommand/DECISION.md`.

## Inherited (verbatim quotes)

- Parent D19: "v1 ships only the cmdman `send-keys` notifier adapter. The
  `Notifier` interface still lands (D9's pluggability survives), but the
  claude and opencode native adapters, the D13 endpoint mounts, and the
  crabswarm-as-channel-server direction move to a dedicated spike plan."
- Parent D9 (narrowed by D19): "pluggable notifier with per-harness
  adapters — Claude Code via its cross-session messaging socket (…),
  OpenCode via its HTTP server API (…), Codex and anything unknown via
  cmdman `send-keys` keystroke injection."

## Stubs

### S1. Track B scope (Q1) — **out: Codex stays send-keys** [2026-08-27]
- **Choice (user's words, near verbatim):** "Ok, just for now Codex is
  only send-keys fallback." The Codex app-server track is out of this
  spike; the spike is Claude Channels only.
- **Rationale:** app-server is experimental, can't attach to foreign
  sessions (openai/codex#25914), and would require cmdman/devenv to own
  the Codex launch — too much architecture for a notification win.
- **Rejected:** timeboxed secondary track in this spike.
- Revisiting Codex-native notification is a future decision, not owned by
  any current plan.

### S2. Reply tool (Q2) — *stub*
- Push-only vs. push + reply tool.
- Tentative default: push-only first.

### S3. OpenCode inclusion (Q3) — *stub*
- Prototype the opencode HTTP adapter here, or leave to the parent plan.
- Tentative default: leave out (needs no spike).

### S4. Deliverable form (Q4) — *stub*
- Throwaway branch + findings vs. landable flag-gated code.
- Tentative default: branch + findings; promotion is the parent's call.
