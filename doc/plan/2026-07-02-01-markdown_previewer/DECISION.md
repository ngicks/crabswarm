# DECISION LOG — Markdown Previewer

One entry per material decision: choice, rationale, rejected alternatives.
All resolved with the user on 2026-07-02.

## D0. Frontend stack (user-specified)

- **Choice**: Preact + preact-iso + @preact/signals + Ark UI (headless
  behavior) + daisyUI on Tailwind (theme/skin) + TanStack Query.
- **Rationale**: user's requested stack; small runtime, signals for UI
  state, headless components styled by daisyUI, query cache pairs with
  live-reload invalidation.
- **Rejected**: React (heavier); styling with only Ark recipes or only
  daisyUI static components (each covers half: behavior vs skin).

## D1. Server-side rendering with a single Go renderer

- **Choice**: markdown rendered to HTML on the Go server; the SPA receives
  `{html, toc, title}`.
- **Rationale**: one canonical renderer, chroma-quality highlighting, TOC
  anchors guaranteed consistent with body anchors, small JS bundle.
- **Rejected**: client-side unified/remark (duplicate renderer, bundle
  bloat); GitHub REST render API (network + token dependency — go-grip
  deliberately avoids it too).

## D2. (OQ1) Process model: cmdman daemonization

- **Choice**: `crabswarm preview [ROOT]` checks daemon presence via cmdman
  under a configurable name (`preview.daemon_name`, default
  `crabswarm-preview`); if absent, `cmdman run --name <name> -- crabswarm
  preview __serve` (hidden cobra command); wait for listen (healthz poll);
  then send the AddRoot RPC. This decision is what forces ConnectRPC (D4) —
  the CLI needs an RPC client anyway.
- **Rationale**: cmdman is the house daemon manager (same stance as the
  agent_swarming plan) — it owns daemonization, logs, restart, stop; no
  foreground terminal is tied up; one command still does everything.
- **Rejected**: foreground probe-or-serve (ties up a terminal, ad-hoc
  lifecycle); explicit `preview serve`+`preview add` (two commands);
  folding into `crabswarm serve` (couples hook IPC with interactive UI).

## D3. (OQ2) Renderer: goldmark, go-grip parity, math included in v1

- **Choice**: goldmark + `extension.GFM` + `extension.Footnote` +
  AutoHeadingID + goldmark-emoji + goldmark-highlighting/v2 (chroma) +
  goldmark-mermaid (client render); GitHub alerts by vendoring go-grip's
  MIT `pkg/alert`; **math ships in v1** via a MathJax-markup goldmark
  extension + client-side MathJax, mirroring go-grip's `pkg/mathjax` split.
- **Rationale**: verified as go-grip's actual stack (goldmark v1.7.16), so
  output parity comes from the same renderer family; user wants math from
  day one.
- **Rejected**: gomarkdown (less GitHub parity); GitHub API rendering;
  deferring math.

## D4. (OQ3) Browser API: connect-go, proto-first

- **Choice**: `PreviewService` defined in
  `pkg/api/schema/proto/crabpreview/v1`, served with connect-go, consumed
  with connect-web + protoc-gen-es v2 generated TS. Live reload is a
  `WatchEvents` server-streaming RPC. `/healthz` and `/raw/...` stay plain
  HTTP.
- **Rationale**: repo convention is proto-first; the cmdman flow (D2)
  already requires an RPC client in the CLI; generated TS types beat
  hand-mirrored JSON shapes.
- **Rejected**: plain JSON REST (second, schema-less surface); SSE for
  events (native reconnect, but a non-proto side channel — reconnect is
  instead hand-rolled in `web/src/api/events.ts` with
  invalidate-on-reconnect).

## D5. (OQ4) Data fetching: @tanstack/preact-query

