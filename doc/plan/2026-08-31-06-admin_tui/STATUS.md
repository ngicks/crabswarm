# Status — admin TUI

Current state: **implemented** 2026-09-02 — all six steps landed on
`worktree-admin-tui`. The contracts this plan consumes all exist on
main: the admin auth plane, `AdminService.Send`, `AdminService.ListRooms`
and `ChatAdminService.History` with its `since_id` cursor. Judgment
calls made while implementing are DECISION.md D7–D14, tagged automatic
like the rest. IDEA.md's gate is still unconfirmed by the user, and the
automatic decisions are still open to review.

## Checklist

- [x] Step 1 — preview mock under the plan dir, "with a MOCK_LIMITS
      note (fakes: RPCs, states, timing)" (D6: mock deferred to
      implementation start) — `mock/main.go`, behind the `tuimock`
      build tag (D7)
- [x] Step 2 — `crabswarm/chat/cli/tui` model: D4 "one screen,
      conversation dominant, roster collapsible", D5 "no
      CLI-presentation logic under ./cmd", D1 bubbletea v2 pinned
      (v2.0.9, with bubbles v2.2.1 and lipgloss v2.0.6)
- [x] Step 3 — data loops: D2 "polls the room-log read RPC with a
      since-id cursor (~1s), plus a slower roster poll (~5s)"; startup
      back-page fill (IDEA.md UC2), which is one tail read (D12)
- [x] Step 4 — send path reusing `chat send`'s `to: text` addressing
      (IDEA.md UC3, D4), parsed by `cli.ParseAddressedLine` (D10)
- [x] Step 5 — command wiring: D3 "`--room` required; unknown rooms
      error listing existing rooms", registered under the `chat admin`
      group; the room check itself lives in `tui.Run` (D11)
- [x] Step 6 — e2e smoke: history fill, live message, admin send
      (IDEA.md UC1–UC3); UC4 failure paths asserted as an exit and a
      line on stderr

## Next action

User review of IDEA.md (gate) and of the automatic decisions — D7–D14
in particular, which are the surface this implementation added beyond
what the plan stated. The upgrade path D2 names, swapping the tail poll
for `ChatService.WatchRoom`, stays open and changes none of the `tui`
package's interfaces.

HANDOFF.md folded into doc/plan/issue/issue.md 2026-09-02 (run of the
implement-all-plans goal); entries there are the durable copies.
