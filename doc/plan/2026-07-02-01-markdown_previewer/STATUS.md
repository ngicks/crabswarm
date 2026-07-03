# STATUS — Markdown Previewer

Current state: **implemented (2026-07-02) — all 10 steps done, reviewed
(approve-with-nits; blocking race fixed), full suite green. Uncommitted
in the working tree.**

## Checklist (mirrors PLAN.md implementation steps)

- [x] 1. Proto `crabpreview/v1` + buf codegen (connect-go, protoc-gen-es TS)
      (TS target configured; generation deferred to step 8 when `web/` exists)
- [x] 2. `pkg/crabswarm/preview/config.go` + wire into `crabswarm.Config`
- [x] 3. `pkg/crabswarm/preview/render/` (goldmark + alerts + math, goldens)
      (alerts + mathjax vendored from go-grip v0.9.2, MIT attribution kept)
- [x] 4. `roots.go` + `tree.go` (in-memory store, all-files listing;
      `SafeJoin`/`ErrPathEscape` exported for httpapi reuse)
- [x] 5. `watch.go` (recursive fsnotify, debounce, broadcast hub; shared
      hub + per-root watcher, `RootsChanged` published by the service)
- [x] 6. `httpapi/` + `service.go` (connect handlers, WatchEvents, /raw, SPA;
      CSS served at /assets/css/{chroma-github,chroma-github-dark,alert}.css;
      `SetStaticFS` injection point added so httpapi never imports `web`)
- [x] 7. `daemon.go` + `client.go` (cmdman EnsureDaemon, connect client;
      verified against real cmdman v0.0.15: `inspect --format '{{.State}}'`,
      stale-name `rm` before `run`)
- [x] 8. `web/` frontend (Preact SPA, connect-web, embed + dev-FS,
         committed dist + CI freshness check). Integration follow-ups:
         chroma CSS link in SPA once httpapi paths are fixed; set mermaid
         NoScript in render.go; CI workflow unverified (no runner here).
- [x] 9. `cmd/crabswarm/commands/preview*.go` (cobra wiring; presentation
      in `pkg/crabswarm/cli/preview.go`; `--addr` on all four subcommands;
      daemon-backed shell completion for `preview remove`)
- [x] 9b. Integration follow-ups: mermaid `NoScript: true` in render.go,
      mathjax verified script-free, chroma/alert CSS linked in SPA theme
      effect, `web/dist` rebuilt.
- [x] 10. Polish: README, e2e (direct `__serve` in CI; cmdman path opt-in —
      both actually ran and passed locally, cmdman 0.0.15 on PATH)

## Done

- Plan scaffold written and go-grip stack verified from upstream source
  (goldmark v1.7.16, chroma v2, client-side mermaid/MathJax, WebSocket
  whole-page reload) — 2026-07-02.
- All 8 open questions resolved with the user; answers folded into PLAN.md
  and DECISION.md (cmdman daemonization, connect-go proto-first, math in
  v1, @tanstack/preact-query, 0.0.0.0 tailnet-first binding, dual-mode
  embed, in-memory roots, all-files tree with confirm-then-raw).
- D7 revised on user review (D7a): `web/dist` and `web/src/gen` are
  committed, not git-ignored — go:embed only sees files in the module zip,
  so `go install @<tag>` requires them in git. CI freshness check added.

## In progress

- (nothing)

## Blocked

- (nothing)

## Review outcome (2026-07-02)

- Multi-agent review: approve-with-nits. Blocking TOCTOU race in
  service.go (removeRoot vs startWatcher could leave a registered root
  with no watcher) — FIXED: store mutation + watcher lifecycle now one
  critical section; covered by concurrent add/remove storm test under
  `-race`. Also fixed: fs error detail leak to RPC clients, stale
  AddRoot proto comment (+ regen), README env-layering wording, crossed
  path-traversal test coverage on GetDocument and /raw.
- Full suite green: `go build/vet/test ./...` (incl. e2e, both cmdman
  and cmdman-free paths), `tsc --noEmit`, golangci-lint 0 issues on all
  new packages (49 pre-existing findings on main in untouched files).

## Deferred (open items)

- inotify watch-limit failures only log a server-side warning; no
  client/UI signal (needs a small design decision — proto change).
- No `-race` CI job for the preview package; CI workflow
  `.github/workflows/web.yml` (dist freshness) unverified — no runner
  available in this environment.
- Manual verification from PLAN.md (phone over tailnet, live-edit UX)
  not performed — needs the user's devices.

## Next action

- User: review + commit the working tree; try `crabswarm preview .`
  manually (desktop + phone over tailnet).
