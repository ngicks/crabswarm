# PLAN — native notify spike (Claude Channels / Codex app-server)

**Summary:** spike a crabswarm-owned Claude Code channel server (and,
secondarily, Codex app-server driving) as the official-protocol
replacement for the chat notifier's `send-keys` nudges, ending in a
go/no-go recommendation for the chat plan's step 6.

> **Skeleton until the IDEA.md gate passes.** Another session picks this
> up: read IDEA.md, the parent plan
> `doc/plan/2026-08-26-01-chat_subcommand` (esp. D9/D13/D19, HANDOFF.md
> entry 1, and `notification_mechanisms.md`), then resolve the idea-level
> open questions with the user and run the gate before detailing here.

## Goal / success criteria (from IDEA.md)

- Working prototype: a Go MCP-stdio channel server that pushes a chat
  arrival event into a live Claude Code session launched with the dev
  flag. (Codex is out of scope — S1: it stays on `send-keys`.)
- A written recommendation: adopt / defer for Claude, with the exact
  adapter surface the chat plan would need (config keys, JoinRequest
  fields 2–3 to un-reserve, launch wiring, degradation story).

## Scope (tentative)

- New spike code, location TBD (suggestion: `crabswarm/chat/channel/` or
  a branch-only prototype per Q4).
- No changes to the chat plan's v1 deliverables; this plan only produces
  evidence and a recommendation.

## Non-goals

- Shipping a production adapter (promotion is the chat plan's decision).
- OpenCode adapter (pending Q3; needs no spike — documented HTTP API).
- Solving the channels preview allowlist (Anthropic-side; we document
  friction, we don't work around it beyond the dev flag).

## Context — verified facts to build on

All verified 2026-08-26/27; details and links in the parent plan's
`notification_mechanisms.md`.

- Channel contract (open, any language): MCP stdio subprocess; declare
  `capabilities.experimental['claude/channel']`; emit
  `notifications/claude/channel`; reply tools + permission relay specced.
  https://code.claude.com/docs/en/channels-reference
- Delivery is fully local; the Anthropic-auth requirement is preview
  feature-flag gating, not routing.
- Custom channels: `claude --dangerously-load-development-channels
  server:<name>` + interactive full-screen confirmation each launch;
  org allowlist (`allowedChannelPlugins`) is Team/Enterprise-only.
- Codex app-server: JSON-RPC 2.0 stdio, powers the VS Code extension;
  `thread/start|resume|fork`, `turn/start`, `turn/steer`,
  `thread/inject_items`; **no attach to foreign sessions**
  (openai/codex#25914); experimental, unsupported for production.
  https://learn.chatgpt.com/docs/app-server
- Parent-plan seams reserved for this spike: `Notifier` interface in
  `crabswarm/chat/notify.go` (chat step 6), `JoinRequest` fields 2–3
  reserved, `chat.notify` config left `{}`.

## Boundary ledger (with parent plan)

| Deliverable | Owner |
| --- | --- |
| v1 send-keys notifier | parent plan step 6 (D19) |
| Notifier interface seam | parent plan step 6 |
| Claude channel-server prototype + go/no-go | **this plan** |
| Codex notification | stays cmdman `send-keys` (S1) — parent plan step 6; native path owned by nobody until revisited |
| OpenCode HTTP adapter | pending Q3 (this plan or parent follow-up) |
| Promotion of any adapter into production | parent plan (future revision) |

## Approach

*(deferred until the idea gate passes)*

## Implementation steps

*(deferred until the idea gate passes)*

## Open questions

Idea-level Q1–Q4 live in IDEA.md and gate this plan.
