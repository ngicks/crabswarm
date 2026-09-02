# Mock limitations — `mock/main.go`

What the preview fakes, and which IDEA.md use cases and DECISION.md
entries it therefore cannot validate. The `# MOCK_LIMITS` header in
`mock/main.go` is the source; this file maps each item onto the plan so a
"validated in the mock" claim can be checked against it. Run from `main/`:

    go run -tags tuimock ./doc/plan/2026-09-02-01-admin_tui_panes_mentions/mock

## What the mock validates

Interaction decisions only: the two-column layout and its collapse (D13,
D18), layout-derived focus moves (D15, measured at 80×24, 60×10, 59×24),
the members cursor and pre-fill (D1), the `@` grammar and completion
(D3), `ctrl+enter`/`ctrl+x` and `enter`-as-newline (D2, D11), per-room
drafts within one process (D16), the editor hand-off (D8, real
`$VISUAL`/`$EDITOR`), the palette (D21) and the mention rule (D22) as
rendering.

## What it fakes, and what that leaves unvalidated

| Faked | Cannot validate |
|---|---|
| **RPCs** — nothing is dialled; rooms, conversation, roster and send are in-process fixtures. | Admin auth (UC9), the room-log read and its cursor paging (D19), `AdminService.Send` with the `oneof` target (D10, D20), the delivered count as the daemon reports it (UC3, UC3b, UC8). |
| **Rooms** — a fixture of four paths; selecting swaps fixtures, and the 8 s tail appends to whichever room is on screen. | How the list is kept current, what happens to a room not being looked at, an empty daemon opening onto an empty screen, a room appearing or vanishing mid-session (UC1b, D14, D19). |
| **Drafts** — per room for the life of the process only. | Whether a draft should outlive the screen (D16 says it does not; not exercised). |
| **Target resolution** — resolved against the fixture roster; bare names take the first match. | The daemon's ambiguity rejection, members that joined since the roster was read, an empty team (UC3, D20, PLAN step 1 tests). |
| **Delivered counts** — counted off the fixture. | A send that reaches nobody (`NotFound`) and how the system line reports it (UC6's failure branch, D4). |
| **Local echo** — a sent message is appended after a fake 300 ms. | The real no-echo rule (inherited prior D13) and the latency until the log brings it back (UC8). |
| **Timing** — a canned message every 8 s instead of a poll. | Poll latency, cursor paging, retention (D19; prior plan's poll loops). |
| **States** — hardcoded, never change. | Roster state updates in place (UC1). |
| **Retention** — fixed scrollback list. | `history_limit` pruning, `gg` on a pruned log (UC2). |
| **Keyboard enhancements** — a terminal that ignores the kitty query answers nothing, so the fallback hint never appears. | Whether the operator's terminal can report `ctrl+enter` (D7, D11); only `ctrl+x` is known to work everywhere. |
| **Editor** — run for real. | Nothing faked here; what it hands back to is. |
| **Palette** — ANSI-256 numbers; what they look like is the terminal's theme. | Readability and contrast on the operator's actual theme (D21 requirement "readable on dark and light terminals"). |
| **Mentions** — textual `@admin` / `@admin/admin` scan by the send tokenizer's rules. | Whether the daemon should instead report who a message reached, and anything a mention should do beyond colour (D22 scope). An operator's own `\@admin` colours itself once sent (PLAN risk). |

## Known screen behaviours the mock is honest about

- Unbinding `ctrl+h` from the textarea's delete-backward costs Backspace on
  terminals that send `^H` for it (D7 mitigates where the terminal can
  disambiguate).
- A multi-line message is folded into one conversation line with ` ⏎ `
  marking newlines rather than drawn as continuation lines; the real
  screen's rendering of multi-line entries is not decided by the mock.

## Promotion rule

When mock code is lifted into `crabswarm/chat/cli/tui`, read its comments
side by side against the DECISION.md wording it implements; a discrepancy
is raised with the user, never copied forward. The mock directory is
deleted at PLAN.md step 6.
