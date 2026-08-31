# Decisions — admin TUI

All entries below are tagged **(automatic decision)**: drafted without
user confirmation per the user's 2026-08-31 directive ("Skip all
questions; I'll use your recommendation for now, so tag it automatic
decision"). Each is revisitable at plan review.

## D1 — TUI framework: charm.land/bubbletea/v2 (automatic decision)

Chosen: bubbletea v2 (with its viewport component for the conversation
pane). Rationale: the repository has no TUI stack yet, and the ngplan
skill's own guidance names `charm.land/bubbletea/v2` as the Go TUI
default; it is the de-facto standard and its Elm-style model suits a
poll-driven watch screen. Rejected: tview (heavier widget model, less
idiomatic here), raw termbox/ansi (hand-rolling layout and input for no
benefit).

## D2 — Live tail by cursor polling, no streaming RPC in v1 (automatic decision)

Chosen: the TUI polls the room-log read RPC with a since-id cursor
(~1s), plus a slower roster poll (~5s), in v1; it switches to
`ChatService.WatchRoom` once plan `2026-08-31-02-chat_mcp_server`
delivers it. Rationale: polling needs no new RPC surface and keeps
this plan decoupled from plans 02/05 landing order. (Reconciled
2026-08-31: this entry originally claimed a streaming RPC "would poll
internally anyway" — that is wrong; the daemon owns every write, and
plan 02's D4 builds `WatchRoom` on an in-daemon per-room fan-out with
no polling. `WatchRoom` is therefore the committed upgrade path, not a
hypothetical.) Rejected: blocking this plan on `WatchRoom` (couples
TUI delivery to the MCP plan), tailing the DB file directly (bypasses
the auth plane).

## D3 — `--room` required; no picker screen (automatic decision)

Chosen: `chat admin tui --room R` errors without `--room`; unknown
rooms error listing existing rooms. Rationale: the user stated the
admin "always needs to specify room id"; a picker would contradict
that and add a screen with no requested use case. Rejected: interactive
room picker on launch, defaulting to the sole room when only one
exists (implicit magic).

## D4 — Single-screen layout: conversation viewport + roster sidebar + input/status (automatic decision)

Chosen: one screen, conversation dominant, roster collapsible on
narrow terminals, one input line reusing `chat send`'s `to: text`
addressing. Rationale: watching is the primary mode (IDEA.md); tabs or
multi-screen navigation add chrome without an oversight use case.
Rejected: multi-room tab view (admin can run one TUI per pane under
cmdman), separate send dialog.

## D5 — Code lives in `crabswarm/chat/cli/tui`; `./cmd` stays thin (automatic decision)

Chosen: new package `crabswarm/chat/cli/tui` holding all presentation;
`cmd/crabswarm/commands/chat_admin_tui.go` only parses flags, dials as
admin, and calls `tui.Run`. Rationale: repo design rule — no
CLI-presentation logic under `./cmd`; terminal control belongs in a
`cli/` package. Rejected: package directly under `crabswarm/chat/tui`
(splits presentation across two roots).

## D6 — No presentation preview in the drafting pass (automatic decision)

Chosen: the ngplan preview mock (bubbletea layout demo with fixture
data + MOCK_LIMITS) is planned as implementation step 1, not created
while drafting. Rationale: the user asked for plan drafts only; the
layout has an IDEA.md flowchart for review, and a runnable mock is
cheap to produce when implementation starts. Rejected: drafting the
mock now (premature while sibling contracts may still move).
