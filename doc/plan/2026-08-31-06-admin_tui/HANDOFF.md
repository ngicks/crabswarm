# Handoff — admin TUI

Deferred/out-of-scope discoveries from the implementation run
(2026-09-02). Entries are candidates for `doc/plan/issue/issue.md`.

# Admin TUI refuses a room that has history but no members

`openRoom` (crabswarm/chat/cli/tui/tui.go) decides room existence from
the roster listing, so once every member leaves, `chat admin tui --room R`
errors while `chat admin log R` still serves the retained transcript.
Deciding existence from the log instead needs a read the admin History
RPC does not offer today (an "any rows for this room?" probe or listing
rooms present in room_log).

# Admin verb inconsistency: `tui` takes --room, the others take a positional

Every other room-scoped admin verb (`log`, `send`, ...) takes the room as
its first positional argument; `chat admin tui` requires a `--room` flag
because the plan fixed that spelling. The group is internally
inconsistent; pick one convention.

# No shell completion for the TUI's room argument

The admin can already enumerate rooms and the completion precedent exists
(`completeChatMembers`, cmd/crabswarm/commands/zz_chat.go), but
`chat admin tui --room` completes nothing.

# e2e TUI screen scraping is hand-rolled

`charm.land/x/exp/teatest/v2` resolves as a module path but has no tagged
version, so e2e/crabswarm/chat_tui_test.go strips ANSI from accumulated
program output itself. Swap to teatest once it tags a release.

# WatchRoom upgrade path for the TUI tail poll

The TUI tails by cursor poll (~1s). The daemon already serves
server-streaming `ChatService.WatchRoom` (member plane); once an
admin-plane stream (or a message-appended event feeding one) exists, the
poll can be replaced without changing the tui package's LogReader
interface consumers.

# TUI conversation re-render is whole-string per poll

`layout()` (crabswarm/chat/cli/tui/model.go) re-renders the entire
conversation string (bounded at 2000 entries) on every poll that brings
entries. Fine at terminal scale; make it incremental if the screen ever
feels heavy.
