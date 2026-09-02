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
