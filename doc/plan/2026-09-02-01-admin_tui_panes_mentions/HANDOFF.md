# Handoff — deferred items found during implementation

## Team sends render as whole-room entries in history and the TUI

A `TeamTarget` send is stored with no recipient because the history
reader (`crabswarm/chat/history.go`, the `ToName != ""` branch) drops a
team-only recipient on read, and this plan does not change persistent
data. `crabswarm chat admin history` and the conversation pane therefore
show `admin → *` for a team send. Fix needs a team-only recipient row
plus a read-side change so the entry says `admin → beta/*`.

## Fresh worktrees need `pnpm install` in `web/` before `go generate ./api/...`

`protoc-gen-es` is resolved from the gitignored `web/node_modules`
(`api/buf.gen.yaml`), so a fresh worktree fails `buf generate` until
`pnpm install --frozen-lockfile` has run in `web/`. Document it or make
buf.gen.yaml resolve the plugin another way.

## `@` grammar edges left unpinned

Found by the final review, not changed: a `@token` on any line of a
multi-line draft addresses (the token ends at whitespace, newlines
included), so a quoted `@alpha/bob` on line 3 of a paragraph is the
target; punctuation-terminated mentions such as `@admin,` or `(@admin)`
are not highlighted because the token is compared whole; an unbalanced
backtick treats the rest of the text as a code span; and pasting the
TUI's `@alpha/ana` spelling into `chat admin send` argv parses as team
`@alpha`. Decide whether any of these deserve a rule.

## Tab completion mid-token keeps the tail

`token()` in `crabswarm/chat/cli/tui/completion.go` looks left of the
cursor only, so Tab on `@alx` with the cursor after `@al` yields
`@backend/alice x`. Shell-like, but nothing pins it; decide whether the
rest of the word should be replaced too.

## Issue backlog entries touched by this work

`doc/plan/issue/open/admin-tui-screen.md` says no TUI code exists under
`crabswarm/chat/` and lists the room picker as a follow-up; both are
now false. `team-fan-out-target-form-team-for-chat-send.md` keeps its
point for the member plane, but its premise (no team target anywhere)
is outdated: admin send has `team/*`. Only the user closes or rewrites
backlog items.

## `crabswarm/chat/notify` tests time out under load

`go test ./...` while other test runs shared the machine failed four
notify tests with `cmdman send-keys ...: signal: killed` (a 3 s
deadline on a fake cmdman). The package passes alone in 0.3 s and this
branch does not touch it; the deadline is load-sensitive.
