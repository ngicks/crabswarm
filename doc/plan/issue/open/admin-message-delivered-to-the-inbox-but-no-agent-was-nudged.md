---
tags: chat notify cmdman proto hooks
---

# Admin message delivered to the inbox but no agent was nudged (2026-09-02)

An admin TUI send into `/home/watage/.dotfiles` landed in the store
(`messages` row 1, recipient token `11c9fdf9...`) and no agent
responded. The admin path does not skip anything: `AdminService.Send`
→ `deliverAdminMessage` → the same `deliverer.sendAs` and
`notify.SendKeys` the member path uses (`crabswarm/chat/admin_rooms.go`,
`crabswarm/chat/delivery.go`, one notifier built in
`crabswarm/server/server.go`). The hard fact is that the row is still
in `messages`: `Store.Read` drains the inbox, so no `chat read` hook
fired and no keystroke landed after the send. The notifier ran and
declined. Its guard chain (`crabswarm/chat/notify/notify.go`,
`crabswarm/chat/internal/cmdman/cmdman.go`): busy-state gate (passes
for a fresh join, state defaults to done), `Kind == KindAgent`, then
`capture-screen` availability and the dialog-marker scan. Leading
hypothesis, not yet confirmed against the live DB: the recipient is
`KindHuman`. `~/.codex/hooks.json` in the operator's home still carries
the SessionStart `crabswarm chat join` hook (flagless, so `KindHuman`)
that the package removed when the bridge took over joining with
`agent=true`; if the claude settings carry the same stale hook, the
flagless SessionStart join wins (`Store.Join` is first-join-wins,
`Service.Join`'s existing-member branch returns the stored kind) and
every nudge is declined forever — at Debug level (`chat: not typing
into a member that runs no harness`), invisible by default. Also
suspicious: one inbox row for a roster of three; the TUI status line
(`sent to X (N delivered)`) says whether the target was one member.

Follow-up:
- Confirm on the host: `select token,name,kind,state,state_reported_at
  from members where room='/home/watage/.dotfiles'`, and grep
  `~/.claude/settings*.json`, `~/.claude.json`, `~/.codex/hooks.json`
  for `chat join`. Remove stale SessionStart join hooks; `chat leave`
  and restart so the bridge's `--agent` join is the first join.
- Make the kind visible: proto `Member`
  (`api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto`)
  carries no kind, so `chat members`, `chat admin list` and the TUI
  roster cannot show who is nudge-capable, although the nudge plan's
  usability requirement asked for exactly that. Add it and render it.
- Let a re-join declaring `agent=true` upgrade `KindHuman` to
  `KindAgent` (resolves "A flagless chat member cannot upgrade to agent
  on the same token").
- Raise the kind decline from Debug to Info: a member that will never
  be nudged deserves one visible line, unlike the transient busy
  decline.
