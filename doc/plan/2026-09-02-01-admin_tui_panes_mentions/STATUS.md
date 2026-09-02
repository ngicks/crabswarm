# Status — admin TUI panes, textarea, `@` addressing, editor

Current state: **implementing** (autonomous run, 2026-09-03). IDEA.md gate confirmed
2026-09-02; every open question resolved (D1–D22); PLAN.md carries the
surface delta and seven steps. Awaiting the go-ahead to implement.

## Checklist

Steps are filled in after the idea gate. Until then:

- [x] Open questions 1, 2, 4, 5, 7 resolved → D1, D2, D3, D9
- [x] Open questions 3, 6, 13 resolved → D12, D4, D11
- [x] Layout redrawn per D13 (left column rooms/members, right column
      log/textarea); open questions 16–19 raised
- [x] Open question 16 resolved → D14 (`--room` optional, pre-selects)
- [x] Open questions 17, 18 resolved → D15 (focus from layout), D16
      (per-room drafts)
- [x] Preview mock written under `mock/` on the D13 layout;
      `MOCK_LIMITS.md` emitted
- [x] IDEA.md gate confirmed by user (2026-09-02)
- [x] Contract questions 8, 9, 11, 14, 15 resolved → D6, D7, D17, D10
- [x] Contract questions 10, 12, 19, 20 resolved → D18, D8, D19, D20
- [x] PLAN.md approach, surface delta, steps, tests written
- [x] Traceability gate passed (see below)

## Implementation checklist

- [x] Step 1 — D10 "`oneof target` … `Everyone`, `TeamTarget{team}`,
      `MemberTarget{team, name}`", D20 "`team` and `name` as two fields;
      empty `team` means the bare-name rule"; D10 "delivers to every
      current member of that team, counted at send time"
- [ ] Step 2 — D10 "`chat admin send` maps its argv grammar … onto the
      cases client-side"; D3 "`cli.ParseAddressedLine` … is removed"
- [ ] Step 3 — D21 palette "one palette block in `tui/styles.go`"; D5
      "rectangles come from `ultraviolet/layout`"; D13
      two-column layout; D15 "target is the pane adjacent … whose rows
      overlap … the most (tie: the upper one)"; D18 "a focus move toward
      the hidden column … brings the left column on screen"
- [ ] Step 4 — D14 "`--room` … optional … opens on that room"; D19
      "same reply now fills both left panes … resets the log cursor";
      D16 "one draft per room"; D1 "`enter` on a member pre-fills
      `@team/name `; `enter` on a team pre-fills … `@team/* `"
- [ ] Step 5 — D22 "bare `@admin` or `@admin/admin` … drawn in the
      mention color"; D2 "`ctrl+enter` sends. `enter` is always a newline";
      D11 `ctrl+x` fallback; D3 "first `@token` outside backticks and
      not escaped as `\@` names the target; the message text is sent
      whole"; D6 parser tui-internal; D7 disambiguation requested; D17
      "one row up to six"
- [ ] Step 6 — D8 "`ctrl+g` runs `$VISUAL`; when that is unset,
      `$EDITOR`"; D4 "every notice … moves to a one-line system line";
      D12 log focusable with vim keys
- [ ] Step 7 — goal 6 e2e for UC1b, UC3, UC3b, UC4, UC7, UC8, `--room`
      omitted

## Traceability

| Decision clause | Owner |
|---|---|
| D1 roster selects members and teams; team send target | steps 4, 1 |
| D2 ctrl+enter sends, enter newline | step 5 |
| D3 `@` grammar, whole text, `to:` removed | steps 5, 2 |
| D4 system line + status bar | step 6 |
| D5 ultraviolet layout, direct dep | step 3 |
| D6 parser tui-internal | step 5 |
| D7 keyboard enhancements | step 5 |
| D8 VISUAL then EDITOR | step 6 |
| D9 mock before the gate | done (mock/) |
| D10 oneof target, team counted at send | steps 1, 2 |
| D11 ctrl+x fallback | step 5 |
| D12 log focusable, vim keys | step 3 (router), 6 (hint) |
| D13 two-column layout | step 3 |
| D14 --room optional | step 4 |
| D15 layout-derived focus | step 3 |
| D16 per-room drafts | step 4 |
| D17 textarea 1–6 rows | step 5 |
| D18 width gate, column swap | step 3 |
| D19 rooms ride the roster poll, re-tail on switch | step 4 |
| D20 MemberTarget {team, name} | steps 1, 2 |
| D21 bubbletea palette in styles.go | step 3 |
| D22 admin mentions colored | step 5 |

Use cases: UC1 → 3, 4; UC1b → 4; UC2 → 3; UC3/UC3b → 4, 1; UC4 → 5;
UC5 → 5, 1; UC6 → 5; UC7 → 6; UC8 → 5, 6; UC9 → 4 (`openRoom`).
Contract areas: public API, dependencies, RPC schema, CLI — fenced in
PLAN.md; project layout — files named per step; persistent data — no
change, stated.

## Next action

User go-ahead, then step 1 (proto `oneof` + daemon), which every later
step's send path depends on.
