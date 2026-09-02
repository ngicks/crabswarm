---
tags: chat mcp daemon hooks
---

# Transform chat daemon into an MCP server, keeping the CLI (2026-08-31)

Native notification/hook methods vary between harnesses, so the chat
side should become an MCP server while every existing CLI verb stays.
One MCP server instance corresponds to one agent and calls Join
automatically at startup; `Service.Join` is already idempotent for a
known token (`crabswarm/chat/service_member.go`), so multiple MCP
servers in the same container are fine. Join reporting then comes from
the server itself, replacing the SessionStart hook.

Constraints verified against the MCP spec (2026-07-28) and Claude Code
docs: MCP defines no client-to-server busy/idle notification and no
turn-end signal, and Claude Code hooks cannot call MCP tools yet
(anthropics/claude-code#26112), so harness-state push stays hook-driven
(hooks shelling out to the CLI / a thin RPC). Related known defect: an
ESC interrupt fires no Stop hook ("Stop hooks ... don't fire on user
interrupts", code.claude.com/docs/en/hooks-guide), so the member state
sticks at working/waiting and `notify.SendKeys` (gating on StateDone in
`crabswarm/chat/notify/notify.go`) never nudges that member again; the
`Notification` hook's `idle_prompt` matcher (fires ~60s idle) mapped to
`report-state done`, plus a staleness fallback in the nudge gate, is
the available mitigation.

Follow-up: design the MCP surface (tools for send/read/report-state,
resources + `resources/updated` subscriptions for member state and room
history), wire server-startup auto-join, and keep the CLI as the
hook-facing entry.
