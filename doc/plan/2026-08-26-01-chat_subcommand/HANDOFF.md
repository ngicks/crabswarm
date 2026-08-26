# HANDOFF — work leaving this plan

## 1. Native notification adapters (user-approved deferral — D19)

- **What:** the non-send-keys notifier adapters for `crabswarm/chat/notify.go`:
  Claude Code (a crabswarm-owned Channels MCP server, or the messaging
  socket) and OpenCode (HTTP server API); plus the D13 devenv endpoint
  mounts and the reserved `JoinRequest` fields 2–3 (notify endpoint
  self-reporting). Codex is NOT part of this handoff: per the user
  ("just for now Codex is only send-keys fallback", spike-plan S1),
  Codex stays on `send-keys` with no native path owned by any plan.
- **Why not here:** user decision D19 ("For now we'll implement send-keys
  provider only. Spawn another plan about Channels so another session can
  spike on it."), superseding D13 for v1.
- **Follow-up:** owned by `doc/plan/2026-08-27-01-chat_channels_spike`,
  to be worked by another session. Research groundwork is in this plan's
  `notification_mechanisms.md`.
