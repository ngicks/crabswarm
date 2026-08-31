# Status — nudge opt-in kind for members

Current state: **deferred by user, not scheduled** (DECISION.md D1). Plan
drafted 2026-08-31 with automatic decisions; IDEA.md gate not confirmed.

Next action: none until the user re-schedules; then run the IDEA.md gate and
review the automatic decisions (D2, D3) before implementing.

## Checklist (all blocked on D1 deferral)

- [ ] Step 1 — proto `JoinRequest.agent` field + regenerate (D2: kind chosen
      from the join request)
- [ ] Step 2 — `Service.Join` maps flag to `KindAgent`/`KindHuman`; kind doc
      comments updated (D2: "represent opt-in via the existing MemberKind")
- [ ] Step 3 — `chat join --agent` CLI flag threaded through
      `chatcli.Client.Join` (D1 quote: "`--agent` means notification. But
      without it, there shouldn't be notification.")
- [ ] Step 4 — apm-package SessionStart hook joins with `--agent` (D3: "hooks
      pass --agent; the daemon does not guess")
- [ ] Step 5 — unit + e2e tests: no-flag member never receives send-keys;
      `--agent` member does