- **Choice**: the official TanStack Query Preact adapter
  (https://tanstack.com/query/latest/docs/framework/preact/overview).
- **Rationale**: first-class adapter exists (user pointed it out —
  planner's compat-alias assumption was outdated); no compat shim needed.
- **Rejected**: `@tanstack/react-query` via preact/compat (kept as a
  drop-in fallback if the adapter misbehaves); signals + hand-rolled fetch
  (reimplements caching/invalidation that live reload leans on).

## D6. (OQ5) Network binding: 0.0.0.0 by default (tailnet-first)

- **Choice**: default `preview.addr` = `0.0.0.0:6419`; intended access from
  phones/tablets via Tailscale MagicDNS. Documented warning for untrusted
  networks (set loopback / firewall).
- **Rationale**: user's access path is the tailnet, where reachability is
  already scoped to their devices; loopback default would make the primary
  use case (phone in bed) need extra flags every time.
- **Rejected**: loopback default with LAN opt-in (friction for the main use
  case); shared-token auth (complexity not warranted inside a tailnet).

## D7. (OQ6) Build/embed: dual mode

- **Choice (superseded by D7a)**: git-ignored `web/dist` + build script +
  `go:embed` for release; a `dev` build tag / env switch serving from the
  source tree (or vite dev proxy) for iteration.
- **Rejected**: committing `dist/` (noisy diffs, stale-asset risk);
  embed-only (painful frontend iteration).

## D7a. (OQ6 revision, 2026-07-02) Commit `web/dist` (and `web/src/gen`)

- **Trigger**: user review — "The dist should not be git-ignored since it
  is embedded into the binary?"
- **Choice**: keep the dual mode (embed for release, dev-FS for iteration)
  but **commit `web/dist`** and the generated TS (`web/src/gen`). CI
  regenerates both and fails on `git diff --exit-code` so committed
  artifacts cannot go stale.
- **Rationale**: `go:embed` includes only files present in the module zip,
  which is derived from the git tag — a git-ignored `dist` makes
  `go install github.com/ngicks/crabswarm/cmd/crabswarm@<tag>` fail (embed
  pattern matches nothing). The user installs their tools exactly that way
  (mise go backend), so `go install` must work from a bare module with no
  node toolchain.
- **Rejected**: git-ignored dist + build script (breaks `go install`;
  clone-only workflow); GitHub-releases-only binary distribution (departs
  from how the user's tools are installed today).

## D8. (OQ7) Root list: in-memory only

- **Choice**: `RootStore` is process state; daemon restart starts empty;
  `crabswarm preview ROOT` re-adds (idempotent).
- **Rationale**: cmdman keeps the daemon long-lived anyway; no stale
  on-disk state to manage; re-adding is one short command.
- **Rejected**: XDG state-file persistence (stale/deleted-dir handling,
  extra config surface).

## D9. (OQ8) File tree: all files, confirm-then-raw for non-markdown

- **Choice**: tree lists all files (dirs first), go-grip-like. Markdown
  navigates in the SPA; clicking a non-markdown file pops a confirmation
  dialog, and on confirm the browser opens `/raw/{rootId}/{path}` with its
  default behavior (render image/PDF, download, ...).
- **Rationale**: truer to the nvim-tree/VSCode tree-view metaphor the user
  asked for; confirmation prevents accidental downloads on touch devices.
- **Rejected**: markdown-only tree (hides context); all-files without
  confirmation (accidental opens).

## D9a. (revision, 2026-07-02) Images render in-app

- **Trigger**: user request post-implementation ("Add image rendering").
- **Choice**: image files (png/jpg/jpeg/gif/webp/svg/avif/bmp/ico) clicked
  in the tree navigate in-SPA and render centered in the content area via
  `/raw` (`ImageView` beside `DocView`, no GetDocument call, TOC hidden).
  Other non-markdown files keep D9's confirm-then-raw dialog.
- **Also fixed** (same batch): root-absolute image refs in markdown
  (`![](/images/...)`, Zenn/GitHub convention) now rewrite to
  `/raw/{rootId}/...`; previously only relative srcs were rewritten and
  leading-slash srcs fell through to the SPA fallback.
