# Status — admin TUI

Current state: **not started** — plan drafted 2026-08-31 with automatic
decisions (see DECISION.md); IDEA.md gate not confirmed. Blocked on:
idea-gate confirmation by the user, and contract reconciliation with
sibling plans 2026-08-31-01-chat_admin_subcommand (admin send verb/RPC)
and 2026-08-31-05-per_room_message_history (room-log read RPC).

## Checklist

- [ ] Step 1 — preview mock under the plan dir, "with a MOCK_LIMITS
      note (fakes: RPCs, states, timing)" (D6: mock deferred to
      implementation start)
- [ ] Step 2 — `crabswarm/chat/cli/tui` model: D4 "one screen,
      conversation dominant, roster collapsible", D5 "no
      CLI-presentation logic under ./cmd", D1 bubbletea v2 pinned
- [ ] Step 3 — data loops: D2 "polls the room-log read RPC with a
      since-id cursor (~1s), plus a slower roster poll (~5s)"; startup
      back-page fill (IDEA.md UC2)
- [ ] Step 4 — send path reusing `chat send`'s `to: text` addressing
      (IDEA.md UC3, D4)
- [ ] Step 5 — command wiring: D3 "`--room` required; unknown rooms
      error listing existing rooms", registered under plan 01's
      `chat admin` group
- [ ] Step 6 — e2e smoke: history fill, live message, admin send
      (IDEA.md UC1–UC3); UC4 failure paths asserted

## Next action

User review of IDEA.md (gate) and of the automatic decisions; then
reconcile `LogReader`/`AdminSender` interface shapes against plans 01
and 05 before starting step 2.
